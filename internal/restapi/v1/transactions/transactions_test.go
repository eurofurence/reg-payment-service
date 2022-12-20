package v1transactions

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/restapi/middleware"
)

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
			att, cncrd,
			logging.NewNoopLogger())

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

		i, err := interaction.NewServiceInteractor(tt.args.db, tt.args.att, tt.args.cncrd, logger)
		require.NoError(t, err)

		fn := MakeGetTransactionsEndpoint(i)

		resp, err := fn(ctx, req, logger)
		require.NoError(t, err)

		require.Len(t, resp.Payload, len(tt.expected.response.Payload))

		for _, tr := range resp.Payload {
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
