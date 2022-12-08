package v1transactions

import (
	"github.com/eurofurence/reg-payment-service/internal/domain"
	"time"
)

func V1TransactionFrom(dom domain.Transaction) Transaction {
	return Transaction{
		DebitorID:             dom.DebitorID,
		TransactionIdentifier: dom.ID,
		TransactionType:       TransactionType(dom.Type.Descriptor()),
		Method:                PaymentMethod(dom.Method.Descriptor()),
		Amount: Amount{
			Currency:  dom.Amount.Currency,
			GrossCent: dom.Amount.GrossCent,
			VatRate:   dom.Amount.VatRate,
		},
		Comment:         dom.Comment,
		Status:          TransactionStatus(dom.Status.Descriptor()),
		Info:            make(map[string]interface{}), // TODO (no field)
		PaymentStartUrl: "",                           // TODO (no field)
		EffectiveDate:   "2022-12-08",                 // TODO convert to iso date (or we get timezone dependence after all)
		DueDate:         "2022-12-20",                 // TODO convert to iso date (or we get timezone dependence after all)
		CreationDate:    time.Now(),                   // TODO (no field)
		StatusHistory:   make([]StatusHistory, 0),     // TODO
	}
}
