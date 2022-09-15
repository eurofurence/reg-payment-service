package entities

import (
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model
	TransactionID       string            `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
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
	EffectiveDate       time.Time
	DueDate             time.Time
}

type PaymentMethod struct {
	gorm.Model
	Description string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;unique"`
}

type TransactionType struct {
	gorm.Model
	Description string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;unique"`
}

type TransactionStatus struct {
	gorm.Model
	Description string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;unique"`
}

type Amount struct {
	Currency  string `gorm:"varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL; "`
	GrossCent int64
	VatRate   float64 `gorm:"type:decimal(10,2)"`
}

type Deletion struct {
	TransactionStatusID int
	Status              TransactionStatus `gorm:"constraint:OnUpdate:CASCADE"`
	Comment             string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	By                  string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
}

/*
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

type TransactionStatus int

const (
	Pending TransactionStatus = iota
	Tentative
	Valid
	Deleted
)

type Deletion struct {
	PreviousStatus TransactionStatus
	Comment        string
	DeletedBy      string
	Date           time.Time
}

type Transaction struct {
	ID            string
	DebitorID     string
	Type          TransactionType
	Method        PaymentMethod
	Amount        Amount
	Comment       string
	Status        TransactionStatus
	EffectiveDate time.Time
	DueDate       time.Time
	Deletion      *Deletion
}

*/
