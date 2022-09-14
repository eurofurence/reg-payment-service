package entities

import (
	"database/sql"

	"gorm.io/gorm"
)

type Foo struct {
	gorm.Model
	Name    string `gorm:"index:idx_foo_name,sort:desc"`
	Age     uint32
	Address sql.NullString
}
