package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

var (
	// ErrTransactionExists is returned when a transaction already exists in the database
	ErrTransactionExists = errors.New("the transaction already exists")
)

func (m *mysqlConnector) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	var exists int64
	m.db.WithContext(tCtx).Model(&entities.Transaction{}).Where(&entities.Transaction{TransactionID: tr.TransactionID}).Count(&exists)
	if exists > 0 {
		return ErrTransactionExists

	}

	result := m.db.WithContext(tCtx).Create(&tr)

	if result.Error != nil {
		return result.Error
	}

	return m.CreateTransactionLog(ctx, tr.ToTransactionLog())
}

func (m *mysqlConnector) GetTransactionByID(ctx context.Context, id int) (*entities.Transaction, error) {
	return nil, nil
}

func (m *mysqlConnector) UpdateTransaction(ctx context.Context, tr entities.Transaction) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := m.db.WithContext(tCtx).
		Model(&entities.Transaction{}).
		Omit("DebitorID").
		Where(&entities.Transaction{TransactionID: tr.TransactionID}).
		Updates(&tr)

	if res.Error != nil {
		return res.Error
	}

	return m.CreateTransactionLog(ctx, tr.ToTransactionLog())
}
