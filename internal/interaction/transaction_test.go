package interaction

import (
	"context"
	"errors"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/attendeeservice"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func TestMain(m *testing.M) {
	f, err := os.Open("../../docs/config.example.yaml")
	if err != nil {
		os.Exit(1)
	}
	_, err = config.UnmarshalFromYamlConfiguration(f)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())

}

func seedDB(db database.Repository, transactions []entities.Transaction) {
	for _, tr := range transactions {
		db.CreateTransaction(context.Background(), tr)
	}
}

func TestCreateTransaction(t *testing.T) {

	type args struct {
		paymentsChangedFunc   func(ctx context.Context, debitorId uint) error
		listRegistrationsFunc func(ctx context.Context) ([]int64, error)
		createPaylinkFunc     func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error)
		transaction           *entities.Transaction
		ctx                   context.Context
		seed                  []entities.Transaction
	}

	type expected struct {
		createPayLink bool
		err           error
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "should fail when currency is not allowed",
			args: args{
				transaction: &entities.Transaction{
					Amount: entities.Amount{
						ISOCurrency: "USD",
						GrossCent:   1000,
						VatRate:     19.0,
					},
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err: apierrors.NewBadRequest("invalid currency USD provided"),
			},
		},
		{
			name: "should fail without permissions",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:       1,
					TransactionID:   "1",
					TransactionType: entities.TransactionTypeDue,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   1000,
						VatRate:     19.0,
					},
				},
				ctx: context.Background(),
			},
			expected: expected{
				err: apierrors.NewForbidden("unable to determine the request permissions"),
			},
		},
		{
			name: "should fail when admin tries to create due transaction",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:       1,
					TransactionID:   "1",
					TransactionType: entities.TransactionTypeDue,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   1000,
						VatRate:     19.0,
					},
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err: apierrors.NewForbidden("Admin role is not allowed to create transactions of type due"),
			},
		},
		{
			name: "should fail when pending payments are present",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:       1,
					TransactionType: entities.TransactionTypePayment,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: adminCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusPending,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err: apierrors.NewConflict("There are pending payments for attendee 1"),
			},
		},
		{
			name: "should create due transaction in s2s call",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:       1,
					TransactionType: entities.TransactionTypeDue,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx:  apiKeyCtx(),
				seed: nil,
			},
			expected: expected{
				err: nil,
			},
		},
		{
			name: "should create transaction and request payment link",
			args: args{
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{Link: "pay.url.go/1"}, nil
				},
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodCredit,
					TransactionStatus: entities.TransactionStatusTentative,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: adminCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err:           nil,
				createPayLink: true,
			},
		},
		{
			name: "should fail validation for attendee transaction due to missing registration id",
			args: args{
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{Link: "pay.url.go/1"}, nil
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{2, 3}, nil
				},
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodCredit,
					TransactionStatus: entities.TransactionStatusTentative,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err: apierrors.NewForbidden("transactions for debitorID 1 may not be altered"),
			},
		},
		{
			name: "should fail validation for attendee transaction due to missing registration id",
			args: args{
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{Link: "pay.url.go/1"}, nil
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{2, 3}, nil
				},
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodCredit,
					TransactionStatus: entities.TransactionStatusTentative,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err: apierrors.NewForbidden("transactions for debitorID 1 may not be altered"),
			},
		},
		{
			name: "should fail validation for attendee transaction if status is incorrect",
			args: args{
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{Link: "pay.url.go/1"}, nil
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{1, 2, 3}, nil
				},
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodGift,
					TransactionStatus: entities.TransactionStatusTentative,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err: apierrors.NewForbidden("transaction is not eligible for requesting a payment link"),
			},
		},
		{
			name: "should fail validation for attendee transaction if status is incorrect",
			args: args{
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{Link: "pay.url.go/1"}, nil
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{1, 2, 3}, nil
				},
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodGift,
					TransactionStatus: entities.TransactionStatusTentative,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err: apierrors.NewForbidden("transaction is not eligible for requesting a payment link"),
			},
		},
		{
			name: "should fail creation of payment for attendee, when pending payments exist",
			args: args{
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{Link: "pay.url.go/1"}, nil
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{1, 2, 3}, nil
				},
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodCredit,
					TransactionStatus: entities.TransactionStatusTentative,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusPending,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err: apierrors.NewConflict("There are pending payments for attendee 1"),
			},
		},
		{
			name: "should create transaction and payment link for attendee",
			args: args{
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{Link: "pay.url.go/1"}, nil
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{1, 2, 3}, nil
				},
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodCredit,
					TransactionStatus: entities.TransactionStatusTentative,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   2000,
							VatRate:     19.0,
						}),
				},
			},
			expected: expected{
				err:           nil,
				createPayLink: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := &AttendeeServiceMock{
				ListMyRegistrationIdsFunc: tt.args.listRegistrationsFunc,
				PaymentsChangedFunc:       tt.args.paymentsChangedFunc,
			}

			ccm := &CncrdAdapterMock{
				CreatePaylinkFunc: tt.args.createPaylinkFunc,
			}

			db := inmemory.NewInMemoryProvider()
			seedDB(db, tt.args.seed)

			i, err := NewServiceInteractor(db, asm, ccm)
			require.NoError(t, err)

			res, err := i.CreateTransaction(tt.args.ctx, tt.args.transaction)

			if tt.expected.err != nil {
				require.EqualError(t, err, tt.expected.err.Error())
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				if tt.expected.createPayLink {
					require.NotEmpty(t, res.PaymentStartUrl)
				}
			}

		})
	}
}

