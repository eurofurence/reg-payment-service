package interaction

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"
)

const (
	transactionIDTimeFormat = "0102-150405" // MMDD-HHmmss
)

var (
	debRegex       = regexp.MustCompile(`^\d{6,}$`)
	randDigitRegex = regexp.MustCompile(`^\d{4}$`)
)

func (s *serviceInteractor) GetTransactionsForDebitor(ctx context.Context, query entities.TransactionQuery) ([]entities.Transaction, error) {
	logger := logging.LoggerFromContext(ctx)
	mgr, err := NewRBACValidator(ctx)
	if err != nil {
		return nil, err
	}

	if mgr.IsRegisteredUser() {
		regIDs, err := s.attendeeClient.ListMyRegistrationIds(ctx)
		if err != nil {
			logger.Error("could not call the attendee service. [error]: %v", err)
			return nil, err
		}

		if !containsDebitor(regIDs, query.DebitorID) {
			return nil, apierrors.NewForbidden(fmt.Sprintf("subject %s may not retrieve transactions for debitor %d", mgr.Subject(), query.DebitorID))
		}

		// will not return deleted transactions
		return s.store.GetTransactionsByFilter(ctx, query)
	}

	if mgr.IsAdmin() || mgr.IsAPITokenCall() {
		// return transactions in any state
		return s.store.GetAdminTransactionsByFilter(ctx, query)
	}

	return nil, apierrors.NewForbidden("unable to determine the request permissions")
}

func (s *serviceInteractor) CreateTransaction(ctx context.Context, tran *entities.Transaction) (*entities.Transaction, error) {
	logger := logging.LoggerFromContext(ctx)
	appConfig, err := config.GetApplicationConfig()
	if err != nil {
		return nil, err
	}

	// check if currency is allowed
	if !isCurrencyAllowed(appConfig.Service.AllowedCurrencies, tran.Amount.ISOCurrency) {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid currency %s provided", tran.Amount.ISOCurrency))
	}

	// generate a transaction ID if none exists
	if tran.TransactionID == "" {
		id, err := generateTransactionID(appConfig.Service.TransactionIDPrefix, tran)
		if err != nil {
			return nil, err
		}

		tran.TransactionID = id
	} else {
		// if a transaction ID is provided, it should be validated against the allowed format (starts with configured prefix, and has the correct number of segments etc.)
		// (because we allow create with transaction ID set)
		if !validateTransactionID(appConfig.Service.TransactionIDPrefix, tran.TransactionID) {
			return nil, apierrors.NewBadRequest("Invalid format for `TransactionID`")
		}
	}

	// default for effective date
	if !tran.EffectiveDate.Valid {
		tran.EffectiveDate = sql.NullTime{Time: time.Now(), Valid: true}
	}

	mgr, err := NewRBACValidator(ctx)
	if err != nil {
		return nil, err
	}

	if mgr.IsAdmin() || mgr.IsAPITokenCall() {
		return s.createTransactionWithElevatedAccess(ctx, tran, mgr)
	}

	if mgr.IsRegisteredUser() {
		// check if attendee is permitted to create this transaction
		if err := s.validateAttendeeTransaction(ctx, tran); err != nil {
			return nil, err
		}

		// create a transaction in the database
		if err := s.store.CreateTransaction(ctx, *tran); err != nil {
			return nil, err
		}

		// generate a payment link
		paymentLink, err := s.createPaymentLink(ctx, *tran)

		if err != nil {
			return nil, apierrors.NewInternalServerError(err.Error())
		}

		tran.PaymentStartUrl = paymentLink

		// update the payment link in the database
		s.store.UpdateTransaction(ctx, *tran, true)

		// inform the attendee service that there is a new payment in the database
		if err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID)); err != nil {
			logger.Error("error when calling the attendee service webhook. [error]: %v", err)
		}

		return tran, nil
	}

	return nil, apierrors.NewForbidden("unable to determine the request permissions")
}

