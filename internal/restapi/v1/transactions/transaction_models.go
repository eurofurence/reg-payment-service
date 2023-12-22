package v1transactions

import (
	"time"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

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

// request and response types
type (
	// GetTransactionsRequest will contain all information that will be sent from the
	// client during calling the GetTransactions endpoint
	GetTransactionsRequest struct {
		// description: The id of a debitor to filter by
		DebitorID int64
		// filter by transaction_identifier
		TransactionIdentifier string
		// filter by effective date (inclusive) lower bound
		EffectiveFrom time.Time
		// filter by effective date (exclusive) upper bound - this makes it easy to get everything in a given month
		EffectiveBefore time.Time
	}

	// GetTransactionsResponse contains a number of transactions depending on the search criteria
	// provided in the `GetTransactionsRequest`
	GetTransactionsResponse struct {
		Payload []Transaction `json:"payload"`
	}

	// CreateTrasactionRequest contains all information to create a new transaction for a given debitor
	CreateTransactionRequest struct {
		Transaction Transaction `json:"transaction"`
	}

	// CreateTransactionResponse contains the transaction, which was created through the request parameters
	CreateTransactionResponse struct {
		Transaction Transaction `json:"transaction"`
	}

	// UpdateTransactionRequest contains information to perform an update to a transaction.
	// Based on the request permissions (JWT, API Token, Admin), the fields that may be altered may vary
	UpdateTransactionRequest struct {
		Transaction Transaction `json:"transaction"`
	}

	// UpdateTransactionResponse is am empty response as this endpoint yields no response
	UpdateTransactionResponse struct{}

	// InitiatePaymentRequest is used for a convenience endpoint to create a payment transaction with the default values.
	InitiatePaymentRequest struct {
		TransactionInitiator TransactionInitiator
	}

	// InitiatePaymentResponse contains the transaction, that was created through the request
	InitiatePaymentResponse struct {
		Transaction Transaction `json:"transaction"`
	}
)

type Transaction struct {
	DebitorID             int64                       `json:"debitor_id"`
	TransactionIdentifier string                      `json:"transaction_identifier"`
	TransactionType       entities.TransactionType    `json:"transaction_type"`
	Method                entities.PaymentMethod      `json:"method"`
	Amount                Amount                      `json:"amount"`
	Comment               string                      `json:"comment"`
	Status                entities.TransactionStatus  `json:"status"`
	Info                  PaymentProcessorInformation `json:"payment_processor_information"`
	PaymentStartUrl       string                      `json:"payment_start_url"`
	EffectiveDate         string                      `json:"effective_date"`
	DueDate               string                      `json:"due_date,omitempty"`
	CreationDate          *time.Time                  `json:"creation_date,omitempty"`
	StatusHistory         []StatusHistory             `json:"status_history"`
}

type TransactionInitiator struct {
	DebitorID int64                  `json:"debitor_id"`
	Method    entities.PaymentMethod `json:"method"`
}
