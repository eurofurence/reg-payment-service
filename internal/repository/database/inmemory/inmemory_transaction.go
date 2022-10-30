package inmemory

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

func (m *inmemoryProvider) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	panic("not implemented") // TODO: Implement
}

func (m *inmemoryProvider) UpdateTransaction(ctx context.Context, tr entities.Transaction) error {
	panic("not implemented") // TODO: Implement
}

func (m *inmemoryProvider) GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType uint) (*entities.Transaction, error) {
	panic("not implemented") // TODO: Implement
}

func (m *inmemoryProvider) GetTransactionsByDebitorID(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	return nil, nil
}
