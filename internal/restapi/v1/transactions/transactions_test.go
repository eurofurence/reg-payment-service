package v1transactions

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
	"github.com/eurofurence/reg-payment-service/internal/restapi/middleware"
)

type statusCodeResponseWriter struct {
	contents   []byte
	statusCode int
	called     bool
	headers    http.Header
}

func (s *statusCodeResponseWriter) Header() http.Header {
	if s.headers == nil {
		s.headers = make(http.Header)
	}
	return s.headers
}

func (s *statusCodeResponseWriter) Write(b []byte) (int, error) {
	s.contents = append(s.contents, b...)
	return len(b), nil
}

func (s *statusCodeResponseWriter) WriteHeader(statusCode int) {
	s.called = true
	s.statusCode = statusCode
}

//go:generate moq -pkg v1transactions -stub -out attendeeservice_moq_test.go ../../../repository/downstreams/attendeeservice/ AttendeeService
//go:generate moq -pkg v1transactions -stub -out cncrdadapter_moq_test.go ../../../repository/downstreams/cncrdadapter/ CncrdAdapter

func setupServer(t *testing.T, att *AttendeeServiceMock, cncrd *CncrdAdapterMock) (string, func()) {
	router := chi.NewRouter()
	router.Use(middleware.RequestIdMiddleware())
	router.Use(middleware.LogRequestIdMiddleware())
	router.Use(middleware.CorsHeadersMiddleware(nil))
	router.Route("/api/rest/v1", func(r chi.Router) {
		// TODO create mock of Interactor interface
		s, err := interaction.NewServiceInteractor(inmemory.NewInMemoryProvider(),
			att, cncrd)

		require.NoError(t, err)
		Create(r, s)
	})

	srv := httptest.NewServer(router)

	closeFunc := func() { srv.Close() }

	return srv.URL, closeFunc

}

func newTransaction(debID int64, tranID string,
	pType entities.TransactionType,
	method entities.PaymentMethod,
	status entities.TransactionStatus,
	effDate time.Time,
) entities.Transaction {
	return entities.Transaction{
		DebitorID:         debID,
		TransactionID:     tranID,
		TransactionType:   pType,
		PaymentMethod:     method,
		PaymentStartUrl:   "not set",
		TransactionStatus: status,
		Amount: entities.Amount{
			ISOCurrency: "EUR",
			GrossCent:   1900,
			VatRate:     19.0,
		},
		Comment: "Comment",
		EffectiveDate: sql.NullTime{
			Time:  effDate,
			Valid: true,
		},
	}
}

func newEffDate(t *testing.T, s string) time.Time {
	eff, err := parseEffectiveDate(s)
	require.NoError(t, err)

	return eff
}

