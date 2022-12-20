package inmemory

import (
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
)

var _ database.Repository = (*inmemoryProvider)(nil)

type inmemoryProvider struct {
	transactions    map[uint]entities.Transaction
	transactionLogs map[uint]entities.TransactionLog
	idSequence      uint32
}

func NewInMemoryProvider() database.Repository {
	return &inmemoryProvider{
		transactions:    make(map[uint]entities.Transaction),
		transactionLogs: make(map[uint]entities.TransactionLog),
	}
}

func (i *inmemoryProvider) Migrate() error {
	// Nothing to do here
	return nil
}
