package interaction

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	return s.store.GetTransactionsByFilter(ctx, query)
}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *entities.Transaction) (*entities.Transaction, error) {
	appConfig, err := config.GetApplicationConfig()
	if err != nil {
		return nil, err
	}

	// check if currency is allowed
	if !isCurrencyAllowed(appConfig.Service.AllowedCurrencies, tran.Amount.ISOCurrency) {
		return nil, apierrors.NewForbidden(fmt.Sprintf("invalid currency %s provided", tran.Amount.ISOCurrency))
	}

	// generate a transaction ID if none exists
	if tran.TransactionID == "" {
		id, err := generateTransactionID(appConfig.Service.TransactionIDPrefix, tran)
		if err != nil {
			return nil, err
		}

		tran.TransactionID = id
	}

	mgr := NewIdentityManager(ctx)
	if mgr.IsAdmin() || mgr.IsAPITokenCall() {
		return s.createTransactionWithElevatedAccess(ctx, tran, mgr)
	}

	if mgr.IsRegisteredUser() {
		// check if attendee is permitted to create this transaction
		if err := s.validateAttendeeTransaction(ctx, tran); err != nil {
			return nil, err
		}

		// TODO new order
		// 1. create transaction
		// 2. create paymentlink
		// 3. update transaction with paymentlink

		// generate a payment link
		paymentLink, err := s.createPaymentLink(ctx, *tran)

		if err != nil {
			return nil, apierrors.NewInternalServerError(err.Error())
		}

		tran.PaymentStartUrl = paymentLink
		// What do we do with response.ReferenceId?

		// create a transaction in the database
		if err := s.store.CreateTransaction(ctx, *tran); err != nil {
			return nil, err
		}

		// inform the attendee service that there is a new payment in the database
		if err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID)); err != nil {
			return nil, err
		}

		return tran, nil
	}

	return nil, apierrors.NewForbidden("unable to determine the request permissions")
}

func (s *serviceInteractor) createTransactionWithElevatedAccess(
	ctx context.Context,
	tran *entities.Transaction,
	mgr *IdentityManager) (*entities.Transaction, error) {

	if mgr.IsAdmin() && tran.TransactionType == entities.TransactionTypeDue {
		return nil, apierrors.NewForbidden("Admin role is not allowed to create transactions of type due")
	}

	if tran.TransactionType == entities.TransactionTypePayment {
		pending, err := s.arePendingPaymentsPresent(ctx, tran.DebitorID)
		if err != nil {
			return nil, err
		}
		if pending {
			return nil, apierrors.NewConflict(fmt.Sprintf("There are pending payments for attendee %d", tran.DebitorID))
		}

		// TODO new order
		// 1. create transaction
		// 2. create paymentlink
		// 3. update transaction with paymentlink

		// create payment link if
		// transaction_type=payment, method=credit, status=tentative
		if tran.TransactionType == entities.TransactionTypePayment &&
			tran.PaymentMethod == entities.PaymentMethodCredit &&
			tran.TransactionStatus == entities.TransactionStatusTentative {

			paymentLink, err := s.createPaymentLink(ctx, *tran)
			if err != nil {
				return nil, apierrors.NewInternalServerError(err.Error())
			}

			tran.PaymentStartUrl = paymentLink
		}

		err = s.store.CreateTransaction(ctx, *tran)
		if err != nil {
			return nil, err
		}

		err = s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID))
		if err != nil {
			return nil, err
		}

		return tran, nil
	} else {
		// create new due transaction
		err := s.store.CreateTransaction(ctx, *tran)
		return tran, err
	}
}

