package v1transactions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/restapi/middleware"
)

func setupServer(t *testing.T) (string, func()) {
	router := chi.NewRouter()
	router.Use(middleware.RequestIdMiddleware())
	router.Use(middleware.LogRequestIdMiddleware())
	router.Use(middleware.CorsHeadersMiddleware(nil))
	router.Route("/api/rest/v1", func(r chi.Router) {
		// TODO create mock of Interactor interface
		s, err := interaction.NewServiceInteractor(inmemory.NewInMemoryProvider(), logging.NewNoopLogger())
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

func TestGetTransactionsRequestHandler(t *testing.T) {
	var testTime time.Time = time.Date(2022, time.January, 10, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name              string
		routeParamBuilder func(params url.Values)
		expectedError     error
		expectedResult    *GetTransactionsRequest
	}{
		{
			name: "Should return result when no debitor ID was provided",
			routeParamBuilder: func(params url.Values) {
				params.Add("transaction_identifier", "123456789")
				params.Add("effective_from", testTime.Format("2006-01-02"))
				params.Add("effective_before", testTime.Format("2006-01-02"))
			},
			expectedResult: &GetTransactionsRequest{
				DebitorID:             0,
				TransactionIdentifier: "123456789",
				EffectiveFrom:         testTime,
				EffectiveBefore:       testTime,
			},
		},
		{
			name: "Should return an error when debitor ID is not an int",
			routeParamBuilder: func(params url.Values) {
				params.Add("debitor_id", "Hans")
				params.Add("transaction_identifier", "123456789")
				params.Add("effective_from", testTime.Format("2006-01-02"))
				params.Add("effective_before", testTime.Format("2006-01-02"))
			},
			expectedError: errors.New("strconv.Atoi: parsing \"Hans\": invalid syntax"),
		},
		{
			// Effective dates must only be defined by an exact day without time.
			name: "Should return an error when effective from date is not in a correct format",
			routeParamBuilder: func(params url.Values) {
				params.Add("debitor_id", "10")
				params.Add("transaction_identifier", "123456789")
				params.Add("effective_from", testTime.Format("02.01.2006"))
				params.Add("effective_before", testTime.Format("2006-01-02"))
			},
			expectedError: errors.New("parsing time \"10.01.2022\" as \"2006-01-02\": cannot parse \"1.2022\" as \"2006\""),
		},
		{
			// Effective dates must only be defined by an exact day without time.
			name: "Should return an error when effective before date is not in a correct format",
			routeParamBuilder: func(params url.Values) {
				params.Add("debitor_id", "10")
				params.Add("transaction_identifier", "123456789")
				params.Add("effective_from", testTime.Format("2006-01-02"))
				params.Add("effective_before", testTime.Format("02.01.2006"))
			},
			expectedError: errors.New("parsing time \"10.01.2022\" as \"2006-01-02\": cannot parse \"1.2022\" as \"2006\""),
		},
		{
			name: "Should return result when only debitor ID is set",
			routeParamBuilder: func(params url.Values) {
				params.Add("debitor_id", "10")
			},
			expectedError: nil,
			expectedResult: &GetTransactionsRequest{
				DebitorID: 10,
			},
		},
		{
			name: "Should return result when debitor ID transaction ID is set",
			routeParamBuilder: func(params url.Values) {
				params.Add("debitor_id", "10")
				params.Add("transaction_identifier", "123456789")
			},
			expectedError: nil,
			expectedResult: &GetTransactionsRequest{
				DebitorID:             10,
				TransactionIdentifier: "123456789",
			},
		},
		{
			name: "Should return result when all values are set",
			routeParamBuilder: func(params url.Values) {
				params.Add("debitor_id", "10")
				params.Add("transaction_identifier", "123456789")
				params.Add("effective_from", "2022-01-10")
				params.Add("effective_before", "2022-01-10")
			},
			expectedError: nil,
			expectedResult: &GetTransactionsRequest{
				DebitorID:             10,
				TransactionIdentifier: "123456789",
				EffectiveFrom:         testTime,
				EffectiveBefore:       testTime,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			q := r.URL.Query()
			tc.routeParamBuilder(q)
			r.URL.RawQuery = q.Encode()

			transactionRequest, err := getTransactionsRequestHandler(r)
			if tc.expectedError != nil {
				require.Nil(t, transactionRequest)
				require.EqualError(t, err, tc.expectedError.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, transactionRequest)
			}
		})
	}
}
