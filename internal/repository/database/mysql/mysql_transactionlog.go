package mysql

import (
	"context"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func (m *mysqlConnector) CreateTransactionLog(ctx context.Context, tl entities.TransactionLog) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := m.db.WithContext(tCtx).Create(&tl)
	return res.Error
}

func (m *mysqlConnector) GetTransactionLogByID(ctx context.Context, id uint) (*entities.TransactionLog, error) {
	// TODO evaluate if needed
	return nil, nil
}
