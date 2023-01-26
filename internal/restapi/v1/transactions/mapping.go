package v1transactions

import (
	"database/sql"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/entities"
)

func ToV1Transaction(tran entities.Transaction) Transaction {
	result := Transaction{
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
		PaymentStartUrl: tran.PaymentStartUrl,
		EffectiveDate:   tran.EffectiveDate.Time.Format("2006-01-02"),
	}

	if !tran.CreatedAt.IsZero() {
		result.CreationDate = &tran.CreatedAt
	}

	if !tran.DueDate.Time.IsZero() {
		result.DueDate = tran.DueDate.Time.Format("2006-01-02")
	}

	return result

}

func ToTransactionEntity(tr Transaction) (*entities.Transaction, error) {
	effDate, err := parseEffectiveDate(tr.EffectiveDate)
	if err != nil {
		return nil, err
	}

	tran := &entities.Transaction{
		DebitorID:         tr.DebitorID,
		TransactionID:     tr.TransactionIdentifier,
		TransactionType:   tr.TransactionType,
		PaymentMethod:     tr.Method,
		PaymentStartUrl:   tr.PaymentStartUrl,
		TransactionStatus: tr.Status,
		Amount: entities.Amount{
			ISOCurrency: tr.Amount.Currency,
			GrossCent:   tr.Amount.GrossCent,
			VatRate:     tr.Amount.VatRate,
		},
		Comment: tr.Comment,
		EffectiveDate: sql.NullTime{
			Valid: true,
			Time:  effDate,
		},
	}

	if tr.DueDate != "" {
		dueDate, err := parseEffectiveDate(tr.DueDate)
		if err != nil {
			return nil, err
		}

		tran.DueDate = sql.NullTime{
			Valid: true,
			Time:  dueDate,
		}
	}

	return tran, nil
}

const isoDateFormat = "2006-01-02"

// Effective dates are only valid for an exact day without time.
// We will parse them in the ISO 8601 (yyyy-mm-dd) format without time
//
// If `effDate` is emty, we will return a zero time instead
func parseEffectiveDate(effDate string) (time.Time, error) {
	if effDate != "" {
		parsed, err := time.Parse(isoDateFormat, effDate)
		if err != nil {
			return time.Time{}, err
		}

		return parsed, nil
	}

	return time.Time{}, nil
}
