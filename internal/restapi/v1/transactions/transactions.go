package v1transactions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"

	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

func Create(router chi.Router, i interaction.Interactor) {
	router.Get("/transactions",
		common.CreateHandler(
			MakeGetTransactionsEndpoint(i),
			getTransactionsRequestHandler,
			getTransactionsResponseHandler),
	)

	router.Post("/transactions",
		common.CreateHandler(
			MakeCreateTransactionEndpoint(i),
			createTransactionRequestHandler,
			createTransactionResponseHandler),
	)

	router.Put("/transactions/{id}",
		common.CreateHandler(
			MakeUpdateTransactionEndpoint(i),
			updateTransactionRequestHandler,
			updateTransactionResponseHandler),
	)

	router.Post("/transactions/initiate-payment",
		common.CreateHandler(
			MakeInitiatePaymentEndpoint(i),
			initiatePaymentRequestHandler,
			initiatePaymentResponseHandler,
		))
}

func MakeGetTransactionsEndpoint(i interaction.Interactor) common.Endpoint[GetTransactionsRequest, GetTransactionsResponse] {
	return func(ctx context.Context, request *GetTransactionsRequest, logger logging.Logger) (*GetTransactionsResponse, error) {
		txList, err := i.GetTransactionsForDebitor(ctx, entities.TransactionQuery{
			DebitorID:             request.DebitorID,
			TransactionIdentifier: request.TransactionIdentifier,
			EffectiveFrom:         request.EffectiveFrom,
			EffectiveBefore:       request.EffectiveBefore,
		})

		if err != nil {
			logger.Error("Could not get transactions. [error]: %v", err)
			return nil, err
		}

		response := GetTransactionsResponse{Payload: make([]Transaction, len(txList))}
		for i, tx := range txList {
			response.Payload[i] = ToV1Transaction(tx)
		}
		return &response, nil
	}
}

func MakeCreateTransactionEndpoint(i interaction.Interactor) common.Endpoint[CreateTransactionRequest, CreateTransactionResponse] {
	return func(ctx context.Context, request *CreateTransactionRequest, logger logging.Logger) (*CreateTransactionResponse, error) {
		tr := request.Transaction

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

		transaction, err := i.CreateTransaction(ctx, tran)
		if err != nil {
			return nil, err
		}

		return &CreateTransactionResponse{Transaction: ToV1Transaction(*transaction)}, nil
	}
}

func MakeUpdateTransactionEndpoint(i interaction.Interactor) common.Endpoint[UpdateTransactionRequest, UpdateTransactionResponse] {
	return func(ctx context.Context, request *UpdateTransactionRequest, logger logging.Logger) (*UpdateTransactionResponse, error) {

		return nil, nil
	}
}

func MakeInitiatePaymentEndpoint(i interaction.Interactor) common.Endpoint[InitiatePaymentRequest, InitiatePaymentResponse] {
	// TODO
	return func(ctx context.Context, request *InitiatePaymentRequest, logger logging.Logger) (*InitiatePaymentResponse, error) {
		return nil, nil
	}
}

func getTransactionsRequestHandler(r *http.Request) (*GetTransactionsRequest, error) {
	var req GetTransactionsRequest

	// debID is required (no, accounting will want to list all debitors for a certain period)
	debIDStr := r.URL.Query().Get("debitor_id")
	var debID int
	var err error
	if debIDStr != "" {
		debID, err = strconv.Atoi(debIDStr)
		if err != nil {
			return nil, err
		}
	}
	req.DebitorID = int64(debID)

	req.TransactionIdentifier = r.URL.Query().Get("transaction_identifier")

	efFrom, err := parseEffectiveDate(r.URL.Query().Get("effective_from"))
	if err != nil {
		return nil, err
	}

	req.EffectiveFrom = efFrom

	efBef, err := parseEffectiveDate(r.URL.Query().Get("effective_before"))
	if err != nil {
		return nil, err
	}

	req.EffectiveBefore = efBef

	return &req, nil
}

func getTransactionsResponseHandler(ctx context.Context, res *GetTransactionsResponse, w http.ResponseWriter) error {
	if res == nil {
		return common.ErrorFromMessage(common.TransactionReadErrorMessage)
	}

	if len(res.Payload) == 0 {
		reqID := common.GetRequestID(ctx)
		logger := logging.WithRequestID(ctx, reqID)
		common.SendStatusNotFoundResponse(w, reqID, logger, "")
		return nil
	}

	return json.NewEncoder(w).Encode(res)
}

func createTransactionRequestHandler(r *http.Request) (*CreateTransactionRequest, error) {
	var request CreateTransactionRequest
	err := json.NewDecoder(r.Body).Decode(&request.Transaction)

	if err != nil {
		return nil, err
	}

	if err := validateTransaction(&request.Transaction); err != nil {
		return nil, err
	}

	return &request, nil
}

func createTransactionResponseHandler(ctx context.Context, res *CreateTransactionResponse, w http.ResponseWriter) error {
	w.Header().Add(headers.Location, fmt.Sprintf("api/rest/v1/transactions/%s", res.Transaction.TransactionIdentifier))

	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(res)
}

func updateTransactionRequestHandler(r *http.Request) (*UpdateTransactionRequest, error) {
	return nil, nil
}

func updateTransactionResponseHandler(ctx context.Context, res *UpdateTransactionResponse, w http.ResponseWriter) error {
	return nil
}

func initiatePaymentRequestHandler(r *http.Request) (*InitiatePaymentRequest, error) {
	// TODO
	return nil, nil
}

func initiatePaymentResponseHandler(ctx context.Context, res *InitiatePaymentResponse, w http.ResponseWriter) error {
	// TODO
	return nil
}

// Effective dates are only valid for an exact day without time.
// We will parse them in the ISO 8601 (yyyy-mm-dd) format without time
//
// If `effDate` is emty, we will return a zero time instead
func parseEffectiveDate(effDate string) (time.Time, error) {
	if effDate != "" {
		parsed, err := time.Parse("2006-01-02", effDate)
		if err != nil {
			return time.Time{}, err
		}

		return parsed, nil
	}

	return time.Time{}, nil
}

func validateTransaction(t *Transaction) error {
	if t.DebitorID <= 0 {
		return fmt.Errorf("invalid debitor id supplied - DebitorID: %d", t.DebitorID)
	}

	if !t.TransactionType.IsValid() {
		return fmt.Errorf("invalid transaction type - TransactionType: %s", string(t.TransactionType))
	}

	if !t.Method.IsValid() {
		return fmt.Errorf("invalid payment method - Method: %s", string(t.Method))
	}

	if !t.Status.IsValid() || t.Status == entities.TransactionStatusDeleted {
		return fmt.Errorf("invalid transaction status - Method: %s", string(t.Status))
	}

	return nil
}
