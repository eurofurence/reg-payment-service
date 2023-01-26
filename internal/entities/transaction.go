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

// dues from attendee service:
//
// 	return paymentservice.Transaction{
//	OK	DebitorID:       attendee.ID,
//	OK	TransactionType: paymentservice.Due,
//	OK	Method:          paymentservice.Internal,
//		Amount: paymentservice.Amount{
//	 OK		Currency:  config.Currency(),
//	 OK		GrossCent: amount,
//	 OK		VatRate:   vat,
//		},
//	OK	Comment:       comment,
//	OK	Status:        paymentservice.Valid,
//	OK	EffectiveDate: s.duesEffectiveDate(),
//	opt	DueDate:       s.duesDueDate(),
//	}

// payments from concardis adapter:
//
// create if got lost:
//
// transaction := paymentservice.Transaction{
//	OK	ID:        paylink.ReferenceID, `transaction_identifier`
//	OK	DebitorID: debitor_id,
//	OK	Type:      paymentservice.Payment, `transaction_type`
//	OK	Method:    paymentservice.Credit, // XXX TODO: this is a guess. We use paylink for credit cards only, atm.
//		Amount: paymentservice.Amount{
//	 OK		GrossCent: paylink.Amount,
//	 OK		Currency:  paylink.Currency,
//	 OK		VatRate:   0, // TODO should set from payload
//		},
//	OK	Comment:       "Auto-created by cncrd adapter because the reference_id could not be found in the payment service.",
//	OK	Status:        paymentservice.Pending,
//	OK	EffectiveDate: today, // XXX TODO: this might be in the payload
//	opt	DueDate:       today,
//	}
//
// update if existing:
//  - read via GET, then:
//	transaction.Amount.GrossCent = paylink.Amount
//	transaction.Amount.Currency = paylink.Currency
//	transaction.Status = paymentservice.Pending // TODO fail if already in valid and values do not match (admin might have done this in the mean time)
//	transaction.EffectiveDate = "xxx"

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
