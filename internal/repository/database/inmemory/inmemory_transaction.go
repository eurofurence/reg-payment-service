package inmemory

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

func (m *inmemoryProvider) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	panic("not implemented") // TODO: Implement
}

func (m *inmemoryProvider) GetTransactionByID(ctx context.Context, id int) (*entities.Transaction, error) {
	panic("not implemented") // TODO: Implement
}

func (m *inmemoryProvider) UpdateTransaction(ctx context.Context, tr entities.Transaction) error {
	panic("not implemented") // TODO: Implement
}
