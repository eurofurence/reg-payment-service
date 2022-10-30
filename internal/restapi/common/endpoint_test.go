package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type testRequest struct {
	Counter int
}

type testResponse struct {
	Counter int
}

func setupHandler(ep Endpoint[testRequest, testResponse], rh RequestHandler[testRequest], resph ResponseHandler[testResponse]) http.HandlerFunc {
	return CreateHandler(ep, rh, resph)
}

func TestCreateHandler(t *testing.T) {
	tReq := &testRequest{
		Counter: 0,
	}
	tRes := &testResponse{
		Counter: 0,
	}

	tests := []struct {
		name                    string
		endpoint                Endpoint[testRequest, testResponse]
		reqHandler              RequestHandler[testRequest]
		respHandler             ResponseHandler[testResponse]
		expectedError           error
		expectedRequestCounter  int
		expectedResponseCounter int
		expectedStatus          int
	}{
		{
			name:       "Should do nothing when no request handler was provided",
			reqHandler: nil,
			endpoint: func(ctx context.Context, request *testRequest, logger logging.Logger) (*testResponse, error) {
				return tRes, nil
			},
			respHandler: func(res *testResponse, w http.ResponseWriter) error {
				res.Counter++
				return nil
			},
			expectedRequestCounter:  0,
			expectedResponseCounter: 0,
			expectedStatus:          http.StatusInternalServerError,
		},
		{
			name: "Should do nothing when no response handler was provided",
			endpoint: func(ctx context.Context, request *testRequest, logger logging.Logger) (*testResponse, error) {
				return tRes, nil
			},
			reqHandler: func(r *http.Request) (*testRequest, error) {
				return tReq, nil
			},
			respHandler:             nil,
			expectedRequestCounter:  0,
			expectedResponseCounter: 0,
			expectedStatus:          http.StatusInternalServerError,
		},
		{
			name: "Should increase counter when all values are set",
			endpoint: func(ctx context.Context, request *testRequest, logger logging.Logger) (*testResponse, error) {
				return tRes, nil
			},
			reqHandler: func(r *http.Request) (*testRequest, error) {
				tReq.Counter++
				return tReq, nil
			},
			respHandler: func(res *testResponse, w http.ResponseWriter) error {
				res.Counter++
				return nil
			},
			expectedRequestCounter:  1,
			expectedResponseCounter: 1,
			expectedStatus:          http.StatusOK,
		},
		{
			name: "Should return bad request when request validation failed",
			endpoint: func(ctx context.Context, request *testRequest, logger logging.Logger) (*testResponse, error) {
				return tRes, nil
			},
			reqHandler: func(r *http.Request) (*testRequest, error) {
				tReq.Counter++
				return nil, errors.New("error error error")
			},
			respHandler: func(res *testResponse, w http.ResponseWriter) error {
				res.Counter++
				return nil
			},
			expectedRequestCounter:  1,
			expectedResponseCounter: 0,
			expectedStatus:          http.StatusBadRequest,
		},
		{
			name: "Should return internal server error when endpoint returns an error",
			endpoint: func(ctx context.Context, request *testRequest, logger logging.Logger) (*testResponse, error) {
				return nil, errors.New("Endpoint failed")
			},
			reqHandler: func(r *http.Request) (*testRequest, error) {
				tReq.Counter++
				return tReq, nil
			},
			respHandler: func(res *testResponse, w http.ResponseWriter) error {
				res.Counter++
				return nil
			},
			expectedRequestCounter:  1,
			expectedResponseCounter: 0,
			expectedStatus:          http.StatusInternalServerError,
		},
		{
			name: "Should return internal server error when Response Handler returns an error",
			endpoint: func(ctx context.Context, request *testRequest, logger logging.Logger) (*testResponse, error) {
				return tRes, nil
			},
			reqHandler: func(r *http.Request) (*testRequest, error) {
				tReq.Counter++
				return tReq, nil
			},
			respHandler: func(res *testResponse, w http.ResponseWriter) error {
				res.Counter++
				return errors.New("Error sending response")
			},
			expectedRequestCounter:  1,
			expectedResponseCounter: 1,
			expectedStatus:          http.StatusInternalServerError,
		},
		{
			name: "Should successfully return result when nothing failed",
			endpoint: func(ctx context.Context, request *testRequest, logger logging.Logger) (*testResponse, error) {
				return tRes, nil
			},
			reqHandler: func(r *http.Request) (*testRequest, error) {
				tReq.Counter++
				return tReq, nil
			},
			respHandler: func(res *testResponse, w http.ResponseWriter) error {
				res.Counter++
				require.NoError(t, json.NewEncoder(w).Encode(res))
				return errors.New("Error sending response")
			},
			expectedRequestCounter:  1,
			expectedResponseCounter: 1,
			expectedStatus:          http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tReq.Counter = 0
			tRes.Counter = 0
			router := chi.NewRouter()
			router.Get("/", setupHandler(tc.endpoint, tc.reqHandler, tc.respHandler))

			srv := httptest.NewServer(router)
			defer srv.Close()

			req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, fmt.Sprintf("%s/", srv.URL), nil)
			require.NoError(t, err)

			cl := &http.Client{
				Timeout: time.Second * 10,
			}

			resp, err := cl.Do(req)
			require.NoError(t, err)

			require.NotNil(t, resp)

			b, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			fmt.Println(string(b))

			require.Equal(t, tc.expectedRequestCounter, tReq.Counter)
			require.Equal(t, tc.expectedResponseCounter, tRes.Counter)

			require.Equal(t, tc.expectedStatus, resp.StatusCode)

		})
	}

}
