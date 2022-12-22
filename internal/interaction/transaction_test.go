package interaction

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/attendeeservice"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
	"github.com/stretchr/testify/require"
)

//go:generate moq -pkg interaction -stub -out attendeeservice_moq_test.go ../repository/downstreams/attendeeservice/ AttendeeService
//go:generate moq -pkg interaction -stub -out cncrdadapter_moq_test.go ../repository/downstreams/cncrdadapter/ CncrdAdapter
//go:generate moq -pkg interaction -stub -out repository_moq_test.go ../repository/database Repository

func tstServiceInteractor(repo database.Repository, attendeeSvc attendeeservice.AttendeeService, adapter cncrdadapter.CncrdAdapter) *serviceInteractor {
	return &serviceInteractor{
		store:          repo,
		attendeeClient: attendeeSvc,
		cncrdClient:    adapter,
	}
}

func TestValidateAttendeeTransaction(t *testing.T) {

	type args struct {
		repoMock         RepositoryMock
		attendeeSvc      *AttendeeServiceMock
		inputTransaction entities.Transaction
	}

	type expected struct {
		shouldFail    bool
		expectedError error
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "Should return true for valid transaction",
			args: args{
				attendeeSvc: &AttendeeServiceMock{
					ListMyRegistrationIdsFunc: func(ctx context.Context) ([]int64, error) {
						return []int64{1}, nil
					},
				},
				repoMock: RepositoryMock{
					GetTransactionsByFilterFunc: func(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
					GetValidTransactionsForDebitorFunc: func(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
				},
				inputTransaction: tstDefaultTransaction(func(t *entities.Transaction) {
					t.TransactionType = entities.TransactionTypePayment
					t.TransactionStatus = entities.TransactionStatusTentative
				}),
			},
			expected: expected{
				shouldFail:    false,
				expectedError: nil,
			},
		},
		{
			name: "Should return forbiddin",
			args: args{
				attendeeSvc: &AttendeeServiceMock{
					ListMyRegistrationIdsFunc: func(ctx context.Context) ([]int64, error) {
						return []int64{2, 3, 4, 5}, nil
					},
				},
				repoMock: RepositoryMock{},
				inputTransaction: tstDefaultTransaction(func(t *entities.Transaction) {
					t.TransactionType = entities.TransactionTypePayment
					t.TransactionStatus = entities.TransactionStatusTentative
				}),
			},
			expected: expected{
				shouldFail:    true,
				expectedError: apierrors.NewForbidden("transactions for debitorID 1 may not be altered"),
			},
		},
		{
			name: "should return forbidden when transaction is not in a valid state",
			args: args{
				attendeeSvc: &AttendeeServiceMock{
					ListMyRegistrationIdsFunc: func(ctx context.Context) ([]int64, error) {
						return []int64{1}, nil
					},
				},
				repoMock: RepositoryMock{
					GetTransactionsByFilterFunc: func(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
					GetValidTransactionsForDebitorFunc: func(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
				},
				inputTransaction: tstDefaultTransaction(func(t *entities.Transaction) {
					t.TransactionType = entities.TransactionTypePayment
					t.TransactionStatus = entities.TransactionStatusPending
				}),
			},
			expected: expected{
				shouldFail:    true,
				expectedError: apierrors.NewForbidden("transaction is not eligible for requesting a payment link"),
			},
		},
		{
			name: "should return error when pending payments could not be retrieved",
			args: args{
				attendeeSvc: &AttendeeServiceMock{
					ListMyRegistrationIdsFunc: func(ctx context.Context) ([]int64, error) {
						return []int64{1}, nil
					},
				},
				repoMock: RepositoryMock{
					GetTransactionsByFilterFunc: func(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
						return nil, errors.New("test")
					},
					GetValidTransactionsForDebitorFunc: func(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
				},
				inputTransaction: tstDefaultTransaction(func(t *entities.Transaction) {
					t.TransactionType = entities.TransactionTypePayment
					t.TransactionStatus = entities.TransactionStatusTentative
				}),
			},
			expected: expected{
				shouldFail:    true,
				expectedError: errors.New("test"),
			},
		},
		{
			name: "should return conflict if pending payments exist",
			args: args{
				attendeeSvc: &AttendeeServiceMock{
					ListMyRegistrationIdsFunc: func(ctx context.Context) ([]int64, error) {
						return []int64{1}, nil
					},
				},
				repoMock: RepositoryMock{
					GetTransactionsByFilterFunc: func(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(func(t *entities.Transaction) {
							t.TransactionType = entities.TransactionTypePayment
							t.TransactionStatus = entities.TransactionStatusPending
						})}, nil
					},
					GetValidTransactionsForDebitorFunc: func(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
				},
				inputTransaction: tstDefaultTransaction(func(t *entities.Transaction) {
					t.TransactionType = entities.TransactionTypePayment
					t.TransactionStatus = entities.TransactionStatusTentative
				}),
			},
			expected: expected{
				shouldFail:    true,
				expectedError: apierrors.NewConflict("There are pending payments for attendee 1"),
			},
		},
		{
			name: "should return bad request for partial payments",
			args: args{
				attendeeSvc: &AttendeeServiceMock{
					ListMyRegistrationIdsFunc: func(ctx context.Context) ([]int64, error) {
						return []int64{1}, nil
					},
				},
				repoMock: RepositoryMock{
					GetTransactionsByFilterFunc: func(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
					GetValidTransactionsForDebitorFunc: func(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
				},
				inputTransaction: tstDefaultTransaction(func(t *entities.Transaction) {
					t.TransactionType = entities.TransactionTypePayment
					t.TransactionStatus = entities.TransactionStatusTentative
					t.Amount.GrossCent = 1900
				}),
			},
			expected: expected{
				shouldFail:    true,
				expectedError: apierrors.NewBadRequest("no outstanding dues or partial payment"),
			},
		},
		{
			name: "should return bad request if not outstanding dues",
			args: args{
				attendeeSvc: &AttendeeServiceMock{
					ListMyRegistrationIdsFunc: func(ctx context.Context) ([]int64, error) {
						return []int64{1}, nil
					},
				},
				repoMock: RepositoryMock{
					GetTransactionsByFilterFunc: func(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil)}, nil
					},
					GetValidTransactionsForDebitorFunc: func(ctx context.Context, debitorID int64) ([]entities.Transaction, error) {
						return []entities.Transaction{tstDefaultTransaction(nil), tstDefaultTransaction(func(t *entities.Transaction) {
							t.TransactionType = entities.TransactionTypePayment
							t.TransactionStatus = entities.TransactionStatusValid
						})}, nil
					},
				},
				inputTransaction: tstDefaultTransaction(func(t *entities.Transaction) {
					t.TransactionType = entities.TransactionTypePayment
					t.TransactionStatus = entities.TransactionStatusTentative
				}),
			},
			expected: expected{
				shouldFail:    true,
				expectedError: apierrors.NewBadRequest("no outstanding dues or partial payment"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := tstServiceInteractor(&tt.args.repoMock, tt.args.attendeeSvc, &CncrdAdapterMock{})
			err := i.validateAttendeeTransaction(context.TODO(), &tt.args.inputTransaction)
			if tt.expected.shouldFail {
				require.EqualError(t, err, tt.expected.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRandomDigits(t *testing.T) {
	tests := []struct {
		name              string
		inputCount        int
		expectedStringLen int
	}{
		{
			name:              "Should return string with len 4",
			inputCount:        4,
			expectedStringLen: 4,
		},
		{
			name:              "Should return empty string when len is negative",
			inputCount:        -1,
			expectedStringLen: 0,
		},
		{
			name:              "Should return empty string when len is zero",
			inputCount:        0,
			expectedStringLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := randomDigits(tt.inputCount)

			require.Len(t, res, tt.expectedStringLen)
			if tt.expectedStringLen > 0 {
				require.Regexp(t, regexp.MustCompile("[0-9]+"), res)
			}
		})
	}
}

func tstDefaultTransaction(f func(t *entities.Transaction)) entities.Transaction {
	res := entities.Transaction{
		DebitorID:         1,
		TransactionID:     "1234567890",
		TransactionType:   entities.TransactionTypeDue,
		PaymentMethod:     entities.PaymentMethodCredit,
		PaymentStartUrl:   "",
		TransactionStatus: entities.TransactionStatusValid,
		Amount: entities.Amount{
			ISOCurrency: "EUR",
			GrossCent:   2000,
			VatRate:     19.0,
		},
	}

	if f != nil {
		f(&res)
	}

	return res

}
