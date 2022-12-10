package entities

import "time"

type TransactionQuery struct {
	// description: The id of a debitor to filter by
	DebitorID int64
	// filter by transaction_identifier
	TransactionIdentifier string
	// filter by effective date (inclusive) lower bound
	EffectiveFrom time.Time
	// filter by effective date (exclusive) upper bound - this makes it easy to get everything in a given month
	EffectiveBefore time.Time
}
