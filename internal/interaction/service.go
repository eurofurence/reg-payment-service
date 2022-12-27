package interaction

import (
	"context"
	"errors"

	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/attendeeservice"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
)

var _ Interactor = (*serviceInteractor)(nil)

type Interactor interface {
	GetTransactionsForDebitor(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error)
	CreateTransaction(ctx context.Context, tran *entities.Transaction) (*entities.Transaction, error)
	CreateTransactionForOutstandingDues(ctx context.Context, debitorID int64) (*entities.Transaction, error)
	UpdateTransaction(ctx context.Context, tran *entities.Transaction) error
}

type serviceInteractor struct {
	store          database.Repository
	attendeeClient attendeeservice.AttendeeService
	cncrdClient    cncrdadapter.CncrdAdapter
}

func NewServiceInteractor(r database.Repository,
	attClient attendeeservice.AttendeeService,
	ccClient cncrdadapter.CncrdAdapter,
	logger logging.Logger,
) (Interactor, error) {

	if r == nil {
		return nil, errors.New("repository must not be nil")
	}

	if attClient == nil {
		return nil, errors.New("no attendee service client provided")
	}

	if ccClient == nil {
		return nil, errors.New("cncrd adapter client provided")
	}

	return &serviceInteractor{
		store:          r,
		attendeeClient: attClient,
		cncrdClient:    ccClient,
	}, nil
}