func (s *serviceInteractor) CreateTransactionForOutstandingDues(ctx context.Context, debitorID int64) (*entities.Transaction, error) {
	appConfig, err := config.GetApplicationConfig()
	if err != nil {
		return nil, err
	}

	logging.LoggerFromContext(ctx)
	validTransactions, err := s.store.GetValidTransactionsForDebitor(ctx, debitorID)
	if err != nil {
		return nil, err
	}

	if len(validTransactions) == 0 {
		return nil, apierrors.NewNotFound("no valid dues found in order to initiate payment")
	}

	first := validTransactions[0]

	dues, err := s.store.QueryOutstandingDuesForDebitor(ctx, debitorID)
	if err != nil {
		return nil, err
	}

	if dues <= 0 {
		return nil, apierrors.NewBadRequest("no outstanding dues for debitor")
	}

	defaultCommentFunc := func() string {
		if strings.TrimSpace(appConfig.Service.DefaultPaymentComment) == "" {
			return "manually initiated credit card payment"
		}

		return appConfig.Service.DefaultPaymentComment
	}

	return s.CreateTransaction(ctx, &entities.Transaction{
		DebitorID:         debitorID,
		TransactionType:   entities.TransactionTypePayment,
		PaymentMethod:     entities.PaymentMethodCredit,
		TransactionStatus: entities.TransactionStatusTentative,
		Comment:           defaultCommentFunc(),
		Amount: entities.Amount{
			ISOCurrency: first.Amount.ISOCurrency,
			VatRate:     first.Amount.VatRate,
			GrossCent:   dues,
		},
	})
}

func (s *serviceInteractor) UpdateTransaction(ctx context.Context, tran *entities.Transaction) error {
	mgr, err := NewRBACValidator(ctx)
	if err != nil {
		return err
	}

	logger := logging.LoggerFromContext(ctx)

	if !mgr.IsAdmin() && !mgr.IsAPITokenCall() {
		return apierrors.NewForbidden("no permission to update transaction")
	}

	res, err := s.store.GetTransactionsByFilter(ctx, entities.TransactionQuery{
		DebitorID:             tran.DebitorID,
		TransactionIdentifier: tran.TransactionID,
	})

	if err != nil {
		return err
	}

	if len(res) == 0 {
		return apierrors.NewNotFound(
			fmt.Sprintf("transaction %s for debitor %d could not be found", tran.TransactionID, tran.DebitorID),
		)
	}

	curTran := res[0]

	if curTran.TransactionType == entities.TransactionTypeDue {
		return apierrors.NewForbidden("cannot change transactions of type due")
	}

	// check if a valid payment should be deleted or not by an admin
	if tran.TransactionStatus == entities.TransactionStatusDeleted &&
		curTran.TransactionStatus == entities.TransactionStatusValid &&
		curTran.TransactionType == entities.TransactionTypePayment &&
		mgr.IsAdmin() {

		logger.Warn("admin trying to delete valid payment %s", tran.TransactionID)

		// Within 3 calendar days of creation, for any transaction an admin may change
		// - status -> deleted
		const maxDaysForDeletion = 3.0
		days := time.Now().UTC().Sub(curTran.CreatedAt.UTC()).Hours() / 24.0

		if days > maxDaysForDeletion {
			return apierrors.NewForbidden("unable to flag valid transaction as deleted after 3 days, please book a compensating transaction instead")
		}

		// remember old values and who made the change
		curTran.Deletion = entities.Deletion{
			Status:  curTran.TransactionStatus, // previous status
			Comment: curTran.Comment,           // previous comment
			By:      mgr.Subject(),             // identity of deleting user
		}
		curTran.TransactionStatus = entities.TransactionStatusDeleted
		curTran.Comment = tran.Comment

		if err := s.store.DeleteTransaction(ctx, curTran); err != nil {
			return err
		}

		// inform the attendee service that a transaction was deleted
		if tran.TransactionType == entities.TransactionTypePayment {
			if err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID)); err != nil {
				// only log an error when the call was not successful but don't cause an internal server error
				logger.Error("error when calling the attendee service webhook. [error]: %v", err)
			}
		}

		logger.Warn("admin successfully deleted valid payment %s", tran.TransactionID)

		return nil

	}

	requireHistorization := false

	// Status changes:
	//    (The previous status is always historized, see the history field)
	if tran.TransactionStatus != curTran.TransactionStatus {
		if !isValidStatusChange(curTran, *tran) {
			return apierrors.NewForbidden(
				fmt.Sprintf("cannot change status from %s to %s for transaction %s",
					curTran.TransactionStatus,
					tran.TransactionStatus,
					tran.TransactionID,
				))
		}

		requireHistorization = true
	}

	if err := s.store.UpdateTransaction(ctx, *tran, requireHistorization); err != nil {
		return err
	}

	if tran.TransactionType == entities.TransactionTypePayment {
		// inform the attendee service that a transaction was updated
		if err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID)); err != nil {
			// only log an error when the call was not successful but don't cause an internal server error
			logger.Error("error when calling the attendee service webhook. [error]: %v", err)
		}
	}

	return nil
}

