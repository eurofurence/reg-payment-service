package entities

import "database/sql"

type Foo struct {
	Name    string
	Age     uint32
	Address sql.NullString
}
