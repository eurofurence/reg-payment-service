package mysql

import (
	"fmt"
	"strings"
	"time"

	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
)

type mysqlConnector struct {
	logger logging.Logger
	db     *gorm.DB
}

func NewMySQLConnector(conf config.DatabaseConfig, logger logging.Logger) (database.Repository, error) {
	dsn, err := buildMySQLDSN(conf.Username, conf.Password, conf.Database, conf.Parameters)
	if err != nil {
		return nil, err
	}

	gormConfig := gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "pay_",
		},
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}
	db, err := gorm.Open(mysql.Open(dsn), &gormConfig)
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
		logger: logger,
		db:     db,
	}, nil

}

func (i *mysqlConnector) Migrate() error {
	err := i.db.AutoMigrate(
		&entities.Transaction{},
		&entities.TransactionLog{},
	)

	if err != nil {
		return err
	}

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
