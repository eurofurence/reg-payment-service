package interaction

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	return s.store.GetTransactionsByFilter(ctx, query)
}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *entities.Transaction) (*entities.Transaction, error) {
	logger := logging.LoggerFromContext(ctx)
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
			logger.Error("error when calling the attendee service webhook. [error]: %v", err)
		}

		return tran, nil
	}

	return nil, apierrors.NewForbidden("unable to determine the request permissions")
}

func (s *serviceInteractor) UpdateTransaction(ctx context.Context, tran *entities.Transaction) error {
	mgr := NewIdentityManager(ctx)

	if !mgr.IsAdmin() && !mgr.IsAPITokenCall() {
		return apierrors.NewForbidden("no permission to update transaction")
	}

	res, err := s.store.GetTransactionsByFilter(ctx, entities.TransactionQuery{
		DebitorID:             tran.DebitorID,
		TransactionIdentifier: tran.TransactionID,
	})

	if err != nil {
		return err
	}

	if len(res) == 0 {
		return apierrors.NewNotFound(
			fmt.Sprintf("transaction %s for debitor %d could not be found", tran.TransactionID, tran.DebitorID),
		)
	}

	curTran := res[0]

	// check if transaction should be deleted or not
	if !reflect.ValueOf(tran.Deletion).IsZero() {
		// Within 3 calendar days of creation, for any transaction an admin may change
		// - status -> deleted
		const maxDaysForDeletion = 3.0
		days := time.Now().UTC().Sub(curTran.CreatedAt.UTC()).Hours() / 24.0

		if days > maxDaysForDeletion {
			return apierrors.NewForbidden("")
		}

		// update the deletion status with the current status that was
		// TODO move logic to database
		tran.Deletion.Status = curTran.TransactionStatus
		tran.TransactionStatus = entities.TransactionStatusDeleted

		return s.store.DeleteTransaction(ctx, *tran)
	}

	if tran.TransactionType == entities.TransactionTypeDue {
		return apierrors.NewForbidden("cannot change the transaction of type due")
	}

	requireHistorization := false

	// Status changes:
	//    (The previous status is always historized, see the history field)
	if tran.TransactionStatus != curTran.TransactionStatus {
		if !isValidStatusChange(curTran, *tran) {
			return apierrors.NewForbidden(
				fmt.Sprintf("cannot change status from %s to %s for transaction %s",
					curTran.TransactionStatus,
					tran.TransactionStatus,
					tran.TransactionID,
				))
		}

		requireHistorization = true
	}

	return s.store.UpdateTransaction(ctx, *tran, requireHistorization)
}

func (s *serviceInteractor) createTransactionWithElevatedAccess(
	ctx context.Context,
	tran *entities.Transaction,
	mgr *IdentityManager) (*entities.Transaction, error) {

	logger := logging.LoggerFromContext(ctx)

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

		// We first make sure that we successfully persisted the transaction
		// in the DB before requesting a payment link if applicable
		err = s.store.CreateTransaction(ctx, *tran)
		if err != nil {
			return nil, err
		}

		// create payment link if
		// transaction_type=payment, method=credit, status=tentative
		if shouldRequestPaymentLink(tran) {

			paymentLink, err := s.createPaymentLink(ctx, *tran)
			if err != nil {
				return nil, apierrors.NewInternalServerError(err.Error())
			}

			tran.PaymentStartUrl = paymentLink

			// update the transaction and insert the payment link,
			// which was provided by the adapter service
			if err := s.store.UpdateTransaction(ctx, *tran, true); err != nil {
				return nil, err
			}
		}

		if err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID)); err != nil {
			// If the webhook was not successful, we write an error log and do not
			// return a 500 error response
			logger.Error("error when calling attendee service webhook. [error]: %v", err)
		}

		return tran, nil
	} else {
		// create new due transaction - must be created in status valid
		tran.TransactionStatus = entities.TransactionStatusValid
		err := s.store.CreateTransaction(ctx, *tran)
		return tran, err
	}
}

func (s *serviceInteractor) validateAttendeeTransaction(ctx context.Context, newTransaction *entities.Transaction) error {
	logger := logging.LoggerFromContext(ctx)
	debitorIDs, err := s.attendeeClient.ListMyRegistrationIds(ctx)
	if err != nil {
		return err
	}

	if !containsDebitor(debitorIDs, newTransaction.DebitorID) {
		return apierrors.NewForbidden(fmt.Sprintf("transactions for debitorID %d may not be altered", newTransaction.DebitorID))
	}

	// User may only create transactions which are valid for requesting payment links
	if !shouldRequestPaymentLink(newTransaction) {
		return apierrors.NewForbidden("transaction is not eligible for requesting a payment link")
	}

	// Check if there are any pending transactions.
	pending, err := s.arePendingPaymentsPresent(ctx, newTransaction.DebitorID)

	if err != nil {
		logger.Error("could not retrieve pending payments for debitor %d - [error]: %v", newTransaction.DebitorID, err)
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
	if !s.isValidPayment(currentTransactions, newTransaction, logger) {
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

func (s *serviceInteractor) isValidPayment(curTransactions []entities.Transaction, newTran *entities.Transaction, logger logging.Logger) bool {
	var allDues int64
	var allPayments int64

	for _, t := range curTransactions {
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
		logger.Info("No outstanding dues for attendee %d", newTran.DebitorID)
		return false
	}

	remaining := allDues - allPayments

	if remaining < 0 || newTran.Amount.GrossCent != remaining {
		// we do not allow partial payments from attendees
		// Admins or s2s calls will not use this validation logic
		logger.Info("rejected partial payment for attendee %d", newTran.DebitorID)
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

func shouldRequestPaymentLink(tran *entities.Transaction) bool {
	// Only the following condition is valid at the time,
	// in order to generate a payment link
	//
	// transaction_type=payment, method=credit, status=tentative
	return tran.TransactionType == entities.TransactionTypePayment &&
		tran.PaymentMethod == entities.PaymentMethodCredit &&
		tran.TransactionStatus == entities.TransactionStatusTentative
}

func isValidStatusChange(curTran, tran entities.Transaction) bool {
	// The only possible change is status for payments
	// * tentative -> pending (payment link has been used)
	// * tentative -> deleted (payment link has been deleted)
	// * pending -> valid (payment is confirmed by admin or by payment provider)
	// * pending -> deleted (payment has been deemed in error)

	if curTran.TransactionStatus == entities.TransactionStatusTentative {
		switch tran.TransactionStatus {
		case entities.TransactionStatusPending, entities.TransactionStatusDeleted:
			return true
		}

		return false
	}

	if curTran.TransactionStatus == entities.TransactionStatusPending {
		switch tran.TransactionStatus {
		case entities.TransactionStatusValid, entities.TransactionStatusDeleted:
			return true
		}
	}

	return false

}
