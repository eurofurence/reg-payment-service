package inmemory

import (
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
)

var _ database.Repository = (*inmemoryProvider)(nil)

type inmemoryProvider struct {
}

func NewInMemoryProvider() database.Repository {
	return &inmemoryProvider{}
}

func (i *inmemoryProvider) Migrate() error {
	// Nothing to do here
	return nil
}