func (s *serviceInteractor) createTransactionWithElevatedAccess(
	ctx context.Context,
	tran *entities.Transaction,
	mgr *RBACValidator) (*entities.Transaction, error) {

	logger := logging.LoggerFromContext(ctx)

	if mgr.IsAdmin() && tran.TransactionType == entities.TransactionTypeDue {
		return nil, apierrors.NewForbidden("Admin role is not allowed to create transactions of type due")
	}

	if tran.TransactionType == entities.TransactionTypePayment {
		// for admin/API user, we do not check if pending payments are present
		// if we get a money or credit card transfer, we need to be able to book it or accounting will be incorrect
		// the money is in our bank, so we must book it, no matter if it makes any sense that we got the payment

		// We first make sure that we successfully persisted the transaction
		// in the DB before requesting a payment link if applicable
		err := s.store.CreateTransaction(ctx, *tran)
		if err != nil {
			return nil, err
		}

		// create payment link if
		// transaction_type=payment, method=credit, status=tentative
		if shouldRequestPaymentLink(tran) {

			paymentLink, err := s.createPaymentLink(ctx, *tran)
			if err != nil {
				return nil, apierrors.NewInternalServerError(err.Error())
			}

			tran.PaymentStartUrl = paymentLink

			// update the transaction and insert the payment link,
			// which was provided by the adapter service
			if err := s.store.UpdateTransaction(ctx, *tran, true); err != nil {
				return nil, err
			}
		}

		if err := s.attendeeClient.PaymentsChanged(ctx, uint(tran.DebitorID)); err != nil {
			// If the webhook was not successful, we write an error log and do not
			// return a 500 error response
			logger.Error("error when calling attendee service webhook. [error]: %v", err)
		}

		return tran, nil
	} else {
		// create new due transaction - must be created in status valid
		tran.TransactionStatus = entities.TransactionStatusValid
		err := s.store.CreateTransaction(ctx, *tran)
		if err != nil {
			return tran, err
		}

		// invalidate existing paylinks by marking their transactions deleted
		//
		// do not trigger payments changed webhook, because that may cause an update cycle
		// (the only one adding dues is the attendee service anyway, and we're only changing tentative payments here, which do not count yet anyway)
		err = s.invalidateTentativePayments(ctx, tran.DebitorID)
		if err != nil {
			return tran, err
		}

		return tran, nil
	}
}

func (s *serviceInteractor) invalidateTentativePayments(ctx context.Context, debitorID int64) error {
	transactions, err := s.store.GetTransactionsByFilter(ctx, entities.TransactionQuery{DebitorID: debitorID})
	if err != nil {
		return err
	}

	// delete existing transactions of type payment in status tentative (that is, paylinks)
	for _, tt := range transactions {
		if tt.TransactionType == entities.TransactionTypePayment && tt.TransactionStatus == entities.TransactionStatusTentative {
			// remember old values and who made the change
			tt.Deletion = entities.Deletion{
				Status:  tt.TransactionStatus, // previous status
				Comment: tt.Comment,           // previous comment
				By:      "internal",           // identity of deleting user
			}
			tt.TransactionStatus = entities.TransactionStatusDeleted
			tt.Comment = "voided paylink - dues have changed"

			if err := s.store.DeleteTransaction(ctx, tt); err != nil {
				return err
			}

			logger := logging.LoggerFromContext(ctx)
			logger.Warn("deleted outdated tentative payment %s", tt.TransactionID)
		}
	}

	return nil
}

