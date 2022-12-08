package v1transactions

import "time"

type TransactionType string

const (
	TransactionTypeDue      TransactionType = "due"
	TransactionTypePayement TransactionType = "payment"
)

func (t TransactionType) IsValid() bool {
	switch t {
	case TransactionTypeDue, TransactionTypePayement:
		return true
	}

	return false
}

type PaymentMethod string

const (
	PaymentMethodCredit   PaymentMethod = "credit"
	PaymentMethodPaypal   PaymentMethod = "paypal"
	PaymentMethodTransfer PaymentMethod = "transfer"
	PaymentMethodInternal PaymentMethod = "internal"
	PaymentMethodGift     PaymentMethod = "gift"
)

func (p PaymentMethod) IsValid() bool {
	switch p {
	case PaymentMethodCredit, PaymentMethodPaypal, PaymentMethodTransfer, PaymentMethodInternal, PaymentMethodGift:
		return true
	}

	return false
}

type TransactionStatus string

const (
	TransactionStatusTentative TransactionStatus = "tentative"
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusValid     TransactionStatus = "valid"
	TransactionStatusDeleted   TransactionStatus = "deleted"
)

func (t TransactionStatus) IsValid() bool {
	switch t {
	case TransactionStatusTentative, TransactionStatusPending, TransactionStatusValid, TransactionStatusDeleted:
		return true
	}

	return false
}

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
	Status     TransactionStatus `json:"status"`
	Comment    string            `json:"comment"`
	ChangedBy  string            `json:"changed_by"`
	ChangeDate time.Time         `json:"change_date"`
}

// CreateTrasactionRequest contains all information to create a new transaction for a given debitor
type CreateTransactionRequest struct {
	Transaction Transaction `json:"transaction"`
}

type CreateTransactionResponse struct {
}

type UpdateTransactionRequest struct{}

type UpdateTransactionResponse struct{}

type AccountingDate string

type Transaction struct {
	// TODO missing ID for responses -> can't update
	DebitorID             int64                       `json:"debitor_id"`
	TransactionIdentifier string                      `json:"transaction_identifier"`
	TransactionType       TransactionType             `json:"transaciont_type"`
	Method                PaymentMethod               `json:"method"`
	Amount                Amount                      `json:"amount"`
	Comment               string                      `json:"comment"`
	Status                TransactionStatus           `json:"status"`
	Info                  PaymentProcessorInformation `json:"payment_processor_information"`
	PaymentStartUrl       string                      `json:"payment_start_url"`
	EffectiveDate         AccountingDate              `json:"effective_date"`
	DueDate               AccountingDate              `json:"due_date"`
	CreationDate          time.Time                   `json:"creation_date"`
	StatusHistory         []StatusHistory             `json:"status_history"`
}
