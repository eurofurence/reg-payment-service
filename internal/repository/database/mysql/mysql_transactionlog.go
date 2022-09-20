package mysql

import (
	"context"
	"log"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

func (m *mysqlConnector) CreateTransactionLog(ctx context.Context, tl entities.TransactionLog) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := m.db.WithContext(tCtx).Create(&tl)
	log.Println(res.Statement.SQL.String())
	return res.Error
}

func (m *mysqlConnector) GetTransactionLogByID(ctx context.Context, id int) (*entities.TransactionLog, error) {
	return nil, nil
}
