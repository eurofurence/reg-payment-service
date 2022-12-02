package database

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

type Repository interface {
	Migrate() error
	TransactionRepository
	TransactionLogRepository
}

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, tr entities.Transaction) error
	GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType uint) (*entities.Transaction, error)
	GetTransactionsByFilter(ctx context.Context, debitorID int64) ([]entities.Transaction, error)
	UpdateTransaction(ctx context.Context, tr entities.Transaction) error
}

type TransactionLogRepository interface {
	CreateTransactionLog(ctx context.Context, h entities.TransactionLog) error
	GetTransactionLogByID(ctx context.Context, id int) (*entities.TransactionLog, error)
}
