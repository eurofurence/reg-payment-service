package v1transactions

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

func Create(router chi.Router, i interaction.Interactor) {
	router.Get("/transactions/{debitor_id}",
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
	return func(ctx context.Context, request *GetTransactionsRequest) (*GetTransactionsResponse, error) {
		logger := logging.LoggerFromContext(ctx)
		_, err := i.GetTransactionsForDebitor(ctx, request.DebitorID)

		if err != nil {
			logger.Error("Could not get transactions. [error]: %v", err)
			return nil, err
		}

		return nil, nil
	}
}

func MakeCreateTransactionEndpoint(i interaction.Interactor) common.Endpoint[CreateTransactionRequest, CreateTransactionResponse] {
	return func(ctx context.Context, request *CreateTransactionRequest) (*CreateTransactionResponse, error) {

		return nil, nil
	}
}

func MakeUpdateTransactionEndpoint(i interaction.Interactor) common.Endpoint[UpdateTransactionRequest, UpdateTransactionResponse] {
	return func(ctx context.Context, request *UpdateTransactionRequest) (*UpdateTransactionResponse, error) {

		return nil, nil
	}
}

func getTransactionsRequestHandler(r *http.Request) (*GetTransactionsRequest, error) {
	ctx := r.Context()

	var req GetTransactionsRequest

	// debID is required
	debID, err := strconv.Atoi(chi.URLParamFromCtx(ctx, "debitor_id"))
	if err != nil {
		return nil, err
	}

	req.DebitorID = int64(debID)

	req.TransactionIdentifier = chi.URLParamFromCtx(ctx, "transaction_identifier")

	efFrom, err := parseEffectiveDate(chi.URLParamFromCtx(ctx, "effective_from"))
	if err != nil {
		return nil, err
	}

	req.EffectiveFrom = efFrom

	efBef, err := parseEffectiveDate(chi.URLParamFromCtx(ctx, "effective_before"))
	if err != nil {
		return nil, err
	}

	req.EffectiveBefore = efBef

	return &req, nil
}

func getTransactionsResponseHandler(res *GetTransactionsResponse, w http.ResponseWriter) error {
	if res == nil {
		return common.ErrorFromMessage(common.TransactionDataInvalidMessage)
	}

	return nil
}

func createTransactionRequestHandler(r *http.Request) (*CreateTransactionRequest, error) {
	var request CreateTransactionRequest

	err := json.NewDecoder(r.Body).Decode(&request)

	if err != nil {
		return nil, err
	}

	return &request, nil
}

func createTransactionResponseHandler(res *CreateTransactionResponse, w http.ResponseWriter) error {

	return nil
}

func updateTransactionRequestHandler(r *http.Request) (*UpdateTransactionRequest, error) {
	return nil, nil
}

func updateTransactionResponseHandler(res *UpdateTransactionResponse, w http.ResponseWriter) error {
	return nil
}

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
