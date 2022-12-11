package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/entities"
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
	// if tr.TransactionType == entities.TransactionTypeDue {
	// 	var exists int64
	// 	m.db.WithContext(tCtx).Model(&entities.Transaction{}).
	// 		Where(&entities.Transaction{
	// 			TransactionID:   tr.TransactionID,
	// 			TransactionType: entities.TransactionTypeDue,
	// 		}).
	// 		Count(&exists)
	// 	if exists > 0 {
	// 		return ErrTransactionExists
	// 	}
	// }

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
			TransactionID:   tr.TransactionID,
			TransactionType: tr.TransactionType,
		}).
		First(&tr)
	if res.Error != nil {
		return res.Error
	}

	return m.CreateTransactionLog(ctx, tr.ToTransactionLog())
}

func (m *mysqlConnector) GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType entities.TransactionType) (*entities.Transaction, error) {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	var tr entities.Transaction
	res := m.db.WithContext(tCtx).Where(&entities.Transaction{
		TransactionID:   transactionID,
		TransactionType: tType,
	}).First(&tr)

	if res.Error != nil {
		return nil, res.Error
	}

	return &tr, nil
}

func (m *mysqlConnector) GetTransactionsByFilter(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	var transactions []entities.Transaction

	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	db := m.db.WithContext(tCtx).
		Where(&entities.Transaction{
			DebitorID:     query.DebitorID,
			TransactionID: query.TransactionIdentifier,
		})

	if !query.EffectiveFrom.IsZero() {
		db.Where("effective_date >= ?", query.EffectiveFrom)
	}

	if !query.EffectiveBefore.IsZero() {
		db.Where("effective_date < ?", query.EffectiveBefore)
	}

	res := db.Find(&transactions)
	if res.Error != nil {
		return nil, res.Error
	}

	return transactions, nil
}
