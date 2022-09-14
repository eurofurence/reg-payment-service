package database

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

type Repository interface {
	CreateFoo(ctx context.Context, f entities.Foo) error
}
