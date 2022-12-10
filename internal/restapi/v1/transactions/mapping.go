package v1transactions

import (
	"time"

	"github.com/eurofurence/reg-payment-service/internal/domain"
)

func V1TransactionFrom(dom domain.Transaction) Transaction {
	return Transaction{
		DebitorID:             dom.DebitorID,
		TransactionIdentifier: dom.ID,
		TransactionType:       dom.Type,
		Method:                dom.Method,
		Amount: Amount{
			Currency:  dom.Amount.Currency,
			GrossCent: dom.Amount.GrossCent,
			VatRate:   dom.Amount.VatRate,
		},
		Comment:         dom.Comment,
		Status:          dom.Status,
		Info:            make(map[string]interface{}), // TODO (no field)
		PaymentStartUrl: "",                           // TODO (no field)
		EffectiveDate:   "2022-12-08",                 // TODO convert to iso date (or we get timezone dependence after all)
		DueDate:         "2022-12-20",                 // TODO convert to iso date (or we get timezone dependence after all)
		CreationDate:    time.Now(),                   // TODO (no field)
		StatusHistory:   make([]StatusHistory, 0),     // TODO
	}
}
