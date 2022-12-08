package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	errUnkonwnTransactionType error = errors.New("transaction type not known")
)

type TransactionType int

const (
	Due TransactionType = iota
	Payment
)

var transactionTypeName = map[TransactionType]string{
	Due:     "Due",
	Payment: "Payment",
}

var transactionTypeValue = map[string]TransactionType{
	"DUE":     Due,
	"PAYMENT": Payment,
}

func (t TransactionType) Descriptor() string {
	if tn, ok := transactionTypeName[t]; ok {
		return tn
	}

	return ""
}

func TransactionTypeFromString(str string) (TransactionType, error) {
	if tv, ok := transactionTypeValue[strings.ToUpper(str)]; ok {
		return tv, nil
	}

	return -1, errUnkonwnTransactionType
}

type PaymentMethod int

const (
	Credit PaymentMethod = iota
	Paypal
	Transfer
	Internal
	Gift
)

var paymentMethodName = map[PaymentMethod]string{
	Credit:   "Credit",
	Paypal:   "Paypal",
	Transfer: "Transfer",
	Internal: "Internal",
	Gift:     "Gift",
}

func (m PaymentMethod) Descriptor() string {
	if pm, ok := paymentMethodName[m]; ok {
		return pm
	}

	return ""
}

type TransactionStatus int

const (
	Pending TransactionStatus = iota
	Tentative
	Valid
	Deleted
)

var transactionStatusName = map[TransactionStatus]string{
	Pending:   "Pending",
	Tentative: "Tentative",
	Valid:     "Valid",
	Deleted:   "Deleted",
}

func (t TransactionStatus) Descriptor() string {
	if ts, ok := transactionStatusName[t]; ok {
		return ts
	}

	return ""
}

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
