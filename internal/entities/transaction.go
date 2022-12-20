package entities

import (
	"database/sql"

	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypeDue     TransactionType = "due"
	TransactionTypePayment TransactionType = "payment"
)

func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypeDue, TransactionTypePayment:
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
	TransactionID     string            `gorm:"uniqueIndex:idx_uq_tid;type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
	TransactionType   TransactionType   `gorm:"type:enum('due', 'payment')"`
	PaymentMethod     PaymentMethod     `gorm:"type:enum('credit', 'paypal', 'transfer', 'internal', 'gift')"`
	PaymentStartUrl   string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;default:NULL"`
	TransactionStatus TransactionStatus `gorm:"type:enum('tentative', 'pending', 'valid', 'deleted')"`
	Amount            Amount            `gorm:"embedded"`
	Comment           string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Deletion          Deletion          `gorm:"embedded;embeddedPrefix:deleted_"`
	EffectiveDate     sql.NullTime      `gorm:"type:date;NOT NULL"`
	DueDate           sql.NullTime      `gorm:"type:date;NULL;default:NULL"`
}

type Amount struct {
	ISOCurrency string `gorm:"type:varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL; "`
	GrossCent   int64
	VatRate     float64 `gorm:"type:decimal(10,2)"`
}

type Deletion struct {
	Status  TransactionStatus `gorm:"type:enum('tentative', 'pending', 'valid', 'deleted');NULL;default:NULL"`
	Comment string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	By      string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
}

func (t *Transaction) ToTransactionLog() TransactionLog {
	return TransactionLog{
		DebitorID:         t.DebitorID,
		TransactionID:     t.TransactionID,
		TransactionType:   t.TransactionType,
		PaymentMethod:     t.PaymentMethod,
		PaymentStartUrl:   t.PaymentStartUrl,
		TransactionStatus: t.TransactionStatus,
		Amount: Amount{
			ISOCurrency: t.Amount.ISOCurrency,
			GrossCent:   t.Amount.GrossCent,
			VatRate:     t.Amount.VatRate,
		},
		Comment: t.Comment,
		Deletion: Deletion{
			Status:  t.Deletion.Status,
			Comment: t.Deletion.Comment,
			By:      t.Deletion.By,
		},
		EffectiveDate: t.EffectiveDate,
		DueDate:       t.DueDate,
	}
}
