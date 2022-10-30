package interaction

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/domain"
	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, debitorID int64) ([]domain.Transaction, error) {
	eTran, err := s.store.GetTransactionsByDebitorID(ctx, debitorID)
	if err != nil {
		return nil, err
	}

	return toDomainTransactions(eTran), nil

}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *domain.Transaction) error {
	return nil
}

func toDomainTransactions(tran []entities.Transaction) []domain.Transaction {
	res := make([]domain.Transaction, len(tran))
	for i, v := range tran {
		res[i] = toDomainTransaction(v)
	}

	return res
}

func toDomainTransaction(tr entities.Transaction) domain.Transaction {
	dtr := domain.Transaction{
		ID:        tr.TransactionID,
		DebitorID: tr.DebitorID,
		Type:      domain.TransactionType(tr.TransactionTypeID),
		Method:    domain.PaymentMethod(tr.PaymentMethodID),
		Amount: domain.Amount{
			Currency:  tr.Amount.ISOCurrency,
			GrossCent: tr.Amount.GrossCent,
			VatRate:   tr.Amount.VatRate,
		},
		Comment: tr.Comment,
		Status:  domain.TransactionStatus(tr.TransactionStatusID),
	}

	if tr.EffectiveDate.Valid {
		dtr.EffectiveDate = tr.EffectiveDate.Time
	}

	if tr.DueDate.Valid {
		dtr.DueDate = tr.DueDate.Time
	}

	if tr.DeletedAt.Valid {
		dtr.Deletion = &domain.Deletion{
			PreviousStatus: domain.TransactionStatus(tr.Deletion.TransactionStatusID),
			Comment:        tr.Deletion.Comment,
			DeletedBy:      tr.Deletion.By,
			Date:           tr.DeletedAt.Time,
		}
	}

	return dtr

}
