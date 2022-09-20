package entities

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model
	TransactionID       string            `gorm:"index;type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
	DebitorID           string            `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
	TransactionTypeID   uint              `gorm:"NOT NULL"`
	TransactionType     TransactionType   `gorm:"constraint:OnUpdate:CASCADE;NOT NULL"`
	PaymentMethodID     uint              `gorm:"NOT NULL"`
	PaymentMethod       PaymentMethod     `gorm:"constraint:OnUpdate:CASCADE;NOT NULL"`
	TransactionStatusID uint              `gorm:"NOT NULL"`
	TransactionStatus   TransactionStatus `gorm:"constraint:OnUpdate:CASCADE;NOT NULL"`
	Amount              Amount            `gorm:"embedded"`
	Comment             string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Deletion            Deletion          `gorm:"embedded;embeddedPrefix:deleted_"`
	EffectiveDate       sql.NullTime
	DueDate             sql.NullTime
}

type TransactionType struct {
	ID          uint `gorm:"primarykey;autoIncrement:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;unique"`
}
type PaymentMethod struct {
	ID          uint `gorm:"primarykey;autoIncrement:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;unique"`
}

type TransactionStatus struct {
	ID          uint `gorm:"primarykey;autoIncrement:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;unique"`
}

type Amount struct {
	ISOCurrency string `gorm:"type:varchar(3) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL; "`
	GrossCent   int64
	VatRate     float64 `gorm:"type:decimal(10,2)"`
}

type Deletion struct {
	TransactionStatusID int
	Status              TransactionStatus `gorm:"constraint:OnUpdate:CASCADE"`
	Comment             string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	By                  string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
}

func (t *Transaction) ToTransactionLog() TransactionLog {
	return TransactionLog{
		TransactionID:     t.TransactionID,
		DebitorID:         t.DebitorID,
		TransactionTypeID: t.TransactionTypeID,
		TransactionType: TransactionType{
			Description: t.TransactionType.Description,
		},
		PaymentMethodID: t.PaymentMethodID,
		PaymentMethod: PaymentMethod{
			Description: t.PaymentMethod.Description,
		},
		TransactionStatusID: t.TransactionStatusID,
		TransactionStatus: TransactionStatus{
			Description: t.TransactionStatus.Description,
		},
		Amount:        t.Amount,
		Comment:       t.Comment,
		Deletion:      t.Deletion,
		EffectiveDate: t.EffectiveDate,
		DueDate:       t.DueDate,
	}
}
