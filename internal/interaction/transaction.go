package interaction

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	return s.store.GetTransactionsByFilter(ctx, query)
}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *entities.Transaction) error {
	return nil
}
