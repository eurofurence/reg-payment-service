package interaction

import (
	"context"
	"errors"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/domain"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
)

var _ Interactor = (*serviceInteractor)(nil)

type Interactor interface {
	GetTransactionsForDebitor(ctx context.Context, debitorID string) ([]domain.Transaction, error)
	CreateTransactionForDebitor(ctx context.Context, debitorID string, tran *domain.Transaction) error
}

type serviceInteractor struct {
	store database.Repository
}

func NewServiceInteractor(r database.Repository) (Interactor, error) {
	if r == nil {
		return nil, errors.New("repository must not be nil")
	}

	return &serviceInteractor{
		store: r,
	}, nil
}

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, debitorID string) ([]domain.Transaction, error) {
	return []domain.Transaction{
		{
			ID:        "1",
			DebitorID: debitorID,
			Type:      domain.Payment,
			Method:    domain.Credit,
			Amount: domain.Amount{
				Currency:  "EUR",
				GrossCent: 190_00,
				VatRate:   19.0,
			},
			Comment:       "Fun Fun Fun",
			Status:        domain.Tentative,
			EffectiveDate: time.Now(),
			DueDate:       time.Now().AddDate(0, 0, 1),
			Deletion:      nil,
		},
	}, nil
}

func (s *serviceInteractor) CreateTransactionForDebitor(ctx context.Context, debitorID string, tran *domain.Transaction) error {
	panic("Not implemented")
}
