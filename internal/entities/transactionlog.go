package entities

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
	Transaction `gorm:"embedded"`
}

// // TableName implements the Tabler interface to change from a pluarlized table name to
// // the singular name.
// func (TransactionLog) TableName() string {
// 	return "transaction_log"
// }
