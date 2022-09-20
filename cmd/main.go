package main

import (
	"database/sql"
	"errors"
	"flag"
	"net/http"
	"path/filepath"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/domain"
	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/mysql"
	"github.com/eurofurence/reg-payment-service/internal/repository/entities"
	"github.com/eurofurence/reg-payment-service/internal/server"

	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var errHelpRequested = errors.New("help text was requested")

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	logging.Ctx(ctx).Debug("loading configuration")
	conf, err := parseArgsAndReadConfig()
	if err != nil {
		if !errors.Is(err, errHelpRequested) {
			logging.Ctx(ctx).Fatal(err)
		}
		os.Exit(0)
	}

	repo := constructOrFail(ctx, func() (database.Repository, error) {
		return mysql.NewMySQLConnector(conf.Database)
	})

	if err := repo.Migrate(); err != nil {
		logging.Ctx(ctx).Fatal(err)
	}

	playDatabase(ctx, repo)

	i := constructOrFail(ctx, func() (interaction.Interactor, error) {
		return interaction.NewServiceInteractor(repo)
	})

	logging.Ctx(ctx).Debug("Setting up router")
	handler := server.CreateRouter(i, conf.Service)

	logging.Ctx(ctx).Debug("setting up server")
	srv := server.NewServer(ctx, &conf.Server, handler)

	go func() {
		<-sig
		defer cancel()
		logging.Ctx(ctx).Info("Stopping services now")

		tCtx, tcancel := context.WithTimeout(ctx, time.Second*5)
		defer tcancel()

		if err := srv.Shutdown(tCtx); err != nil {
			logging.Ctx(ctx).Fatal("Couldn't shutdown server gracefully")
		}
	}()

	logging.Ctx(ctx).Info("Running service on port ", conf.Server.Port)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logging.Ctx(ctx).Fatal("Server closed unexpectedly", err)
	}
}

func parseArgsAndReadConfig() (*config.Application, error) {
	var showHelp, migrate bool
	var configFilePath string

	// TODO parse flags into variable that is available to the main function.
	// Extrat the flags logic into different function.
	flag.BoolVar(&showHelp, "h", false, "Displays the help text")
	flag.StringVar(&configFilePath, "config", "", "The path to a configuration file")
	flag.BoolVar(&migrate, "migrate", false, "Performs database migrations before the service starts")

	flag.Parse()

	if showHelp {
		flag.PrintDefaults()
		return nil, errHelpRequested
	}

	if configFilePath == "" {
		flag.PrintDefaults()
		return nil, errors.New("no config file was provided")
	}
	logging.NoCtx().Debug()

	fi, err := os.Stat(configFilePath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return nil, errors.New("directory provided but yaml file was expected")
	}

	f, err := os.Open(filepath.Clean(configFilePath))
	if err != nil {
		return nil, err
	}

	return config.UnmarshalFromYamlConfiguration(f)
}

func constructOrFail[T any](ctx context.Context, constructor func() (T, error)) T {
	if constructor == nil {
		logging.Ctx(ctx).Fatal(errors.New("construction func must not be nil"))
	}

	t, err := constructor()
	if err != nil {
		logging.Ctx(ctx).Fatal(err)
	}

	return t

}

func playDatabase(ctx context.Context, r database.Repository) {
	// TODO use test function to test again mysql db.
	err := testCreateTransaction(ctx, r)
	if err != nil {
		if !errors.Is(err, mysql.ErrTransactionExists) {
			logging.Ctx(ctx).Fatal(err)
		}

		dt := defaultTransaction()

		dt.Comment = "Hello1"
		dt.TransactionStatusID = uint(domain.Tentative)
		err = testUpdateTransaction(ctx, r, dt)
		if err != nil {
			logging.Ctx(ctx).Fatal(err)
		}

	}

}

func testCreateTransaction(ctx context.Context, r database.Repository) error {
	return r.CreateTransaction(ctx, defaultTransaction())
}

func testUpdateTransaction(ctx context.Context, r database.Repository, tr entities.Transaction) error {
	return r.UpdateTransaction(ctx, tr)
}

func defaultTransaction() entities.Transaction {
	return entities.Transaction{
		TransactionID:     "123456789",
		DebitorID:         "1",
		TransactionTypeID: uint(domain.Due),
		TransactionType: entities.TransactionType{
			Description: "Due",
		},
		PaymentMethodID: uint(domain.Credit),
		PaymentMethod: entities.PaymentMethod{
			Description: "Credit",
		},
		TransactionStatusID: uint(domain.Pending),
		TransactionStatus: entities.TransactionStatus{
			Description: "Pending",
		},
		Amount: entities.Amount{
			ISOCurrency: "EUR",
			GrossCent:   19000,
			VatRate:     19.0,
		},
		Comment:       "Payment Noroth",
		EffectiveDate: sql.NullTime{},
		DueDate:       sql.NullTime{},
	}
}
