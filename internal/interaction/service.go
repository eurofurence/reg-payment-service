package interaction

import (
	"context"
	"errors"

	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
)

var _ Interactor = (*serviceInteractor)(nil)

type Interactor interface {
	GetTransactionsForDebitor(ctx context.Context, debitorID int64) ([]entities.Transaction, error)
	CreateTransaction(ctx context.Context, tran *entities.Transaction) error
}

type serviceInteractor struct {
	logger logging.Logger
	store  database.Repository
}

func NewServiceInteractor(r database.Repository, logger logging.Logger) (Interactor, error) {
	if r == nil {
		return nil, errors.New("repository must not be nil")
	}

	return &serviceInteractor{
		logger: logger,
		store:  r,
	}, nil
}
