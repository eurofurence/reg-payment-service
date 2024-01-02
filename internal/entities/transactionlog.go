package entities

import (
	"database/sql"

	"gorm.io/gorm"
)

// type History struct {
// 	gorm.Model
// 	Entity    string `gorm:"type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL;index:entity_idx"` // the name of the entity
// 	EntityId  uint   `gorm:"index:entity_idx"`                                                                            // the pk of the entity
// 	RequestId string `gorm:"type:varchar(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`                            // optional request id that triggered the change
// 	UserId    uint   // optional - id of user who made the change
// 	Diff      string `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
// }

// TransactionLog holds information about the state of a transaction for a given time
//
// This table is append only
type TransactionLog struct {
	gorm.Model
	DebitorID         int64             `gorm:"index;type:bigint;NOT NULL"`
	TransactionID     string            `gorm:"index;type:varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;NOT NULL"`
	TransactionType   TransactionType   `gorm:"type:enum('due', 'payment')"`
	PaymentMethod     PaymentMethod     `gorm:"type:enum('credit', 'paypal', 'transfer', 'internal', 'gift', 'cash')"`
	PaymentStartUrl   string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;default:NULL"`
	TransactionStatus TransactionStatus `gorm:"type:enum('tentative', 'pending', 'valid', 'deleted')"`
	Amount            Amount            `gorm:"embedded"`
	Comment           string            `gorm:"type:text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`
	Deletion          Deletion          `gorm:"embedded;embeddedPrefix:deleted_"`
	EffectiveDate     sql.NullTime      `gorm:"type:date;NOT NULL"`
	DueDate           sql.NullTime      `gorm:"type:date;NULL;default:NULL"`
}

// // TableName implements the Tabler interface to change from a pluarlized table name to
// // the singular name.
// func (TransactionLog) TableName() string {
// 	return "transaction_log"
// }