func (s *serviceInteractor) validateAttendeeTransaction(ctx context.Context, newTransaction *entities.Transaction) error {
	logger := logging.LoggerFromContext(ctx)
	debitorIDs, err := s.attendeeClient.ListMyRegistrationIds(ctx)
	if err != nil {
		return err
	}

	if !containsDebitor(debitorIDs, newTransaction.DebitorID) {
		return apierrors.NewForbidden(fmt.Sprintf("transactions for debitorID %d may not be altered", newTransaction.DebitorID))
	}

	// User may only create transactions which are valid for requesting payment links
	if !shouldRequestPaymentLink(newTransaction) {
		return apierrors.NewForbidden("transaction is not eligible for requesting a payment link")
	}

	// Check if there are any pending transactions.
	pending, err := s.arePendingPaymentsPresent(ctx, newTransaction.DebitorID)

	if err != nil {
		logger.Error("could not retrieve pending payments for debitor %d - [error]: %v", newTransaction.DebitorID, err)
		return err
	}

	// do not create a new transaction if there is a pending payment.
	if pending {
		return apierrors.NewConflict(fmt.Sprintf("There are pending payments for attendee %d", newTransaction.DebitorID))
	}

	// We defined, that we only query transactions in status valid.
	currentTransactions, err := s.store.GetValidTransactionsForDebitor(ctx, newTransaction.DebitorID)
	if err != nil {
		return err
	}

	// in error case: 400
	// if partial payment || no outstanding dues
	if !s.isValidAttendeePayment(currentTransactions, newTransaction, logger) {
		return apierrors.NewBadRequest("no outstanding dues or partial payment")
	}

	return nil
}

func containsDebitor(debIDs []int64, debID int64) bool {
	for _, id := range debIDs {
		if id == debID {
			return true
		}
	}

	return false
}

func validateTransactionID(prefix, transactionID string) bool {
	if transactionID == "" {
		return false
	}

	segments := strings.Split(transactionID, "-")

	// we expect 5 segments (Time also contains a dash `-`)
	if len(segments) != 5 {
		return false
	}

	// The first segment contains the configured prefix
	if segments[0] != prefix {
		return false
	}

	// we expect at least 6 digits with leading zeros.
	if !debRegex.MatchString(segments[1]) {
		return false
	}

	// third and fourth segment needs to match the predefined time format.
	if _, err := time.Parse(transactionIDTimeFormat, fmt.Sprintf("%s-%s", segments[2], segments[3])); err != nil {
		return false
	}

	// last segment contains exactly 4 random digits
	if !randDigitRegex.MatchString(segments[4]) {
		return false
	}

	return true
}

func generateTransactionID(prefix string, tran *entities.Transaction) (string, error) {

	parsedTime := time.Now().UTC().Format(transactionIDTimeFormat)
	return fmt.Sprintf("%s-%06d-%s-%s", prefix, tran.DebitorID, parsedTime, randomDigits(4)), nil

}

var digitRunes = []rune("0123456789")

func randomDigits(count int) string {
	if count < 0 {
		return ""
	}

	res := make([]rune, count)

	for i := 0; i < count; i++ {
		rnd, err := rand.Int(rand.Reader, big.NewInt(int64(len(digitRunes))))
		if err != nil {
			return ""
		}

		res[i] = digitRunes[rnd.Int64()]

	}

	return string(res)
}

