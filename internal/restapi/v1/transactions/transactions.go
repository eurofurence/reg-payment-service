package v1transactions

import (
	"fmt"
	"net/http"

	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/restapi/types"
	"github.com/go-chi/chi/v5"
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

// TODO implement a generic error result for all endpoints
type myFancyResult struct {
	ResultMessage string `json:"result_message"`
}

func (t *transactionHandler) handleTransactionsGet(w http.ResponseWriter, r *http.Request) {
	dID := chi.URLParam(r, "debitor_id")

	fmt.Println(dID)
	types.NewResult(&myFancyResult{
		ResultMessage: "This has been successful",
	}, 200).EncodeToJson(w)

}

func (t *transactionHandler) handleTransactionsPost(w http.ResponseWriter, r *http.Request) {
	// TODO implement
}
