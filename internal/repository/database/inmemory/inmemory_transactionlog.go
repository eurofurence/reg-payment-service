package inmemory

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

func (m *inmemoryProvider) CreateTransactionLog(ctx context.Context, tl entities.TransactionLog) error {
	return nil
}

func (m *inmemoryProvider) GetTransactionLogByID(ctx context.Context, id int) (*entities.TransactionLog, error) {
	return nil, nil
}
