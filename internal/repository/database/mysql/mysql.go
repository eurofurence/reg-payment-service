package mysql

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

type mysqlConnector struct {
	db *gorm.DB
}

func NewMySQLConnector(conf config.DatabaseConfig) (database.Repository, error) {
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
		db: db,
	}, nil
}

func (i *mysqlConnector) Migrate() error {
	i.db.AutoMigrate(
		&entities.TransactionType{},
		&entities.TransactionStatus{},
		&entities.PaymentMethod{},
		&entities.Transaction{},
		&entities.History{},
	)
	return nil
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

	return fmt.Sprintf("%s:%s@%s?%s", username, password, database, strings.Join(parameters, "&")), nil
}

func checkValue(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", name)
	}

	return nil
}
