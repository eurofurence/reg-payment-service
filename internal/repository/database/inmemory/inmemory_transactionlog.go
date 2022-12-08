package inmemory

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

func (m *inmemoryProvider) CreateTransactionLog(ctx context.Context, tl entities.TransactionLog) error {
	if tl.ID != 0 {
		return errors.New("create needs a new transaction log entry")
	}
	tl.ID = uint(atomic.AddUint32(&m.idSequence, 1))
	m.transactionLogs[tl.ID] = tl
	return nil
}

func (m *inmemoryProvider) GetTransactionLogByID(ctx context.Context, id uint) (*entities.TransactionLog, error) {
	tl, ok := m.transactionLogs[id]
	if !ok {
		return &entities.TransactionLog{}, errors.New("no such transaction log in database")
	}
	return &tl, nil
}
