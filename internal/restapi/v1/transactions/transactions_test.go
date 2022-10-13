package v1transactions

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/restapi/middleware"
)

func setupServer(t *testing.T) (string, func()) {
	router := chi.NewRouter()
	router.Use(middleware.RequestIdMiddleware())
	router.Use(middleware.LogRequestIdMiddleware())
	router.Use(middleware.CorsHeadersMiddleware())
	router.Route("/api/rest/v1", func(r chi.Router) {
		// TODO create mock of Interactor interface
		s, err := interaction.NewServiceInteractor(inmemory.NewInMemoryProvider())
		require.NoError(t, err)
		Create(r, s)
	})

	srv := httptest.NewServer(router)

	closeFunc := func() { srv.Close() }

	return srv.URL, closeFunc

}

func TestHandleTransactionsGet(t *testing.T) {
	url, close := setupServer(t)
	defer close()

	apiBasePath := fmt.Sprintf("%s/%s", url, "api/rest/v1")

	cl := http.DefaultClient

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s/%s", apiBasePath, "transactions/10"), nil)
	require.NoError(t, err)

	resp, err := cl.Do(req)

	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

}
