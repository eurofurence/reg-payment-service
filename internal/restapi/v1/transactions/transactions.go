package v1transactions

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

type transactionHandler struct {
	interactor interaction.Interactor
}

func Create(router chi.Router, i interaction.Interactor) {
	handler := transactionHandler{
		interactor: i,
	}

	router.Get("/transactions/{debitor_id}",
		common.CreateHandler(
			CreateGetTransactionsEndpoint(i),
			getTransactionsRequestHandler,
			getTransactionsResponseHandler),
	)

	router.Post("/transactions", handler.handleTransactionsPost)
}

func (t *transactionHandler) handleTransactionsPost(w http.ResponseWriter, r *http.Request) {
	// TODO implement
}

func CreateGetTransactionsEndpoint(i interaction.Interactor) common.Endpoint[GetTransactionsRequest, GetTransactionsResponse] {
	return func(ctx context.Context, request *GetTransactionsRequest) (*GetTransactionsResponse, error) {
		// TODO
		// i.GetTransactionsForDebitor()

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

func getTransactionsResponseHandler(response *GetTransactionsResponse, w http.ResponseWriter) error {
	if response == nil {
		// Send 404
		w.WriteHeader(http.StatusNotFound)
	}

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
