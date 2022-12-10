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

func (m *inmemoryProvider) GetTransactionsByFilter(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	result := make([]entities.Transaction, 0)
	for _, t := range m.transactions {
		if t.DebitorID == debitorID {
			result = append(result, t)
		}
	}
	return result, nil
}
