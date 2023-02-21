package v1transactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

		eTran, err := ToTransactionEntity(request.Transaction)
		if err != nil {
			return nil, err
		}

		transaction, err := i.CreateTransaction(ctx, eTran)
		if err != nil {
			return nil, err
		}

		return &CreateTransactionResponse{Transaction: ToV1Transaction(*transaction)}, nil
	}
}

func MakeUpdateTransactionEndpoint(i interaction.Interactor) common.Endpoint[UpdateTransactionRequest, UpdateTransactionResponse] {
	return func(ctx context.Context, request *UpdateTransactionRequest, logger logging.Logger) (*UpdateTransactionResponse, error) {

		eTran, err := ToTransactionEntity(request.Transaction)
		if err != nil {
			return nil, err
		}

		err = i.UpdateTransaction(ctx, eTran)
		return nil, err
	}
}

func MakeInitiatePaymentEndpoint(i interaction.Interactor) common.Endpoint[InitiatePaymentRequest, InitiatePaymentResponse] {
	return func(ctx context.Context, request *InitiatePaymentRequest, logger logging.Logger) (*InitiatePaymentResponse, error) {
		logger.Debug("initiating payment for debitor %d", request.TransactionInitiator.DebitorID)
		res, err := i.CreateTransactionForOutstandingDues(ctx, request.TransactionInitiator.DebitorID)

		if err != nil {
			return nil, err
		}

		return &InitiatePaymentResponse{
			Transaction: ToV1Transaction(*res),
		}, nil

	}
}

func getTransactionsRequestHandler(r *http.Request) (*GetTransactionsRequest, error) {
	var req GetTransactionsRequest

	// debID is not required, because accounting will want to list transactions for all debitors for a certain period
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
		reqID := logging.GetRequestID(ctx)
		logger := logging.LoggerFromContext(ctx)
		common.SendStatusNotFoundResponse(w, reqID, logger, "")
		return nil
	}

	return json.NewEncoder(w).Encode(res)
}

var nowFunc = time.Now // needed for tests

func createTransactionRequestHandler(r *http.Request) (*CreateTransactionRequest, error) {
	var request CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&request.Transaction); err != nil {
		return nil, err
	}

	if request.Transaction.EffectiveDate == "" {
		// default to today
		request.Transaction.EffectiveDate = nowFunc().Format(isoDateFormat)
	}

	if err := validateTransaction(&request.Transaction, true); err != nil {
		return nil, err
	}

	return &request, nil
}

func createTransactionResponseHandler(ctx context.Context, res *CreateTransactionResponse, w http.ResponseWriter) error {
	if res == nil {
		return errors.New("invalid response - cannot provide transaction information")
	}
	w.Header().Add(headers.Location, fmt.Sprintf("api/rest/v1/transactions/%s", res.Transaction.TransactionIdentifier))

	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(res)
}

func updateTransactionRequestHandler(r *http.Request) (*UpdateTransactionRequest, error) {
	transactionID := chi.URLParam(r, "id")
	if transactionID == "" {
		return nil, errors.New("expected transaction id in url parameter, but received empty value")
	}

	var request UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&request.Transaction); err != nil {
		return nil, err
	}

	if request.Transaction.TransactionIdentifier == "" {
		request.Transaction.TransactionIdentifier = transactionID
	}
	if request.Transaction.TransactionIdentifier != transactionID {
		return nil, errors.New("transaction id in payload must match URL parameter")
	}

	if err := validateTransaction(&request.Transaction, false); err != nil {
		return nil, err
	}

	return &request, nil
}

func updateTransactionResponseHandler(ctx context.Context, _ *UpdateTransactionResponse, w http.ResponseWriter) error {
	// Write status header without content here
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func initiatePaymentRequestHandler(r *http.Request) (*InitiatePaymentRequest, error) {
	var payReq InitiatePaymentRequest

	if err := json.NewDecoder(r.Body).Decode(&payReq.TransactionInitiator); err != nil {
		return nil, err
	}

	if payReq.TransactionInitiator.DebitorID <= 0 {
		return nil, fmt.Errorf("invalid value %d for debitor. Value must be greater than zero", payReq.TransactionInitiator.DebitorID)
	}

	return &payReq, nil
}

func initiatePaymentResponseHandler(ctx context.Context, res *InitiatePaymentResponse, w http.ResponseWriter) error {
	if res == nil {
		return errors.New("invalid response - cannot provide transaction information")
	}
	w.Header().Add(headers.Location, fmt.Sprintf("api/rest/v1/transactions/%s", res.Transaction.TransactionIdentifier))

	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(res)
}

func validateTransaction(t *Transaction, forbidDeleted bool) error {
	if t.DebitorID <= 0 {
		return fmt.Errorf("invalid debitor id supplied - DebitorID: %d", t.DebitorID)
	}

	if !t.TransactionType.IsValid() {
		return fmt.Errorf("invalid transaction type - TransactionType: %s", url.QueryEscape(string(t.TransactionType)))
	}

	if !t.Method.IsValid() {
		return fmt.Errorf("invalid payment method - Method: %s", url.QueryEscape(string(t.Method)))
	}

	if !t.Status.IsValid() || t.Status == entities.TransactionStatusDeleted && forbidDeleted {
		return fmt.Errorf("invalid transaction status - Status: %s", url.QueryEscape(string(t.Status)))
	}

	_, err := parseEffectiveDate(t.EffectiveDate)
	if t.EffectiveDate == "" || err != nil {
		return fmt.Errorf("invalid effective date - EffectiveDate: %s", url.QueryEscape(t.EffectiveDate))
	}

	if t.Amount.Currency == "" {
		return errors.New("empty currency is not allowed")
	}

	if t.Amount.GrossCent == 0 {
		return errors.New("GrossCent cannot be 0, use delete instead")
	}

	return nil
}
