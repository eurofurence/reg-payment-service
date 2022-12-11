package interaction

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

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
			s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID))
		}

		return tran, err
	}

	if mgr.IsRegisteredUser() {
		if err := s.validateAttendeeTransaction(ctx, tran); err != nil {
			// 403
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

	return nil, errors.New("400")
}

func (s *serviceInteractor) validateAttendeeTransaction(ctx context.Context, tran *entities.Transaction) error {
	debitorIDs, err := s.attendeeClient.ListMyRegistrationIds(ctx)
	if err != nil {
		return err
	}

	if !containsDebitor(debitorIDs, tran.DebitorID) {
		return fmt.Errorf("transactions for debitorID %d may not be altered", tran.DebitorID)
	}

	if tran.TransactionType != entities.TransactionTypePayment ||
		tran.TransactionStatus != entities.TransactionStatusTentative ||
		tran.PaymentMethod != entities.PaymentMethodCredit {

		// only payment transactions
		// only status tentative
		// only method credit
		// -> Status 403

		return errors.New("transaction is not in a valid state")
	}

	// TODO: Continue here

	// check if outstanding dues
	// sum all status valid due transactions
	// subtract all valid payments
	// where debitorID == currentDebID
	// -> current_dues
	// in error case: 400

	// don't allow partial payments
	// Amount Grosscent == sum(current_dues)

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
