package v1transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

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
}

func MakeGetTransactionsEndpoint(i interaction.Interactor) common.Endpoint[GetTransactionsRequest, GetTransactionsResponse] {
	return func(ctx context.Context, request *GetTransactionsRequest, logger logging.Logger) (*GetTransactionsResponse, error) {
		txList, err := i.GetTransactionsForDebitor(ctx, request.DebitorID)

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

		return nil, nil
	}
}

func MakeUpdateTransactionEndpoint(i interaction.Interactor) common.Endpoint[UpdateTransactionRequest, UpdateTransactionResponse] {
	return func(ctx context.Context, request *UpdateTransactionRequest, logger logging.Logger) (*UpdateTransactionResponse, error) {

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

	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		return err
	}

	return nil
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

	return nil
}

func updateTransactionRequestHandler(r *http.Request) (*UpdateTransactionRequest, error) {
	return nil, nil
}

func updateTransactionResponseHandler(ctx context.Context, res *UpdateTransactionResponse, w http.ResponseWriter) error {
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

	// Todo validation
	/*
			      required:
		        - amount
		        - status
		        - effective_date
	*/

	// 0 is not a valid debitor ID
	if t.DebitorID <= 0 {
		return fmt.Errorf("invalid debitor id supplied - DebitorID: %d", t.DebitorID)
	}

	if !t.TransactionType.IsValid() {
		return fmt.Errorf("invalid transaction type - TransactionType: %s", string(t.TransactionType))
	}

	if !t.Method.IsValid() {
		return fmt.Errorf("invalid payment method - Method: %s", string(t.Method))
	}

	// We cannot validate the status when creating a new transaction. Therefore the status cannot be required, right?
	//
	// This requires some more information @Jumpy

	// if !t.Status.IsValid() {
	// 	return fmt.Errorf("invalid transaction status - Method: %s", string(t.Status))
	// }

	return nil
}
