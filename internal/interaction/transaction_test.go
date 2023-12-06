package interaction

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/attendeeservice"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
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

func compareDB(db database.Repository, expectedTransactions []entities.Transaction) error {
	if len(expectedTransactions) == 0 {
		return nil
	}

	storedTransactions, err := db.GetTransactionsByFilter(context.Background(), entities.TransactionQuery{})
	if err != nil {
		return err
	}

	if len(expectedTransactions) != len(storedTransactions) {
		return errors.New("unexpected number of transactions")
	}

	for i, expect := range expectedTransactions {
		stored := storedTransactions[i]
		if stored.TransactionID != expect.TransactionID ||
			stored.DebitorID != expect.DebitorID ||
			stored.TransactionType != expect.TransactionType ||
			stored.TransactionStatus != expect.TransactionStatus ||
			stored.PaymentMethod != expect.PaymentMethod ||
			stored.Amount.GrossCent != expect.Amount.GrossCent ||
			stored.Amount.VatRate != expect.Amount.VatRate {
			return fmt.Errorf("transaction %d does not match", i+1)
		}
	}

	return nil
}

func TestGetTransactionsForDebitor(t *testing.T) {
	type args struct {
		listRegistrationsFunc func(ctx context.Context) ([]int64, error)
		query                 entities.TransactionQuery
		ctx                   context.Context
		seed                  []entities.Transaction
	}

	type expected struct {
		transactions []entities.Transaction
		err          error
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "should return forbidden, when context doesn't contain any permissions",
			args: args{
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{1}, nil
				},
				query: entities.TransactionQuery{
					DebitorID: 1,
				},
				ctx:  context.Background(),
				seed: []entities.Transaction{},
			},
			expected: expected{
				transactions: nil,
				err:          apierrors.NewForbidden("unable to determine the request permissions"),
			},
		},
		{
			name: "should return error when attendee service call was not succesfull",
			args: args{
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return nil, errors.New("service call failed")
				},
				query: entities.TransactionQuery{
					DebitorID: 1,
				},
				ctx:  attendeeCtx(),
				seed: []entities.Transaction{},
			},
			expected: expected{
				transactions: nil,
				err:          errors.New("service call failed"),
			},
		},
		{
			name: "should return forbidden error, when attendee ID was not found in response",
			args: args{
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{2}, nil
				},
				query: entities.TransactionQuery{
					DebitorID: 1,
				},
				ctx:  attendeeCtx(),
				seed: []entities.Transaction{},
			},
			expected: expected{
				transactions: nil,
				err:          apierrors.NewForbidden("subject 1234567890 may not retrieve transactions for debitor 1"),
			},
		},
		{
			name: "should return debitor transactions, which are not deleted",
			args: args{
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{1}, nil
				},
				query: entities.TransactionQuery{
					DebitorID: 1,
				},
				ctx: attendeeCtx(),
				seed: []entities.Transaction{
					{
						DebitorID:         1,
						TransactionID:     "1",
						TransactionType:   entities.TransactionTypeDue,
						TransactionStatus: entities.TransactionStatusValid,
					},
					{
						DebitorID: 1,
						Model: gorm.Model{
							DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
						},
						TransactionID:     "2",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusDeleted,
					},
					{
						DebitorID:         1,
						TransactionID:     "3",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusValid,
					},
				},
			},
			expected: expected{
				transactions: []entities.Transaction{
					{
						DebitorID:         1,
						TransactionID:     "1",
						TransactionType:   entities.TransactionTypeDue,
						TransactionStatus: entities.TransactionStatusValid,
					},
					{
						DebitorID:         1,
						TransactionID:     "3",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusValid,
					},
				},
				err: nil,
			},
		},
		{
			name: "should return debitor transactions, which are deleted",
			args: args{
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{1}, nil
				},
				query: entities.TransactionQuery{
					DebitorID: 1,
				},
				ctx: adminCtx(),
				seed: []entities.Transaction{
					{
						DebitorID:         1,
						TransactionID:     "1",
						TransactionType:   entities.TransactionTypeDue,
						TransactionStatus: entities.TransactionStatusValid,
					},
					{
						DebitorID: 1,
						Model: gorm.Model{
							DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
						},
						TransactionID:     "2",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusDeleted,
					},
					{
						DebitorID:         1,
						TransactionID:     "3",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusValid,
					},
				},
			},
			expected: expected{
				transactions: []entities.Transaction{
					{
						DebitorID:         1,
						TransactionID:     "1",
						TransactionType:   entities.TransactionTypeDue,
						TransactionStatus: entities.TransactionStatusValid,
					},
					{
						DebitorID:         1,
						TransactionID:     "2",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusDeleted,
					},
					{
						DebitorID:         1,
						TransactionID:     "3",
						TransactionType:   entities.TransactionTypePayment,
						TransactionStatus: entities.TransactionStatusValid,
					},
				},
				err: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			db := inmemory.NewInMemoryProvider()
			seedDB(db, tt.args.seed)

			asm := &AttendeeServiceMock{
				ListMyRegistrationIdsFunc: tt.args.listRegistrationsFunc,
			}

			i, err := NewServiceInteractor(db, asm, &CncrdAdapterMock{})
			require.NoError(t, err)

			rt, err := i.GetTransactionsForDebitor(tt.args.ctx, tt.args.query)

			if tt.expected.err != nil {
				require.EqualError(t, err, tt.expected.err.Error())
				require.Nil(t, rt)
			} else {
				omitModelTransactions := make([]entities.Transaction, len(rt))
				for i, tr := range rt {
					tr := tr
					tr.Model = gorm.Model{}
					omitModelTransactions[i] = tr
				}
				require.NoError(t, err)

				for _, tr := range omitModelTransactions {
					require.Contains(t, tt.expected.transactions, tr)
				}

			}

		})
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
		expectedDbStateAfter  []entities.Transaction
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
			name: "should not fail when pending payments are present",
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
				err:           nil,
				createPayLink: true,
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
		{
			name: "should fail with incorrect passed transaction ID",
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
					TransactionID:     "123456",
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
				err:           apierrors.NewBadRequest("Invalid format for `TransactionID`"),
				createPayLink: false,
			},
		},
		{
			name: "should delete existing tentative payment with pay link if new dues transaction created",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypeDue,
					PaymentMethod:     entities.PaymentMethodInternal,
					TransactionStatus: entities.TransactionStatusValid,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: apiKeyCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusTentative,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
				},
				expectedDbStateAfter: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusDeleted, // this is the important part
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "3",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
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
				createPayLink: false,
			},
		},
		{
			name: "should not touch existing pending payment with pay link if new dues transaction created",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypeDue,
					PaymentMethod:     entities.PaymentMethodInternal,
					TransactionStatus: entities.TransactionStatusValid,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: apiKeyCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusPending,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
				},
				expectedDbStateAfter: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusPending,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "3",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
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
				createPayLink: false,
			},
		},
		{
			name: "should not touch existing valid payment with pay link if new dues transaction created",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypeDue,
					PaymentMethod:     entities.PaymentMethodInternal,
					TransactionStatus: entities.TransactionStatusValid,
					Amount: entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   2000,
						VatRate:     19.0,
					},
				},
				ctx: apiKeyCtx(),
				seed: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
				},
				expectedDbStateAfter: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "3",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
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
				createPayLink: false,
			},
		},
		{
			name: "should not delete existing tentative payment with pay link if new payment transaction created",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionType:   entities.TransactionTypePayment,
					PaymentMethod:     entities.PaymentMethodTransfer,
					TransactionStatus: entities.TransactionStatusValid,
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
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusTentative,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
				},
				expectedDbStateAfter: []entities.Transaction{
					newTransaction(1, "1",
						entities.TransactionTypeDue,
						entities.PaymentMethodInternal,
						entities.TransactionStatusValid,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "2",
						entities.TransactionTypePayment,
						entities.PaymentMethodCredit,
						entities.TransactionStatusTentative,
						entities.Amount{
							ISOCurrency: "EUR",
							GrossCent:   3000,
							VatRate:     19.0,
						}),
					newTransaction(1, "3",
						entities.TransactionTypePayment,
						entities.PaymentMethodTransfer,
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
				createPayLink: false,
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

			compareDB(db, tt.args.expectedDbStateAfter)
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
			name: "should return an error when admin is trying to delete a valid payment which is older than three days",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionID:     "12345",
					TransactionStatus: entities.TransactionStatusDeleted,
					TransactionType:   entities.TransactionTypePayment,
					Comment:           "deleted for a reason",
				},
				seed: []entities.Transaction{
					{
						Model: gorm.Model{
							CreatedAt: time.Now().AddDate(0, 0, -3).Add(-time.Second * 10),
						},
						DebitorID:         1,
						TransactionID:     "12345",
						TransactionStatus: entities.TransactionStatusValid,
						TransactionType:   entities.TransactionTypePayment,
					},
				},
				ctx: adminCtx(),
			},
			expected: expected{
				err: apierrors.NewForbidden("unable to flag valid transaction as deleted after 3 days, please book a compensating transaction instead"),
			},
		},
		{
			name: "should succesfully let admin delete a valid payment when all conditions met",
			args: args{
				transaction: &entities.Transaction{
					DebitorID:         1,
					TransactionID:     "12345",
					TransactionStatus: entities.TransactionStatusDeleted,
					TransactionType:   entities.TransactionTypePayment,
					Comment:           "deleted for a reason",
				},
				seed: []entities.Transaction{
					{
						DebitorID: 1,
						Model: gorm.Model{
							CreatedAt: time.Now().AddDate(0, 0, -1),
						},
						TransactionID:     "12345",
						TransactionStatus: entities.TransactionStatusValid,
						TransactionType:   entities.TransactionTypePayment,
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
				err: apierrors.NewForbidden("cannot change transactions of type due"),
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

func TestCreateTransactionForOutstandingDues(t *testing.T) {
	type args struct {
		paymentsChangedFunc   func(ctx context.Context, debitorId uint) error
		listRegistrationsFunc func(ctx context.Context) ([]int64, error)
		createPaylinkFunc     func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error)
		debitorID             int64
		method                entities.PaymentMethod
		ctx                   context.Context
		seed                  []entities.Transaction
	}

	type expected struct {
		createPayLink   bool
		expectedAmount  int64
		expectedComment string
		err             error
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "should return error when no valid transactions were found",
			args: args{
				debitorID: 10,
			},
			expected: expected{
				err: apierrors.NewNotFound("no valid dues found in order to initiate payment"),
			},
		},
		{
			name: "should return error when no outstanding dues exist",
			args: args{
				debitorID: 10,
				seed: []entities.Transaction{
					newTransaction(10, "1234", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusValid, entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   200_00,
						VatRate:     19.0,
					}),
					newTransaction(10, "1234", entities.TransactionTypePayment, entities.PaymentMethodCredit, entities.TransactionStatusValid, entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   200_00,
						VatRate:     19.0,
					}),
				},
			},
			expected: expected{
				err: apierrors.NewBadRequest("no outstanding dues for debitor"),
			},
		},
		{
			name: "should return error context does not contain a valid token",
			args: args{
				debitorID: 10,
				seed: []entities.Transaction{
					newTransaction(10, "1234", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusValid, entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   200_00,
						VatRate:     19.0,
					}),
				},
			},
			expected: expected{
				err: apierrors.NewForbidden("unable to determine the request permissions"),
			},
		},
		{
			name: "should return error when pending transactions exist",
			args: args{
				debitorID: 10,
				ctx:       attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(10, "1234", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusValid, entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   200_00,
						VatRate:     19.0,
					}),
					newTransaction(10, "1234", entities.TransactionTypePayment, entities.PaymentMethodCredit, entities.TransactionStatusPending, entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   200_00,
						VatRate:     19.0,
					}),
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{10}, nil
				},
			},
			expected: expected{
				err: apierrors.NewConflict("There are pending payments for attendee 10"),
			},
		},
		{
			name: "should create transaction with paylink and remaining amount",
			args: args{
				debitorID: 10,
				ctx:       attendeeCtx(),
				seed: []entities.Transaction{
					newTransaction(10, "1234", entities.TransactionTypeDue, entities.PaymentMethodCredit, entities.TransactionStatusValid, entities.Amount{
						ISOCurrency: "EUR",
						GrossCent:   200_00,
						VatRate:     19.0,
					}),
				},
				listRegistrationsFunc: func(ctx context.Context) ([]int64, error) {
					return []int64{10}, nil
				},
				paymentsChangedFunc: func(ctx context.Context, debitorId uint) error {
					return nil
				},
				createPaylinkFunc: func(ctx context.Context, request cncrdadapter.PaymentLinkRequestDto) (cncrdadapter.PaymentLinkDto, error) {
					return cncrdadapter.PaymentLinkDto{
						ReferenceId: "12345",
						Link:        "abc123",
					}, nil
				},
			},
			expected: expected{
				err:             nil,
				createPayLink:   true,
				expectedAmount:  200_00,
				expectedComment: "manually initiated credit card payment",
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

			if tt.args.ctx == nil {
				tt.args.ctx = context.TODO()
			}

			res, err := i.CreateTransactionForOutstandingDues(tt.args.ctx, tt.args.debitorID, tt.args.method)

			if tt.expected.err != nil {
				require.EqualError(t, err, tt.expected.err.Error())
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				if tt.expected.createPayLink {
					require.NotEmpty(t, res.PaymentStartUrl)
				}

				require.Equal(t, tt.expected.expectedAmount, res.Amount.GrossCent)
				require.Equal(t, tt.expected.expectedComment, res.Comment)
			}

		})
	}
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
	ctx := contextWithClaims(&common.AllClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "1234567890",
		},
		CustomClaims: common.CustomClaims{
			Groups: []string{"admin"},
		},
	})

	// TODO remove after 2FA is available
	// See reference https://github.com/eurofurence/reg-payment-service/issues/57
	return context.WithValue(ctx, common.CtxKeyAdminHeader{}, "available")
}

