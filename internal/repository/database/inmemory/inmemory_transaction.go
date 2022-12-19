package inmemory

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func (m *inmemoryProvider) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	if tr.ID != 0 {
		return errors.New("create needs a new transaction")
	}
	tr.ID = uint(atomic.AddUint32(&m.idSequence, 1))
	m.transactions[tr.ID] = tr
	return nil
}

func (m *inmemoryProvider) UpdateTransaction(ctx context.Context, tr entities.Transaction) error {
	if tr.ID == 0 {
		return errors.New("cannot update a new transaction")
	}
	_, ok := m.transactions[tr.ID]
	if !ok {
		return errors.New("transaction not found in database")
	}
	m.transactions[tr.ID] = tr
	return nil
}

func (m *inmemoryProvider) GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType entities.TransactionType) (*entities.Transaction, error) {
	for _, t := range m.transactions {
		if t.TransactionID == transactionID && t.TransactionType == tType {
			copy := t
			return &copy, nil
		}
	}
	return &entities.Transaction{}, errors.New("no matching transaction in database")
}

func (m *inmemoryProvider) GetTransactionsByFilter(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	result := make([]entities.Transaction, 0)
	for _, t := range m.transactions {
		if query.DebitorID != 0 && t.DebitorID != query.DebitorID {
			continue
		}
		if query.TransactionIdentifier != "" && t.TransactionID != query.TransactionIdentifier {
			continue
		}

		if !query.EffectiveFrom.IsZero() && query.EffectiveFrom.After(t.EffectiveDate.Time) {
			continue
		}
		if !query.EffectiveBefore.IsZero() && !t.EffectiveDate.Time.Before(query.EffectiveBefore) {
			// if !(20 < 28) break
			continue
		}
		result = append(result, t)
	}
	return result, nil
}

func (m *inmemoryProvider) GetValidTransactionsForDebitor(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	result := make([]entities.Transaction, 0)
	for _, t := range m.transactions {
		if t.DebitorID == debitorID && t.TransactionStatus == entities.TransactionStatusValid {
			result = append(result, t)
		}
	}

	return result, nil
}

func (m *inmemoryProvider) QueryOutstandingDuesForDebitor(ctx context.Context, debutorID int64) (int64, error) {
	panic("not implemented") // TODO: Implement
}
