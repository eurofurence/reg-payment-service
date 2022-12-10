package interaction

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	return s.store.GetTransactionsByFilter(ctx, debitorID)
}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *entities.Transaction) error {
	return nil
}
