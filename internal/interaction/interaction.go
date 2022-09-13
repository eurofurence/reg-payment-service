package interaction

import (
	"context"
)

var _ Interactor = (*serviceInteractor)(nil)

// TODO put in domain package?
type Transaction struct{}

type Interactor interface {
	GetTransactionsForDebitor(ctx context.Context, debitorID int) ([]Transaction, error)
	CreateTransactionForDebitor(ctx context.Context, debitorID int, tran *Transaction) error
}

type serviceInteractor struct {
}

func NewServiceInteractor() Interactor {
	// TODO add database and cross service handler
	return &serviceInteractor{}
}

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, debitorID int) ([]Transaction, error) {
	panic("Not implemented")
}

func (s *serviceInteractor) CreateTransactionForDebitor(ctx context.Context, debitorID int, tran *Transaction) error {
	panic("Not implemented")
}
