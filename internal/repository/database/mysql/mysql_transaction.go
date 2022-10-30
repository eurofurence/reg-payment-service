package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/domain"
	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

var allowedFields = []string{"TransactionStatusID", "Comment", "Deletion", "EffectiveDate", "DueDate"}

var (
	// ErrTransactionExists is returned when a transaction already exists in the database
	ErrTransactionExists = errors.New("the transaction already exists")
)

func (m *mysqlConnector) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	// Transactions of type due can only be created once.
	if tr.TransactionTypeID == uint(domain.Due) {
		var exists int64
		m.db.WithContext(tCtx).Model(&entities.Transaction{}).
			Where(&entities.Transaction{
				TransactionID:     tr.TransactionID,
				TransactionTypeID: uint(domain.Due),
			}).
			Count(&exists)
		if exists > 0 {
			return ErrTransactionExists
		}
	}

	result := m.db.WithContext(tCtx).Create(&tr)

	if result.Error != nil {
		return result.Error
	}

	return m.CreateTransactionLog(ctx, tr.ToTransactionLog())
}

func (m *mysqlConnector) UpdateTransaction(ctx context.Context, tr entities.Transaction) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := m.db.WithContext(tCtx).
		Model(&tr).
		Select(allowedFields).
		Where(&entities.Transaction{TransactionID: tr.TransactionID}).
		Updates(tr)

	if res.Error != nil {
		return res.Error
	}

	res = m.db.WithContext(tCtx).
		Where(&entities.Transaction{
			TransactionID:     tr.TransactionID,
			TransactionTypeID: tr.TransactionTypeID,
		}).
		First(&tr)
	if res.Error != nil {
		return res.Error
	}

	return m.CreateTransactionLog(ctx, tr.ToTransactionLog())
}

func (m *mysqlConnector) GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType uint) (*entities.Transaction, error) {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	var tr entities.Transaction
	res := m.db.WithContext(tCtx).Where(&entities.Transaction{TransactionID: transactionID}).First(&tr)

	if res.Error != nil {
		return nil, res.Error
	}

	return &tr, nil
}

func (m *mysqlConnector) GetTransactionsByDebitorID(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	var transactions []entities.Transaction

	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := m.db.WithContext(tCtx).
		Where(&entities.Transaction{DebitorID: debitorID}).
		Find(&transactions)
	if res.Error != nil {
		return nil, res.Error
	}

	return transactions, nil
}
