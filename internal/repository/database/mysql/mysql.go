package mysql

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

type mysqlConnector struct {
	lock   sync.RWMutex
	logger logging.Logger
	db     *gorm.DB
}

func NewMySQLConnector(conf config.DatabaseConfig, logger logging.Logger) (database.Repository, error) {
	dsn, err := buildMySQLDSN(conf.Username, conf.Password, conf.Database, conf.Parameters)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetConnMaxLifetime(time.Minute * 10)

	return &mysqlConnector{
		lock:   sync.RWMutex{},
		logger: logger,
		db:     db,
	}, nil

}

func (i *mysqlConnector) Migrate() error {
	err := i.db.AutoMigrate(
		&entities.TransactionType{},
		&entities.PaymentMethod{},
		&entities.TransactionStatus{},
		&entities.Transaction{},
		&entities.TransactionLog{},
	)

	if err != nil {
		return err
	}

	i.populateStates()

	return nil
}

func (i *mysqlConnector) populateStates() {
	// Transaction Types
	i.db.FirstOrCreate(&entities.TransactionType{ID: 0, Description: "Due"})
	i.db.FirstOrCreate(&entities.TransactionType{ID: 1, Description: "Payment"})

	// Payment Methods
	i.db.FirstOrCreate(&entities.PaymentMethod{ID: 0, Description: "Credit"})
	i.db.FirstOrCreate(&entities.PaymentMethod{ID: 1, Description: "Paypal"})
	i.db.FirstOrCreate(&entities.PaymentMethod{ID: 2, Description: "Transfer"})
	i.db.FirstOrCreate(&entities.PaymentMethod{ID: 3, Description: "Internal"})
	i.db.FirstOrCreate(&entities.PaymentMethod{ID: 4, Description: "Gift"})

	// Transaction States
	i.db.FirstOrCreate(&entities.TransactionStatus{ID: 0, Description: "Pending"})
	i.db.FirstOrCreate(&entities.TransactionStatus{ID: 1, Description: "Tentative"})
	i.db.FirstOrCreate(&entities.TransactionStatus{ID: 2, Description: "Valid"})
	i.db.FirstOrCreate(&entities.TransactionStatus{ID: 3, Description: "Deleted"})
}

func buildMySQLDSN(username, password, database string, parameters []string) (string, error) {
	vals := map[string]string{
		"username": username,
		"password": password,
		"database": database,
	}

	for n, v := range vals {
		err := checkValue(n, v)
		if err != nil {
			return "", err
		}
	}

	paramStr := func() string {
		if len(parameters) == 0 {
			return ""
		}

		return fmt.Sprintf("?%s", strings.Join(parameters, "&"))
	}

	return fmt.Sprintf("%s:%s@%s%s", username, password, database, paramStr()), nil
}

func checkValue(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", name)
	}

	return nil
}
