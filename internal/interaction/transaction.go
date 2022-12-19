package interaction

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	return s.store.GetTransactionsByFilter(ctx, query)
}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *entities.Transaction) (*entities.Transaction, error) {
	mgr := NewIdentityManager(ctx)
	if mgr.IsAdmin() || mgr.IsAPITokenCall() {
		if tran.TransactionID == "" {
			id, err := generateTransactionID(tran)
			if err != nil {
				return nil, err
			}

			tran.TransactionID = id
		}

		err := s.store.CreateTransaction(ctx, *tran)
		if tran.TransactionType == entities.TransactionTypePayment {
			err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID))
			if err != nil {
				return nil, err
			}
		}

		return tran, err
	}

	if mgr.IsRegisteredUser() {
		if err := s.validateAttendeeTransaction(ctx, tran); err != nil {
			return nil, err
		}

		if err := s.store.CreateTransaction(ctx, *tran); err != nil {
			return nil, err
		}

		if err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID)); err != nil {
			return nil, err
		}

		return tran, nil

	}

	return nil, apierrors.NewForbidden("unable to determine the request permissions")
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

	// We defined, that we only query transactions in status valid.
	// What if we have tentative transactions of type payment here?
	// Then we would create transactions, where we exceed the due amount.
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

func generateTransactionID(tran *entities.Transaction) (string, error) {
	appConfig, err := config.GetApplicationConfig()
	if err != nil {
		return "", err
	}

	parsedTime := time.Now().UTC().Format("0102-150405")
	return fmt.Sprintf("%s-%06d-%s-%s", appConfig.Service.TransactionIDPrefix, tran.DebitorID, parsedTime, randomDigits(4)), nil

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