func TestUpdateTransaction(t *testing.T) {
	type args struct {
		paymentsChangedFunc   func(ctx context.Context, debitorId uint) error
		listRegistrationsFunc func(ctx context.Context) ([]int64, error)
		createPaylinkFunc     func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error)
		transaction           *entities.Transaction
		ctx                   context.Context
		seed                  []entities.Transaction
	}

	type expected struct {
		err    error
		status entities.TransactionStatus
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "should fail if not admin or service token call",
			args: args{
				ctx: attendeeCtx(),
			},
			expected: expected{
				err: apierrors.NewForbidden("no permission to update transaction"),
			},
		},
		{
			name: "should return not found if original transaction could not be found in the database",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:     1,
					TransactionID: "12345",
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err: apierrors.NewNotFound("transaction 12345 for debitor 1 could not be found"),
			},
		},
		{
			name: "should return an error when admin is trying to delete a transaction which is older than three days",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionID:     "12345",
					TransactionStatus: entities.TransactionStatusDeleted,
					Deletion: entities.Deletion{
						Comment: "deleted for a reason",
						By:      "Kevin",
					},
				},
				seed: []entities.Transaction{
					{
						Model: gorm.Model{
							CreatedAt: time.Now().AddDate(0, 0, -3).Add(-time.Second * 10),
						},
						DebitorID:     1,
						TransactionID: "12345",
					},
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err: apierrors.NewForbidden("unable to flag transaction as deleted after 3 days"),
			},
		},
		{
			name: "should succesfully let admin delete the transaction when all conditions met",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionID:     "12345",
					TransactionStatus: entities.TransactionStatusDeleted,
					Deletion: entities.Deletion{
						Comment: "deleted for a reason",
						By:      "Kevin",
					},
				},
				seed: []entities.Transaction{
					{
						DebitorID: 1,
						Model: gorm.Model{
							CreatedAt: time.Now().AddDate(0, 0, -1),
						},
						TransactionID: "12345",
					},
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err:    nil,
				status: entities.TransactionStatusDeleted,
			},
		},
		{
			name: "should return error when trying to update a due transaction",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionID:     "12345",
					TransactionStatus: entities.TransactionStatusPending,
				},
				seed: []entities.Transaction{
					{
						DebitorID:       1,
						TransactionID:   "12345",
						TransactionType: entities.TransactionTypeDue,
					},
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err: apierrors.NewForbidden("cannot change the transaction of type due"),
			},
		},
		{
			name: "should return error if status change is not valid",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionID:     "12345",
					TransactionStatus: entities.TransactionStatusTentative,
				},
				seed: []entities.Transaction{
					{
						DebitorID:         1,
						TransactionID:     "12345",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusPending,
					},
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err: apierrors.NewForbidden("cannot change status from pending to tentative for transaction 12345"),
			},
		},
		{
			name: "should successfully update a transaction",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionID:     "12345",
					TransactionType:   entities.TransactionTypePayment,
					TransactionStatus: entities.TransactionStatusPending,
				},
				seed: []entities.Transaction{
					{
						DebitorID:         1,
						TransactionID:     "12345",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusTentative,
					},
				},
				ctx: apiKeyCtx(),
			},
			expected: expected{
				err:    nil,
				status: entities.TransactionStatusPending,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asm := &AttendeeServiceMock{
				ListMyRegistrationIdsFunc: tt.args.listRegistrationsFunc,
				PaymentsChangedFunc:       tt.args.paymentsChangedFunc,
			}

			ccm := &CncrdAdapterMock{
				CreatePaylinkFunc: tt.args.createPaylinkFunc,
			}

			db := inmemory.NewInMemoryProvider()
			seedDB(db, tt.args.seed)

			i, err := NewServiceInteractor(db, asm, ccm)
			require.NoError(t, err)

			err = i.UpdateTransaction(tt.args.ctx, tt.args.transaction)

			if tt.expected.err != nil {
				require.EqualError(t, err, tt.expected.err.Error())
			} else {
				require.NoError(t, err)
				tran, err := db.GetTransactionByTransactionIDAndType(tt.args.ctx, tt.args.transaction.TransactionID, tt.args.transaction.TransactionType)
				require.NoError(t, err)
				require.Equal(t, tt.expected.status, tran.TransactionStatus)
			}
		})
	}
	// TODO
}

