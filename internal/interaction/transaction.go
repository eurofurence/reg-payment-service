package interaction

import (
	"context"
	"crypto/rand"
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

	// TODO: implement identity management
	// TODO: call attendee service if needed

	if tran.TransactionID == "" {
		id, err := generateTransactionID(tran)
		if err != nil {
			return nil, err
		}

		tran.TransactionID = id
	}

	err := s.store.CreateTransaction(ctx, *tran)

	// TODO: call downstream cncrd adapter service

	return tran, err
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