func attendeeCtx() context.Context {
	return contextWithClaims(&common.AllClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "1234567890",
		},
		CustomClaims: common.CustomClaims{
			Groups: []string{""},
		},
	})
}

func contextWithClaims(claims *common.AllClaims) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeyClaims{}, claims)
	ctx = context.WithValue(ctx, common.CtxKeyIdToken{}, "token12345")

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
			name: "should return true for tentative to valid",
			args: args{
				oldStatus: entities.TransactionStatusTentative,
				newStatus: entities.TransactionStatusValid,
			},
			expected: true,
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
		{
			name: "should return false for deleted to tentative",
			args: args{
				oldStatus: entities.TransactionStatusDeleted,
				newStatus: entities.TransactionStatusTentative,
			},
			expected: false,
		},
		{
			name: "should return false for deleted to pending",
			args: args{
				oldStatus: entities.TransactionStatusDeleted,
				newStatus: entities.TransactionStatusPending,
			},
			expected: false,
		},
		{
			name: "should return false for deleted to valid",
			args: args{
				oldStatus: entities.TransactionStatusDeleted,
				newStatus: entities.TransactionStatusValid,
			},
			expected: true,
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

func TestValidateTransactionID(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		inputID  string
		expected bool
	}{
		{
			name:     "should succeed with valid transaction ID",
			prefix:   "abc123",
			inputID:  "abc123-000302-0101-163055-1234",
			expected: true,
		},
		{
			name:     "should fail with invalid segments",
			prefix:   "abc123",
			inputID:  "abc1234-000302-0101-163055",
			expected: false,
		},
		{
			name:     "should fail with invalid prefix",
			prefix:   "abc123",
			inputID:  "abc1234-000302-0101-163055-1234",
			expected: false,
		},
		{
			name:     "should fail with letters in second segment",
			prefix:   "abc123",
			inputID:  "abc123-000ab302-0101-163055-1234",
			expected: false,
		},
		{
			name:     "should fail with less than six digits in second segemnt",
			prefix:   "abc123",
			inputID:  "abc123-00023-0101-163055-1234",
			expected: false,
		},
		{
			name:     "should fail with invalid time format",
			prefix:   "abc123",
			inputID:  "abc123-000302-0101-166100-1234", // 01.01T16:61:00 <- Minute 61 should fail
			expected: false,
		},
		{
			name:     "should fail with letters in last segment",
			prefix:   "abc123",
			inputID:  "abc123-000302-0101-163055-1234a", // 01.01T16:61:00 <- Minute 61 should fail
			expected: false,
		},
		{
			name:     "should fail with less than 4 numbers in last segment",
			prefix:   "abc123",
			inputID:  "abc123-000302-0101-163055-123", // 01.01T16:61:00 <- Minute 61 should fail
			expected: false,
		},
		{
			name:     "should fail with more than 4 numbers in last segment",
			prefix:   "abc123",
			inputID:  "abc123-000302-0101-163055-12345", // 01.01T16:61:00 <- Minute 61 should fail
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()

			require.Equal(t, tt.expected, validateTransactionID(tt.prefix, tt.inputID))
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
