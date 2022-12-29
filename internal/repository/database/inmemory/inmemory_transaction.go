package inmemory

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func (m *inmemoryProvider) CreateTransaction(ctx context.Context, tr entities.Transaction) error {
	if tr.ID != 0 {
		return errors.New("create needs a new transaction")
	}
	tr.ID = uint(atomic.AddUint32(&m.idSequence, 1))

	// set a creation date if none was provided beforehand
	if tr.CreatedAt.IsZero() {
		tr.CreatedAt = time.Now()
	}

	m.transactions[tr.ID] = tr
	return nil
}

func (m *inmemoryProvider) UpdateTransaction(ctx context.Context, tr entities.Transaction, _ bool) error {
	if tr.ID == 0 {
		found, err := m.GetTransactionByTransactionIDAndType(ctx, tr.TransactionID, tr.TransactionType)
		if err != nil {
			return err
		}

		tr.ID = found.ID
	}

	_, ok := m.transactions[tr.ID]
	if !ok {
		return errors.New("transaction not found in database")
	}
	m.transactions[tr.ID] = tr

	return nil
}

func (m *inmemoryProvider) GetTransactionByTransactionIDAndType(ctx context.Context, transactionID string, tType entities.TransactionType) (*entities.Transaction, error) {
	for _, t := range m.transactions {
		if t.TransactionID == transactionID && t.TransactionType == tType {
			copy := t
			return &copy, nil
		}
	}
	return &entities.Transaction{}, errors.New("no matching transaction in database")
}

func (m *inmemoryProvider) GetTransactionsByFilter(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	result := make([]entities.Transaction, 0)
	for _, t := range m.transactions {
		if query.DebitorID != 0 && t.DebitorID != query.DebitorID {
			continue
		}
		if query.TransactionIdentifier != "" && t.TransactionID != query.TransactionIdentifier {
			continue
		}

		if !query.EffectiveFrom.IsZero() && query.EffectiveFrom.After(t.EffectiveDate.Time) {
			continue
		}
		if !query.EffectiveBefore.IsZero() && !t.EffectiveDate.Time.Before(query.EffectiveBefore) {
			// if !(20 < 28) break
			continue
		}
		result = append(result, t)
	}
	return result, nil
}

func (m *inmemoryProvider) GetValidTransactionsForDebitor(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
	result := make([]entities.Transaction, 0)
	for _, t := range m.transactions {
		if t.DebitorID == debitorID && t.TransactionStatus == entities.TransactionStatusValid {
			result = append(result, t)
		}
	}

	return result, nil
}

func (m *inmemoryProvider) QueryOutstandingDuesForDebitor(ctx context.Context, debutorID int64) (int64, error) {
	dues := int64(0)
	payments := int64(0)

	for _, tr := range m.transactions {
		if reflect.ValueOf(tr.Deletion).IsZero() && tr.TransactionStatus == entities.TransactionStatusValid {
			if tr.TransactionType == entities.TransactionTypeDue {
				dues += tr.Amount.GrossCent
			}

			if tr.TransactionType == entities.TransactionTypePayment {
				payments += tr.Amount.GrossCent
			}
		}
	}

	return (dues - payments), nil
}

func (m *inmemoryProvider) DeleteTransaction(ctx context.Context, tr entities.Transaction) error {
	if cur, e := m.transactions[tr.ID]; e {
		cur.DeletedAt = gorm.DeletedAt{Time: time.Now().UTC(), Valid: true}
		cur.Deletion = entities.Deletion{
			Status:  tr.TransactionStatus,
			Comment: tr.Deletion.Comment,
			By:      tr.Deletion.By,
		}
		cur.TransactionStatus = entities.TransactionStatusDeleted

		m.transactions[cur.ID] = cur
	}

	return nil
}
