package database

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

type Repository interface {
	Migrate() error
	TransactionRepository
	TransactionLogRepository
}

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, tr entities.Transaction) error
	GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType entities.TransactionType) (*entities.Transaction, error)
	GetTransactionsByFilter(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error)
	GetAdminTransactionsByFilter(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error)
	GetValidTransactionsForDebitor(ctx context.Context, debitorID int64) ([]entities.Transaction, error)
	QueryOutstandingDuesForDebitor(ctx context.Context, debitorID int64) (int64, error)
	UpdateTransaction(ctx context.Context, tr entities.Transaction, historize bool) error
	DeleteTransaction(ctx context.Context, tr entities.Transaction) error
}

type TransactionLogRepository interface {
	CreateTransactionLog(ctx context.Context, h entities.TransactionLog) error
	GetTransactionLogByID(ctx context.Context, id uint) (*entities.TransactionLog, error)
}
