package database

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

var _ Repository = (*inmemoryProvider)(nil)

type inmemoryProvider struct{}

func NewInMemoryProvider() Repository {
	return &inmemoryProvider{}
}

func (i *inmemoryProvider) CreateFoo(ctx context.Context, f entities.Foo) error {

	return nil
}
