package domain

import (
	"errors"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

var (
	errUnkonwnTransactionType error = errors.New("transaction type not known")
)

type AccountingDate string

type Deletion struct {
	PreviousStatus entities.TransactionStatus
	Comment        string
	DeletedBy      string
	Date           time.Time
}

type Transaction struct {
	ID            string
	DebitorID     int64
	Type          entities.TransactionType
	Method        entities.PaymentMethod
	Amount        Amount
	Comment       string
	Status        entities.TransactionStatus
	EffectiveDate time.Time
	DueDate       time.Time
	Deletion      *Deletion
}