func fillDefaultDBValues(t *testing.T, db database.Repository) {

	transactions := []entities.Transaction{
		newTransaction(1, "1234567890", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-01")),
		newTransaction(1, "1234567891", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-10")),
		newTransaction(1, "1234567892", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-20")),
		newTransaction(1, "1234567893", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-28")),
		newTransaction(1, "1234567894", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-29")),
		newTransaction(1, "1234567895", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-30")),

		newTransaction(2, "2234567890", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-01")),
		newTransaction(2, "2234567891", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-10")),
		newTransaction(2, "2234567892", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-20")),
		newTransaction(2, "2234567893", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-28")),
		newTransaction(2, "2234567894", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-29")),
		newTransaction(2, "2234567895", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-30")),
	}

	for _, tr := range transactions {
		db.CreateTransaction(context.Background(), tr)
	}
}

func TestHandleTransactions(t *testing.T) {

	type args struct {
		att       *AttendeeServiceMock
		cncrd     *CncrdAdapterMock
		db        database.Repository
		populator func(*testing.T, database.Repository)
		request   GetTransactionsRequest
	}

	type expected struct {
		response *GetTransactionsResponse
		err      error
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "Should return transactions for debitor ID",
			args: args{
				att:   &AttendeeServiceMock{},
				cncrd: &CncrdAdapterMock{},
				db:    inmemory.NewInMemoryProvider(),
				populator: func(t *testing.T, db database.Repository) {
					fillDefaultDBValues(t, db)
				},
				request: GetTransactionsRequest{
					DebitorID: 1,
				},
			},
			expected: expected{
				response: &GetTransactionsResponse{
					Payload: []Transaction{
						ToV1Transaction(newTransaction(1, "1234567890", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-01"))),
						ToV1Transaction(newTransaction(1, "1234567891", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-10"))),
						ToV1Transaction(newTransaction(1, "1234567892", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-20"))),
						ToV1Transaction(newTransaction(1, "1234567893", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-28"))),
						ToV1Transaction(newTransaction(1, "1234567894", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-29"))),
						ToV1Transaction(newTransaction(1, "1234567895", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-30"))),
					},
				},
				err: nil,
			},
		},
		{
			name: "Should return transaction for transaction identifier",
			args: args{
				att:   &AttendeeServiceMock{},
				cncrd: &CncrdAdapterMock{},
				db:    inmemory.NewInMemoryProvider(),
				populator: func(t *testing.T, db database.Repository) {
					fillDefaultDBValues(t, db)
				},
				request: GetTransactionsRequest{
					TransactionIdentifier: "1234567890",
				},
			},
			expected: expected{
				response: &GetTransactionsResponse{
					Payload: []Transaction{
						ToV1Transaction(newTransaction(1, "1234567890", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-01"))),
					},
				},
				err: nil,
			},
		},
		{
			name: "Should return transactions after effective date from",
			args: args{
				att:   &AttendeeServiceMock{},
				cncrd: &CncrdAdapterMock{},
				db:    inmemory.NewInMemoryProvider(),
				populator: func(t *testing.T, db database.Repository) {
					fillDefaultDBValues(t, db)
				},
				request: GetTransactionsRequest{
					EffectiveFrom: newEffDate(t, "2022-12-20"),
				},
			},
			expected: expected{
				response: &GetTransactionsResponse{
					Payload: []Transaction{
						ToV1Transaction(newTransaction(1, "1234567892", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-20"))),
						ToV1Transaction(newTransaction(1, "1234567893", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-28"))),
						ToV1Transaction(newTransaction(1, "1234567894", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-29"))),
						ToV1Transaction(newTransaction(1, "1234567895", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-30"))),

						ToV1Transaction(newTransaction(2, "2234567892", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-20"))),
						ToV1Transaction(newTransaction(2, "2234567893", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-28"))),
						ToV1Transaction(newTransaction(2, "2234567894", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-29"))),
						ToV1Transaction(newTransaction(2, "2234567895", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-30"))),
					},
				},
				err: nil,
			},
		},
		{
			name: "Should return transactions before effective date before",
			args: args{
				att:   &AttendeeServiceMock{},
				cncrd: &CncrdAdapterMock{},
				db:    inmemory.NewInMemoryProvider(),
				populator: func(t *testing.T, db database.Repository) {
					fillDefaultDBValues(t, db)
				},
				request: GetTransactionsRequest{
					EffectiveBefore: newEffDate(t, "2022-12-20"),
				},
			},
			expected: expected{
				response: &GetTransactionsResponse{
					Payload: []Transaction{
						ToV1Transaction(newTransaction(1, "1234567890", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-01"))),
						ToV1Transaction(newTransaction(1, "1234567891", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-10"))),

						ToV1Transaction(newTransaction(2, "2234567890", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-01"))),
						ToV1Transaction(newTransaction(2, "2234567891", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, newEffDate(t, "2022-12-10"))),
					},
				},
				err: nil,
			},
		},
		// {
		// 	name: "Should return transaction for debitor ID and transaction identifier",
		// },
		// {
		// 	name: "Should return transaction for debitor ID and effective date range",
		// },
		// {
		// 	name: "Should not return transactions for invalid data",
		// },
	}

	for _, tt := range tests {
		ctx := context.Background()

		tt.args.populator(t, tt.args.db)

		req := &GetTransactionsRequest{
			DebitorID:             tt.args.request.DebitorID,
			TransactionIdentifier: tt.args.request.TransactionIdentifier,
			EffectiveFrom:         tt.args.request.EffectiveFrom,
			EffectiveBefore:       tt.args.request.EffectiveBefore,
		}

		logger := logging.NewNoopLogger()

		i, err := interaction.NewServiceInteractor(tt.args.db, tt.args.att, tt.args.cncrd)
		require.NoError(t, err)

		fn := MakeGetTransactionsEndpoint(i)

		resp, err := fn(ctx, req, logger)
		require.NoError(t, err)

		require.Len(t, resp.Payload, len(tt.expected.response.Payload))

		for _, tr := range resp.Payload {
			// do not check for creation date
			tr.CreationDate = nil
			require.Contains(t, tt.expected.response.Payload, tr)
		}

	}
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

func TestGetTransactionsResponseHandler(t *testing.T) {
	type expected struct {
		err         error
		statusCode  int
		contentType reflect.Type
	}

	tests := []struct {
		name     string
		input    *GetTransactionsResponse
		expected expected
	}{
		{
			name:  "Should return error when response is nil",
			input: nil,
			expected: expected{
				err:         common.ErrorFromMessage(common.TransactionReadErrorMessage),
				statusCode:  0,
				contentType: nil,
			},
		},
		{
			name:  "Should return not found when payload doesn't contain transaction",
			input: &GetTransactionsResponse{},
			expected: expected{
				err:         nil,
				statusCode:  404,
				contentType: reflect.TypeOf(common.APIError{}),
			},
		},
		{
			name: "Should return transactions when payload is set",
			input: &GetTransactionsResponse{
				Payload: []Transaction{
					{
						DebitorID:             1,
						TransactionIdentifier: "abc",
					},
				},
			},
			expected: expected{
				err:         nil,
				statusCode:  0,
				contentType: reflect.TypeOf(GetTransactionsResponse{}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &statusCodeResponseWriter{}
			err := getTransactionsResponseHandler(context.Background(), tt.input, w)

			if tt.expected.err != nil {
				require.EqualError(t, tt.expected.err, err.Error())
				require.Equal(t, tt.expected.statusCode, w.statusCode)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected.statusCode, w.statusCode)

				rtIface := reflect.New(tt.expected.contentType).Interface()

				err := json.Unmarshal(w.contents, rtIface)
				require.NoError(t, err)
				require.NotZero(t, rtIface)

				if tt.expected.statusCode == 0 {
					if res, ok := rtIface.(*GetTransactionsResponse); ok {
						require.NotNil(t, res)
					} else {
						require.FailNow(t, "invalid type")
					}
				}
			}
		})
	}
}

func TestCreateTransactionRequestHandler(t *testing.T) {
	testTime := time.Now()
	type expected struct {
		err error
		req *CreateTransactionRequest
	}

	tests := []struct {
		name     string
		input    Transaction
		expected expected
	}{
		{
			name:  "Should return error when body is empty",
			input: Transaction{},
			expected: expected{
				err: errors.New("EOF"),
				req: nil,
			},
		},
		{
			name:  "Should return error when transaction contains invalid debitor ID",
			input: ToV1Transaction(newTransaction(0, "1230", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusTentative, testTime)),
			expected: expected{
				err: errors.New("invalid debitor id supplied - DebitorID: 0"),
				req: nil,
			},
		},
		{
			name:  "Should return error when transaction contains invalid type",
			input: ToV1Transaction(newTransaction(1, "1230", "Kevin", entities.PaymentMethodCredit, entities.TransactionStatusTentative, testTime)),
			expected: expected{
				err: errors.New("invalid transaction type - TransactionType: Kevin"),
				req: nil,
			},
		},
		{
			name:  "Should return error when transaction contains invalid method",
			input: ToV1Transaction(newTransaction(1, "1230", entities.TransactionTypeDue, "Kevin", entities.TransactionStatusTentative, testTime)),
			expected: expected{
				err: errors.New("invalid payment method - Method: Kevin"),
				req: nil,
			},
		},
		{
			name:  "Should return error when transaction contains invalid status",
			input: ToV1Transaction(newTransaction(1, "1230", entities.TransactionTypeDue, entities.PaymentMethodCredit, "Kevin", testTime)),
			expected: expected{
				err: errors.New("invalid transaction status - Status: Kevin"),
				req: nil,
			},
		},
		{
			name:  "Should return error when transaction status is deleted",
			input: ToV1Transaction(newTransaction(1, "1230", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusDeleted, testTime)),
			expected: expected{
				err: errors.New("invalid transaction status - Status: deleted"),
				req: nil,
			},
		},
		{
			name:  "Should return valid request when transaction is validated",
			input: ToV1Transaction(newTransaction(1, "1230", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusPending, testTime)),
			expected: expected{
				err: nil,
				req: &CreateTransactionRequest{
					Transaction: ToV1Transaction(newTransaction(1, "1230", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusPending, testTime)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "http://example.com/transaction", toTransactionRequestBody(tt.input))
			req, err := createTransactionRequestHandler(r)
			if tt.expected.err != nil {
				require.EqualError(t, err, tt.expected.err.Error())
				require.Nil(t, req)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected.req, req)
			}
		})
	}
}

func TestCreateTransactionResponseHandler(t *testing.T) {
	type expected struct {
		err        error
		statusCode int
		content    reflect.Type
	}

	tests := []struct {
		name     string
		input    *CreateTransactionResponse
		expected expected
	}{
		{
			name:  "should send status created and provide location header",
			input: &CreateTransactionResponse{Transaction: Transaction{TransactionIdentifier: "12345"}},
			expected: expected{
				err:        nil,
				statusCode: http.StatusCreated,
				content:    reflect.TypeOf(CreateTransactionResponse{}),
			},
		},
		{
			name:  "should return error when response is nil",
			input: nil,
			expected: expected{
				err:        errors.New("invalid response - cannot provide transaction information"),
				statusCode: 0,
				content:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &statusCodeResponseWriter{}
			err := createTransactionResponseHandler(context.Background(), tt.input, w)

			if tt.expected.err != nil {
				require.EqualError(t, tt.expected.err, err.Error())
				require.Equal(t, tt.expected.statusCode, w.statusCode)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected.statusCode, w.statusCode)

				rtIface := reflect.New(tt.expected.content).Interface()

				err := json.Unmarshal(w.contents, rtIface)
				require.NoError(t, err)
				require.NotZero(t, rtIface)

				if tt.expected.statusCode == http.StatusCreated {
					if res, ok := rtIface.(*CreateTransactionResponse); ok {
						require.NotNil(t, res)
					} else {
						require.FailNow(t, "invalid type")
					}
				}
			}
		})
	}
}

func TestUpdateTransactionRequestHandler(t *testing.T) {
	testTime := time.Now()

	type args struct {
		transactionID string
		transaction   Transaction
	}

	type expected struct {
		err error
		req *UpdateTransactionRequest
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "should return error when no transaction ID provided",
			args: args{
				transactionID: "",
				transaction:   ToV1Transaction(newTransaction(1, "1234", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusPending, testTime)),
			},
			expected: expected{
				err: errors.New("expected transaction id in url paramter, but received empty value"),
				req: nil,
			},
		},
		{
			name: "should return error when debitor id is not valid",
			args: args{
				transactionID: "1234",
				transaction:   ToV1Transaction(newTransaction(0, "1234", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusPending, testTime)),
			},
			expected: expected{
				err: errors.New("no debitor was provided in the request"),
				req: nil,
			},
		},
		{
			name: "should return error when paymenturl set",
			args: args{
				transactionID: "1234",
				transaction:   Transaction{DebitorID: 10, Status: entities.TransactionStatusPending, PaymentStartUrl: "12398"},
			},
			expected: expected{
				err: errors.New("updates on transactions may only change the status, payment processor information and due date"),
				req: nil,
			},
		},
		{
			name: "should return update transaction request when everything is successful",
			args: args{
				transactionID: "1234",
				transaction:   Transaction{DebitorID: 10, Status: entities.TransactionStatusPending},
			},
			expected: expected{
				err: nil,
				req: &UpdateTransactionRequest{
					Transaction: Transaction{
						DebitorID:             10,
						TransactionIdentifier: "1234",
						Status:                entities.TransactionStatusPending,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := httptest.NewRequest(http.MethodPost, "http://example.com/transaction/{id}", toTransactionRequestBody(tt.args.transaction))
			ctx := chi.NewRouteContext()
			ctx.URLParams.Add("id", tt.args.transactionID)

			r = r.WithContext(context.WithValue(context.TODO(), chi.RouteCtxKey, ctx))

			req, err := updateTransactionRequestHandler(r)
			if tt.expected.err != nil {
				require.EqualError(t, err, tt.expected.err.Error())
				require.Nil(t, req)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected.req, req)
			}
		})
	}
}

func toTransactionRequestBody(req Transaction) io.Reader {
	if reflect.ValueOf(req).IsZero() {
		return nil
	}

	b, _ := json.Marshal(req)
	return bytes.NewBuffer(b)
}