func (s *serviceInteractor) validateAttendeeTransaction(ctx context.Context, newTransaction *entities.Transaction) error {
	debitorIDs, err := s.attendeeClient.ListMyRegistrationIds(ctx)
	if err != nil {
		return err
	}

	if !containsDebitor(debitorIDs, newTransaction.DebitorID) {
		return apierrors.NewForbidden(fmt.Sprintf("transactions for debitorID %d may not be altered", newTransaction.DebitorID))
	}

	if newTransaction.TransactionType != entities.TransactionTypePayment ||
		newTransaction.TransactionStatus != entities.TransactionStatusTentative ||
		newTransaction.PaymentMethod != entities.PaymentMethodCredit {

		// only payment transactions
		// only status tentative
		// only method credit
		// -> Status 403
		return apierrors.NewForbidden("transaction is not in a valid state")
	}

	// Check if there are any pending transactions.
	pending, err := s.arePendingPaymentsPresent(ctx, newTransaction.DebitorID)

	if err != nil {
		s.logger.Error("could not retrieve pending payments for debitor %d - [error]: %v", newTransaction.DebitorID, err)
		return err
	}

	// do not create a new transaction if there is a pending payment.
	if pending {
		return apierrors.NewConflict(fmt.Sprintf("There are pending payments for attendee %d", newTransaction.DebitorID))
	}

	// We defined, that we only query transactions in status valid.
	currentTransactions, err := s.store.GetValidTransactionsForDebitor(ctx, newTransaction.DebitorID)
	if err != nil {
		return err
	}

	// in error case: 400
	// if partial payment || no outstanding dues
	if !s.isValidPayment(currentTransactions, newTransaction) {
		return apierrors.NewBadRequest("no outstanding dues or partial payment")
	}

	return nil
}

func containsDebitor(debIDs []int64, debID int64) bool {
	for _, id := range debIDs {
		if id == debID {
			return true
		}
	}

	return false
}

func generateTransactionID(prefix string, tran *entities.Transaction) (string, error) {

	parsedTime := time.Now().UTC().Format("0102-150405")
	return fmt.Sprintf("%s-%06d-%s-%s", prefix, tran.DebitorID, parsedTime, randomDigits(4)), nil

}

var digitRunes = []rune("0123456789")

func randomDigits(count int) string {
	if count < 0 {
		return ""
	}

	res := make([]rune, count)

	for i := 0; i < count; i++ {
		rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(digitRunes))))
		if err != nil {
			return ""
		}

		res[i] = digitRunes[rnd.Int64()]

	}

	return string(res)
}

func (s *serviceInteractor) arePendingPaymentsPresent(ctx context.Context, debitorID int64) (bool, error) {
	transactions, err := s.store.GetTransactionsByFilter(ctx, entities.TransactionQuery{DebitorID: debitorID})
	if err != nil {
		return false, err
	}

	// check if there are any existing transactions of type payment, and return if they are
	// in pending or tentative state
	for _, tt := range transactions {
		switch tt.TransactionStatus {
		case entities.TransactionStatusPending, entities.TransactionStatusTentative:
			if tt.TransactionType == entities.TransactionTypePayment {
				return true, nil
			}
		}
	}

	// no pending payment transactions
	return false, nil
}

func (s *serviceInteractor) isValidPayment(curTransactions []entities.Transaction, newTran *entities.Transaction) bool {
	var allDues int64
	var allPayments int64

	for _, t := range curTransactions {
		// Do we need a currency check here?
		// if t.Amount.ISOCurrency != newTran.Amount.ISOCurrency {
		//  s.logger.Error(...)
		// 	return false
		// }

		if t.TransactionType == entities.TransactionTypeDue {
			allDues += t.Amount.GrossCent
		} else if t.TransactionType == entities.TransactionTypePayment {
			allPayments += t.Amount.GrossCent
		}
	}

	// check if there are any outstanding dues
	// sum all status valid due transactions
	// subtract all valid payments
	// -> current_dues

	// can we have negative dues if we owe an attendee money?
	if allDues <= 0 {
		s.logger.Info("No outstanding dues for attendee %d", newTran.DebitorID)
		return false
	}

	remaining := allDues - allPayments

	if remaining < 0 || newTran.Amount.GrossCent != remaining {
		// we do not allow partial payments from attendees
		// Admins or s2s calls will not use this validation logic
		s.logger.Info("rejected partial payment for attendee %d", newTran.DebitorID)
		return false
	}

	return true
}

func (s *serviceInteractor) createPaymentLink(ctx context.Context, tran entities.Transaction) (string, error) {
	response, err := s.cncrdClient.CreatePaylink(ctx, cncrdadapter.PaymentLinkRequestDto{
		DebitorId: tran.DebitorID,
		Currency:  tran.Amount.ISOCurrency,
		VatRate:   tran.Amount.VatRate,
		AmountDue: tran.Amount.GrossCent,
	})

	if err != nil {
		return "", apierrors.NewInternalServerError(err.Error())
	}

	return response.Link, nil
}

func isCurrencyAllowed(allowedCurrencies []string, isoCurrency string) bool {
	for _, cur := range allowedCurrencies {
		if strings.EqualFold(cur, isoCurrency) {
			return true
		}
	}

	return false
}