func newTransaction(debID int64, tranID string,
	pType entities.TransactionType,
	method entities.PaymentMethod,
	status entities.TransactionStatus,
	amount entities.Amount,
) entities.Transaction {
	return entities.Transaction{
		DebitorID:         debID,
		TransactionID:     tranID,
		TransactionType:   pType,
		PaymentMethod:     method,
		TransactionStatus: status,
		Amount:            amount,
		Comment:           "Comment",
	}
}

func apiKeyCtx() context.Context {
	return context.WithValue(context.Background(), common.CtxKeyAPIKey{}, "123456")
}

func adminCtx() context.Context {
	return contextWithClaims(&common.AllClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "1234567890",
		},
		CustomClaims: common.CustomClaims{
			Global: common.GlobalClaims{
				Roles: []string{"admin"},
			},
		},
	})
}

func attendeeCtx() context.Context {
	return contextWithClaims(&common.AllClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "1234567890",
		},
		CustomClaims: common.CustomClaims{
			Global: common.GlobalClaims{
				Roles: []string{""},
			},
		},
	})
}

func contextWithClaims(claims *common.AllClaims) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeyClaims{}, claims)
	ctx = context.WithValue(ctx, common.CtxKeyToken{}, "token12345")

	return ctx
}

func TestIsValidStatusChange(t *testing.T) {
	type args struct {
		oldStatus entities.TransactionStatus
		newStatus entities.TransactionStatus
	}

	/*
		TransactionStatusTentative TransactionStatus = "tentative"
		TransactionStatusPending   TransactionStatus = "pending"
		TransactionStatusValid     TransactionStatus = "valid"
		TransactionStatusDeleted   TransactionStatus = "deleted"
	*/
	tests := []struct {
		name     string
		args     args
		expected bool
	}{
		{
			name: "should return true for tentative to pending",
			args: args{
				oldStatus: entities.TransactionStatusTentative,
				newStatus: entities.TransactionStatusPending,
			},
			expected: true,
		},
		{
			name: "should return true for tentative to deleted",
			args: args{
				oldStatus: entities.TransactionStatusTentative,
				newStatus: entities.TransactionStatusDeleted,
			},
			expected: true,
		},
		{
			name: "should return false for tentative to tentative",
			args: args{
				oldStatus: entities.TransactionStatusTentative,
				newStatus: entities.TransactionStatusTentative,
			},
			expected: false,
		},
		{
			name: "should return false for tentative to valid",
			args: args{
				oldStatus: entities.TransactionStatusTentative,
				newStatus: entities.TransactionStatusValid,
			},
			expected: false,
		},
		{
			name: "should return true for pending to valid",
			args: args{
				oldStatus: entities.TransactionStatusPending,
				newStatus: entities.TransactionStatusValid,
			},
			expected: true,
		},
		{
			name: "should return true for pending to deleted",
			args: args{
				oldStatus: entities.TransactionStatusPending,
				newStatus: entities.TransactionStatusDeleted,
			},
			expected: true,
		},
		{
			name: "should return false for pending to pending",
			args: args{
				oldStatus: entities.TransactionStatusPending,
				newStatus: entities.TransactionStatusPending,
			},
			expected: false,
		},
		{
			name: "should return false for pending to tentative",
			args: args{
				oldStatus: entities.TransactionStatusPending,
				newStatus: entities.TransactionStatusTentative,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curTran := entities.Transaction{
				TransactionStatus: tt.args.oldStatus,
			}
			newTran := entities.Transaction{
				TransactionStatus: tt.args.newStatus,
			}

			require.Equal(t, tt.expected, isValidStatusChange(curTran, newTran))
		})
	}

}

func TestValidateAttendeeTransaction(t *testing.T) {

	type args struct {
		repoMock         *RepositoryMock
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
				repoMock: &RepositoryMock{
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
				repoMock: &RepositoryMock{},
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
				repoMock: &RepositoryMock{
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
				repoMock: &RepositoryMock{
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
				repoMock: &RepositoryMock{
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
				repoMock: &RepositoryMock{
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
				repoMock: &RepositoryMock{
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
			i := tstServiceInteractor(tt.args.repoMock, tt.args.attendeeSvc, &CncrdAdapterMock{})
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