func (s *serviceInteractor) arePendingPaymentsPresent(ctx context.Context, debitorID int64) (bool, error) {
	transactions, err := s.store.GetTransactionsByFilter(ctx, entities.TransactionQuery{DebitorID: debitorID})
	if err != nil {
		return false, err
	}

	// check if there are any existing transactions of type payment, and return if they are
	// in pending or tentative state
	for _, tt := range transactions {
		switch tt.TransactionStatus {
		case entities.TransactionStatusPending, entities.TransactionStatusTentative:
			if tt.TransactionType == entities.TransactionTypePayment {
				return true, nil
			}
		}
	}

	// no pending payment transactions
	return false, nil
}

func (s *serviceInteractor) isValidAttendeePayment(curTransactions []entities.Transaction, newTran *entities.Transaction, logger logging.Logger) bool {
	var allDues int64
	var allPayments int64

	for _, t := range curTransactions {
		if t.TransactionType == entities.TransactionTypeDue {
			allDues += t.Amount.GrossCent
		} else if t.TransactionType == entities.TransactionTypePayment {
			allPayments += t.Amount.GrossCent
		}
	}

	// check if there are any outstanding dues
	// sum all status valid due transactions
	// subtract all valid payments
	// -> current_dues

	// can we have negative dues if we owe an attendee money?
	// - Yes, if the attendee overpaid due to currency conversion or made a mistake when making a bank/SWIFT transfer
	// - Also yes, if the attendee made a payment, then asked an admin to remove sponsor status
	if allDues <= 0 {
		logger.Info("No outstanding dues for attendee %d", newTran.DebitorID)
		return false
	}

	remaining := allDues - allPayments

	if remaining < 0 || newTran.Amount.GrossCent != remaining {
		// we do not allow partial payments from attendees
		// Admins or s2s calls will not use this validation logic
		logger.Info("rejected partial payment for attendee %d", newTran.DebitorID)
		return false
	}

	return true
}

func (s *serviceInteractor) createPaymentLink(ctx context.Context, tran entities.Transaction) (string, error) {
	response, err := s.cncrdClient.CreatePaylink(ctx, cncrdadapter.PaymentLinkRequestDto{
		ReferenceId: tran.TransactionID,
		DebitorId:   tran.DebitorID,
		Currency:    tran.Amount.ISOCurrency,
		VatRate:     tran.Amount.VatRate,
		AmountDue:   tran.Amount.GrossCent,
	})

	if err != nil {
		return "", apierrors.NewInternalServerError(err.Error())
	}

	return response.Link, nil
}

func isCurrencyAllowed(allowedCurrencies []string, isoCurrency string) bool {
	for _, cur := range allowedCurrencies {
		if strings.EqualFold(cur, isoCurrency) {
			return true
		}
	}

	return false
}

func shouldRequestPaymentLink(tran *entities.Transaction) bool {
	// Only the following condition is valid at the time,
	// in order to generate a payment link
	//
	// transaction_type=payment, method=credit, status=tentative
	return tran.TransactionType == entities.TransactionTypePayment &&
		tran.PaymentMethod == entities.PaymentMethodCredit &&
		tran.TransactionStatus == entities.TransactionStatusTentative
}

func isValidStatusChange(curTran, tran entities.Transaction) bool {
	// The only possible change is status for payments
	// * tentative -> pending (payment link has been used)
	// * tentative -> deleted (payment link has been deleted)
	// * tentative -> valid (payment link has been used, manual or automatic booking)
	// * pending -> valid (payment is confirmed by admin or by payment provider)
	// * pending -> deleted (payment has been deemed in error)

	if curTran.TransactionStatus == entities.TransactionStatusTentative {
		switch tran.TransactionStatus {
		case entities.TransactionStatusPending, entities.TransactionStatusDeleted, entities.TransactionStatusValid:
			return true
		}

		return false
	}

	if curTran.TransactionStatus == entities.TransactionStatusPending {
		switch tran.TransactionStatus {
		case entities.TransactionStatusValid, entities.TransactionStatusDeleted:
			return true
		}
	}

	return false

}
