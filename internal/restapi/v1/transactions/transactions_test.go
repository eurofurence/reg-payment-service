package v1transactions

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/restapi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func setupServer() (string, func()) {
	router := chi.NewRouter()
	router.Use(middleware.RequestIdMiddleware())
	router.Use(middleware.LogRequestIdMiddleware())
	router.Use(middleware.CorsHeadersMiddleware())
	router.Route("/api/rest/v1", func(r chi.Router) {
		// TODO create mock of Interactor interface
		Create(r, interaction.NewServiceInteractor())
	})

	srv := httptest.NewServer(router)

	closeFunc := func() { srv.Close() }

	return srv.URL, closeFunc

}

func TestHandleTransactionsGet(t *testing.T) {
	url, close := setupServer()
	defer close()

	apiBasePath := fmt.Sprintf("%s/%s", url, "api/rest/v1")

	cl := http.DefaultClient

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", apiBasePath, "transactions/10"), nil)
	require.NoError(t, err)

	resp, err := cl.Do(req)
	require.NoError(t, err)

	require.NoError(t, io.NopCloser(resp.Body).Close())

}
