package mysql

import "github.com/eurofurence/reg-payment-service/internal/repository/entities"

func (m *mysqlConnector) CreateTransaction(tr entities.Transaction) error {
	return nil
}

func (m *mysqlConnector) GetTransactionByID(id int) (*entities.Transaction, error) {
	return nil, nil
}

func (m *mysqlConnector) UpdateTransaction(tr entities.Transaction) error {
	return nil
}
