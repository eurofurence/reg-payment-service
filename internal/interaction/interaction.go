package interaction

import (
	"context"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/domain"
)

var _ Interactor = (*serviceInteractor)(nil)

type Interactor interface {
	GetTransactionsForDebitor(ctx context.Context, debitorID string) ([]domain.Transaction, error)
	CreateTransactionForDebitor(ctx context.Context, debitorID string, tran *domain.Transaction) error
}

type serviceInteractor struct {
}

func NewServiceInteractor() Interactor {
	// TODO add database and cross service handler
	return &serviceInteractor{}
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
