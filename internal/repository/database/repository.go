package database

import (
	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

type Repository interface {
	Migrate() error
	TransactionCRUD
	//HistoryCRUD
}

type TransactionCRUD interface {
	CreateTransaction(tr entities.Transaction) error
	GetTransactionByID(id int) (*entities.Transaction, error)
	UpdateTransaction(tr entities.Transaction) error
}

type HistoryCRUD interface {
	CreateHistory(h entities.History) error
	GetHistoryByID(id int) (*entities.History, error)
	UpdateHistory(h entities.History) error
}
