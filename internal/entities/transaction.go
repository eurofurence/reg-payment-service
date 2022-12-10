package entities

import (
	"database/sql"

	"gorm.io/gorm"
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

type Transaction struct {
	gorm.Model
	DebitorID         int64             `gorm:"index;type:bigint;NOT NULL"`
	TransactionID     string            `gorm:"index;type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
	TransactionType   TransactionType   `gorm:"type:enum('due', 'payment')"`
	PaymentMethod     PaymentMethod     `gorm:"type:enum('credit', 'paypal', 'transfer', 'internal', 'gift')"`
	TransactionStatus TransactionStatus `gorm:"type:enum('tentative', 'pending', 'valid', 'deleted')"`
	Amount            Amount            `gorm:"embedded"`
	Comment           string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Deletion          Deletion          `gorm:"embedded;embeddedPrefix:deleted_"`
	EffectiveDate     sql.NullTime
	DueDate           sql.NullTime
}

type Amount struct {
	ISOCurrency string `gorm:"type:varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL; "`
	GrossCent   int64
	VatRate     float64 `gorm:"type:decimal(10,2)"`
}

type Deletion struct {
	Status  TransactionStatus `gorm:"type:enum('tentative', 'pending', 'valid', 'deleted')"`
	Comment string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	By      string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
}

func (t *Transaction) ToTransactionLog() TransactionLog {
	return TransactionLog{
		Transaction: *t,
	}
}
