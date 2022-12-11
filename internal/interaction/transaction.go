package interaction

import (
	"context"
	"fmt"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	return s.store.GetTransactionsByFilter(ctx, query)
}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *entities.Transaction) (*entities.Transaction, error) {
	// TODO: call downstream service

	if tran.TransactionID == "" {
		id, err := generateTransactionID(tran)
		if err != nil {
			return nil, err
		}

		tran.TransactionID = id
	}
	err := s.store.CreateTransaction(ctx, *tran)

	return tran, err
}

func generateTransactionID(tran *entities.Transaction) (string, error) {
	appConfig, err := config.GetApplicationConfig()
	if err != nil {
		return "", err
	}
	srvConf := appConfig.Service

	// TODO generate transaction ID
	return fmt.Sprintf("%s-%d-%s-%d", srvConf.TransactionIDPrefix, tran.DebitorID), nil

	//{prefix-from-config}-NNNNNN-MMDD-HHMMSS-RRRR

}
