package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
)

type mysqlProvider struct {
	db *gorm.DB
}

func buildMySQLDSN(username, password, database string, parameters []string) string {
	// TODO validation?

	return fmt.Sprintf("%s:%s@%s?%s", username, password, database, strings.Join(parameters, "&"))
}

func NewMySQLProvider(conf config.DatabaseConfig) (Repository, error) {
	dsn := buildMySQLDSN(conf.Username, conf.Password, conf.Database, conf.Parameters)

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

	return &mysqlProvider{
		db: db,
	}, nil
}

func (m *mysqlProvider) CreateFoo(ctx context.Context, f entities.Foo) error {
	if !m.db.Migrator().HasTable(&entities.Foo{}) {
		if err := m.db.AutoMigrate(&entities.Foo{}); err != nil {
			return err
		}
	}

	return m.db.WithContext(ctx).Create(&f).Error
}
