package v1transactions

import "time"

// GetTransactionsRequest will contain all information that will be sent from the
// client during calling the GetTransactions endpoint
type GetTransactionsRequest struct {
	// description: The id of a debitor to filter by
	DebitorID int64
	// filter by transaction_identifier
	TransactionIdentifier string
	// filter by effective date (inclusive) lower bound
	EffectiveFrom time.Time
	// filter by effective date (exclusive) upper bound - this makes it easy to get everything in a given month
	EffectiveBefore time.Time
}

type GetTransactionsResponse struct {
}
