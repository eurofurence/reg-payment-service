package domain

import (
	"errors"
	"time"
)

var (
	errUnkonwnTransactionType error = errors.New("transaction type not known")
)

type TransactionType string

const (
	TransactionTypeDue      TransactionType = "due"
	TransactionTypePayement TransactionType = "payment"
)

func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypeDue, TransactionTypePayement:
		return true
	}

	return false
}

type PaymentMethod string

const (
	PaymentMethodCredit   PaymentMethod = "credit"
	PaymentMethodPaypal   PaymentMethod = "paypal"
	PaymentMethodTransfer PaymentMethod = "transfer"
	PaymentMethodInternal PaymentMethod = "internal"
	PaymentMethodGift     PaymentMethod = "gift"
)

func (p PaymentMethod) IsValid() bool {
	switch p {
	case PaymentMethodCredit, PaymentMethodPaypal, PaymentMethodTransfer, PaymentMethodInternal, PaymentMethodGift:
		return true
	}

	return false
}

type TransactionStatus string

const (
	TransactionStatusTentative TransactionStatus = "tentative"
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusValid     TransactionStatus = "valid"
	TransactionStatusDeleted   TransactionStatus = "deleted"
)

func (t TransactionStatus) IsValid() bool {
	switch t {
	case TransactionStatusTentative, TransactionStatusPending, TransactionStatusValid, TransactionStatusDeleted:
		return true
	}

	return false
}

type AccountingDate string

type Deletion struct {
	PreviousStatus TransactionStatus
	Comment        string
	DeletedBy      string
	Date           time.Time
}

type Transaction struct {
	ID            string
	DebitorID     int64
	Type          TransactionType
	Method        PaymentMethod
	Amount        Amount
	Comment       string
	Status        TransactionStatus
	EffectiveDate time.Time
	DueDate       time.Time
	Deletion      *Deletion
}
