package entities

import (
	"database/sql"
	"github.com/eurofurence/reg-payment-service/internal/domain"
	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model
	DebitorID         int64                    `gorm:"index;type:bigint;NOT NULL"`
	TransactionID     string                   `gorm:"index;type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
	TransactionType   domain.TransactionType   `gorm:"type:enum('due', 'payment')"`
	PaymentMethod     domain.PaymentMethod     `gorm:"type:enum('credit', 'paypal', 'transfer', 'internal', 'gift')"`
	TransactionStatus domain.TransactionStatus `gorm:"type:enum('tentative', 'pending', 'valid', 'deleted')"`
	Amount            Amount                   `gorm:"embedded"`
	Comment           string                   `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Deletion          Deletion                 `gorm:"embedded;embeddedPrefix:deleted_"`
	EffectiveDate     sql.NullTime
	DueDate           sql.NullTime
}

type Amount struct {
	ISOCurrency string `gorm:"type:varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL; "`
	GrossCent   int64
	VatRate     float64 `gorm:"type:decimal(10,2)"`
}

type Deletion struct {
	Status  domain.TransactionStatus `gorm:"type:enum('tentative', 'pending', 'valid', 'deleted')"`
	Comment string                   `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	By      string                   `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
}

func (t *Transaction) ToTransactionLog() TransactionLog {
	return TransactionLog{
		Transaction: *t,
	}
}
