package inmemory

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

func (m *inmemoryProvider) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	return nil
}

func (m *inmemoryProvider) UpdateTransaction(ctx context.Context, tr entities.Transaction) error {
	return nil
}

func (m *inmemoryProvider) GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType uint) (*entities.Transaction, error) {
	return &entities.Transaction{}, nil
}

func (m *inmemoryProvider) GetTransactionsByFilter(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	return make([]entities.Transaction, 0), nil
}
