package main

import (
	"database/sql"
	"errors"
	"flag"
	"net/http"
	"path/filepath"

	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/attendeeservice"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/entities"
	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
	"github.com/eurofurence/reg-payment-service/internal/repository/database/mysql"
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
	logger := logging.NewLogger()

	ctx = context.WithValue(ctx, logging.LoggerKey, logger)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	logger.Debug("loading configuration")
	conf, err := parseArgsAndReadConfig(logger)
	if err != nil {
		if !errors.Is(err, errHelpRequested) {
			logger.Fatal("%v", err)
		}
		os.Exit(0)
	}

	repo := constructOrFail(ctx, logger, func() (database.Repository, error) {
		if conf.Database.Use == config.Mysql {
			return mysql.NewMySQLConnector(conf.Database, logger)
		} else if conf.Database.Use == config.Inmemory {
			return inmemory.NewInMemoryProvider(), nil
		} else {
			return nil, errors.New("invalid configuration")
		}
	})

	if err := repo.Migrate(); err != nil {
		logger.Fatal("%v", err)
	}

	//playDatabase(ctx, repo)

	attClient := constructOrFail(ctx, logger, func() (attendeeservice.AttendeeService, error) {
		return attendeeservice.New(conf.Service.AttendeeService, conf.Security.Fixed.Api)
	})

	ccClient := constructOrFail(ctx, logger, func() (cncrdadapter.CncrdAdapter, error) {
		return cncrdadapter.New(conf.Service.ProviderAdapter, conf.Security.Fixed.Api)
	})

	i := constructOrFail(ctx, logger, func() (interaction.Interactor, error) {
		return interaction.NewServiceInteractor(repo, attClient, ccClient, logger)
	})

	logger.Debug("Setting up router")
	handler := server.CreateRouter(i, conf.Security)

	logger.Debug("setting up server")
	srv := server.NewServer(ctx, &conf.Server, handler)

	go func() {
		<-sig
		defer cancel()
		logger.Info("Stopping services now")

		tCtx, tcancel := context.WithTimeout(ctx, time.Second*5)
		defer tcancel()

		if err := srv.Shutdown(tCtx); err != nil {
			logger.Fatal("Couldn't shutdown server gracefully")
		}
	}()

	logger.Info("Running service on port %d", conf.Server.Port)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal("Server closed unexpectedly. [error]: %v", err)
	}
}

func parseArgsAndReadConfig(logger logging.Logger) (*config.Application, error) {
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

	conf, err := config.UnmarshalFromYamlConfiguration(f)
	if err != nil {
		return nil, err
	}

	if err := config.Validate(conf, logger.Warn); err != nil {
		return nil, err
	}

	return conf, nil
}

func constructOrFail[T any](ctx context.Context, logger logging.Logger, constructor func() (T, error)) T {
	const failMsg = "Construction failed. [error]: %v"
	if constructor == nil {
		logger.Fatal(failMsg, errors.New("construction func must not be nil"))
	}

	t, err := constructor()
	if err != nil {
		logger.Fatal(failMsg, err)
	}

	return t

}

func playDatabase(ctx context.Context, r database.Repository) {
	logger := logging.NewLogger()
	// TODO use test function to test again mysql db.
	err := testCreateTransaction(ctx, r)
	if err != nil {
		if !errors.Is(err, mysql.ErrTransactionExists) {
			logger.Fatal("An error occurred. [error]: %v", err)
		}

		dt := defaultTransaction()

		dt.Comment = "Hello1"
		dt.TransactionStatus = entities.TransactionStatusValid
		dt.DebitorID = 1
		err = testUpdateTransaction(ctx, r, dt)
		if err != nil {
			logger.Fatal("An error occurred. [error]: %v", err)
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
	ti, _ := time.Parse("2006-01-02", "2022-12-30")

	return entities.Transaction{
		TransactionID:     "234567895",
		DebitorID:         2,
		TransactionType:   entities.TransactionTypeDue,
		PaymentMethod:     entities.PaymentMethodCredit,
		TransactionStatus: entities.TransactionStatusPending,
		Amount: entities.Amount{
			ISOCurrency: "EUR",
			GrossCent:   19000,
			VatRate:     19.0,
		},
		Comment: "Payment Noroth",
		EffectiveDate: sql.NullTime{
			Valid: true,
			Time:  ti,
		},
		DueDate: sql.NullTime{},
	}
}
