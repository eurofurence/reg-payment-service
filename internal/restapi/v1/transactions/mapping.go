package v1transactions

import (
	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func ToV1Transaction(tran entities.Transaction) Transaction {
	return Transaction{
		DebitorID:             tran.DebitorID,
		TransactionIdentifier: tran.TransactionID,
		TransactionType:       tran.TransactionType,
		Method:                tran.PaymentMethod,
		Amount: Amount{
			Currency:  tran.Amount.ISOCurrency,
			GrossCent: tran.Amount.GrossCent,
			VatRate:   tran.Amount.VatRate,
		},
		Comment:         tran.Comment,
		Status:          tran.TransactionStatus,
		Info:            make(map[string]interface{}), // TODO (no field)
		PaymentStartUrl: tran.PayLinkURL,
		EffectiveDate:   tran.EffectiveDate.Time.Format("2006-01-02"),
		DueDate:         tran.DueDate.Time.Format("2006-01-02"),
		CreationDate:    tran.CreatedAt,
	}
}
