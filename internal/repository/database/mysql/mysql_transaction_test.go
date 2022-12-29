//go:build database
// +build database

package mysql

// import (
// 	"context"
// 	"database/sql"
// 	"testing"
// 	"time"

// 	"github.com/eurofurence/reg-payment-service/internal/config"
// 	"github.com/eurofurence/reg-payment-service/internal/entities"
// 	"github.com/eurofurence/reg-payment-service/internal/logging"
// 	"github.com/stretchr/testify/require"
// )

// // build only when docker container is running
// // TODO use testcontainers?
// func TestABC(t *testing.T) {
// 	dbConf := config.DatabaseConfig{
// 		Use:      config.Mysql,
// 		Username: "root",
// 		Password: "example",
// 		Database: "tcp(localhost:3306)/test",
// 		Parameters: []string{
// 			"charset=utf8mb4",
// 			"collation=utf8mb4_general_ci",
// 			"parseTime=True",
// 			"timeout=30s",
// 		},
// 	}

// 	repo, err := NewMySQLConnector(dbConf, logging.NewNoopLogger())
// 	require.NoError(t, err)

// 	dues, err := repo.QueryOutstandingDuesForDebitor(context.Background(), 1)
// 	require.NoError(t, err)

// 	require.Equal(t, 2000, dues)
// }

// func newTransaction(builder func(t entities.Transaction)) entities.Transaction {
// 	t := entities.Transaction{
// 		DebitorID:         999,
// 		TransactionID:     "",
// 		TransactionType:   entities.TransactionTypeDue,
// 		PaymentMethod:     entities.PaymentMethodCredit,
// 		PaymentStartUrl:   "",
// 		TransactionStatus: entities.TransactionStatusTentative,
// 		Amount: entities.Amount{
// 			ISOCurrency: "EUR",
// 			GrossCent:   1500,
// 			VatRate:     19.,
// 		},
// 		Comment: "Test Comment",
// 		EffectiveDate: sql.NullTime{
// 			Time:  time.Now().UTC(),
// 			Valid: false,
// 		},
// 	}

// 	return t

// }
