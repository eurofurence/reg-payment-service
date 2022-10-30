package v1transactions

import "time"

type TransactionType string

const (
	TransactionTypeDue      TransactionType = "due"
	TransactionTypePayement TransactionType = "payment"
)

type PayementMethod string

const (
	PaymentMethodCredit   PayementMethod = "credit"
	PaymentMethodPaypal   PayementMethod = "paypal"
	PaymentMethodTransfer PayementMethod = "transfer"
	PaymentMethodInternal PayementMethod = "internal"
	PaymentMethodGift     PayementMethod = "gift"
)

type TransactionStatus string

const (
	TransactionStatusTentative TransactionStatus = "tentative"
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusValid     TransactionStatus = "valid"
	TransactionStatusDeleted   TransactionStatus = "deleted"
)

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

type Amount struct {
	// Currency is the ISO 4217 currency code
	Currency string `json:"currency"`
	// The value contained for the given currency in the smallest possible definition
	GrossCent int64 `json:"gross_cent"`
	// The defined vat rate
	VatRate float64 `json:"vat_rate"`
}

type PaymentProcessorInformation map[string]interface{}

type StatusHistory struct {
	Status     TransactionStatus `json:"status"`
	Comment    string            `json:"comment"`
	ChangedBy  string            `json:"changed_by"`
	ChangeDate time.Time         `json:"change_date"`
}

// CreateTrasactionRequest contains all information to create a new transaction for a given debitor
type CreateTransactionRequest struct {
	DebitorID             int64                       `json:"debitor_id"`
	TransactionIdentifier string                      `json:"transaction_identifier"`
	TransactionType       TransactionType             `json:"transaciont_type"`
	Method                PayementMethod              `json:"method"`
	Amount                Amount                      `json:"amount"`
	Comment               string                      `json:"comment"`
	Status                TransactionStatus           `json:"status"`
	Info                  PaymentProcessorInformation `json:"payment_processor_information"`
	PaymentStartUrl       string                      `json:"payment_start_url"`
	EffectiveDate         time.Time                   `json:"effective_date"`
	DueDate               time.Time                   `json:"due_date"`
	CreationDate          time.Time                   `json:"creation_date"`
	StatusHistory         []StatusHistory             `json:"status_history"`
}

type CreateTransactionResponse struct {
}
