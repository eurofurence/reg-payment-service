package v1transactions

import (
	"time"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

type InitiatePaymentRequest struct{}

type InitiatePaymentResponse struct{}

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
	Payload []Transaction `json:"payload"`
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
	Status     entities.TransactionStatus `json:"status"`
	Comment    string                     `json:"comment"`
	ChangedBy  string                     `json:"changed_by"`
	ChangeDate time.Time                  `json:"change_date"`
}

// CreateTrasactionRequest contains all information to create a new transaction for a given debitor
type CreateTransactionRequest struct {
	Transaction Transaction `json:"transaction"`
}

type CreateTransactionResponse struct {
	Transaction Transaction `json:"transaction"`
}

type UpdateTransactionRequest struct{}

type UpdateTransactionResponse struct{}

type Transaction struct {
	// TODO missing ID for responses -> can't update
	DebitorID             int64                       `json:"debitor_id"`
	TransactionIdentifier string                      `json:"transaction_identifier"`
	TransactionType       entities.TransactionType    `json:"transaciont_type"`
	Method                entities.PaymentMethod      `json:"method"`
	Amount                Amount                      `json:"amount"`
	Comment               string                      `json:"comment"`
	Status                entities.TransactionStatus  `json:"status"`
	Info                  PaymentProcessorInformation `json:"payment_processor_information"`
	PaymentStartUrl       string                      `json:"payment_start_url"`
	EffectiveDate         string                      `json:"effective_date"`
	DueDate               string                      `json:"due_date,omitempty"`
	CreationDate          time.Time                   `json:"creation_date"`
	StatusHistory         []StatusHistory             `json:"status_history"`
}
