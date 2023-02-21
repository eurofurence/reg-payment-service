package mysql

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

var allowedFields = []string{
	"Amount",
	"TransactionStatus",
	"Comment",
	// TODO Missing payment processor information,
	"PaymentStartUrl",
	"EffectiveDate",
	"DueDate",
}

var (
	// ErrTransactionExists is returned when a transaction already exists in the database
	ErrTransactionExists = errors.New("the transaction already exists")
)

func (m *mysqlConnector) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	result := m.db.WithContext(tCtx).Create(&tr)

	if result.Error != nil {
		return result.Error
	}

	return m.CreateTransactionLog(ctx, tr.ToTransactionLog())
}

func (m *mysqlConnector) UpdateTransaction(ctx context.Context, tr entities.Transaction, historize bool) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := m.db.WithContext(tCtx).
		Model(&entities.Transaction{}).
		Select(allowedFields).
		Where(&entities.Transaction{
			DebitorID:     tr.DebitorID,
			TransactionID: tr.TransactionID,
		}).
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

	if historize {
		return m.CreateTransactionLog(ctx, tr.ToTransactionLog())
	}

	return nil
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

func (m *mysqlConnector) GetAdminTransactionsByFilter(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
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

	res := db.Unscoped().Find(&transactions)
	if res.Error != nil {
		return nil, res.Error
	}

	return transactions, nil
}

func (m *mysqlConnector) GetValidTransactionsForDebitor(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	var transactions []entities.Transaction

	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := m.db.WithContext(tCtx).
		Where(&entities.Transaction{
			DebitorID:         debitorID,
			TransactionStatus: entities.TransactionStatusValid,
		}).Find(&transactions)

	if res.Error != nil {
		return nil, res.Error
	}

	return transactions, nil
}

func (m *mysqlConnector) QueryOutstandingDuesForDebitor(ctx context.Context, debitorID int64) (int64, error) {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	stmt := `SELECT 
COALESCE(SUM(p.gross_cent),0) - (
	SELECT
		COALESCE(SUM(psub.gross_cent),0)
	FROM
		pay_transactions psub
	WHERE
		psub.debitor_id = @debitorID AND psub.transaction_type = "payment" AND psub.transaction_status = "valid"
	)
FROM
	pay_transactions p
WHERE
p.debitor_id = @debitorID AND p.transaction_type = "due" AND p.transaction_status = "valid"`

	var amount int64

	res := m.db.WithContext(tCtx).
		Raw(stmt, sql.Named("debitorID", debitorID)).
		Find(&amount)

	return amount, res.Error
}

func (m *mysqlConnector) DeleteTransaction(ctx context.Context, tr entities.Transaction) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	// Do we have to make sure that this information was provided?
	if reflect.ValueOf(tr.Deletion).IsZero() {
		return errors.New("no deletion information was provided. Transaction cannot be flagged as deleted without")
	}

	res := m.db.WithContext(tCtx).
		Model(&entities.Transaction{}).
		Select([]string{"deleted_at", "deleted_status", "deleted_comment", "deleted_by", "transaction_status", "comment"}).
		Where(&entities.Transaction{
			DebitorID:     tr.DebitorID,
			TransactionID: tr.TransactionID,
		}).Updates(tr)

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
