package v1transactions

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/restapi/types"
)

type transactionHandler struct {
	interactor interaction.Interactor
}

func Create(router chi.Router, i interaction.Interactor) {
	handler := transactionHandler{
		interactor: i,
	}

	router.Get("/transactions/{debitor_id}", handler.handleTransactionsGet)
	router.Post("/transactions", handler.handleTransactionsPost)
}

func (t *transactionHandler) handleTransactionsGet(w http.ResponseWriter, r *http.Request) {
	dID := chi.URLParamFromCtx(r.Context(), "debitor_id")

	ctx := r.Context()

	result, err := t.interactor.GetTransactionsForDebitor(ctx, dID)
	if err != nil {
		logging.Ctx(ctx).Error(err)
		types.
			NewErrorResponse(err, http.StatusInternalServerError).
			EncodeToJSON(w)

		return
	}

	if err := types.NewResponse(result, http.StatusOK).EncodeToJSON(w); err != nil {
		logging.Ctx(ctx).Error(err)
	}
}

func (t *transactionHandler) handleTransactionsPost(w http.ResponseWriter, r *http.Request) {
	// TODO implement
}
