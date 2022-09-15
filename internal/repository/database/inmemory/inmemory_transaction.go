package inmemory

import "github.com/eurofurence/reg-payment-service/internal/repository/entities"

func (m *inmemoryProvider) CreateTransaction(tr entities.Transaction) error {
	panic("not implemented") // TODO: Implement
}

func (m *inmemoryProvider) GetTransactionByID(id int) (*entities.Transaction, error) {
	panic("not implemented") // TODO: Implement
}

func (m *inmemoryProvider) UpdateTransaction(tr entities.Transaction) error {
	panic("not implemented") // TODO: Implement
}
