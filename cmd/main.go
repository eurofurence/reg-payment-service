package main

import (
	"errors"
	"flag"
	"net/http"
	"path/filepath"

	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/authservice"

	"github.com/eurofurence/reg-payment-service/internal/repository/database/inmemory"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/attendeeservice"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams/cncrdadapter"

	"github.com/eurofurence/reg-payment-service/internal/config"
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

var (
	showHelp       bool
	migrate        bool
	configFilePath string
)

const (
	envDbPassword = "REG_SECRET_DB_PASSWORD"
	envApiToken   = "REG_SECRET_API_TOKEN"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	logging.SetupLogging("payment-service", false)
	logger := logging.NewLogger()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	logger.Debug("parsing command line flags")
	if err := parseArgs(logger); err != nil {
		if !errors.Is(err, errHelpRequested) {
			logger.Fatal("%v", err)
		}
		os.Exit(0)
	}

	logger.Debug("loading configuration")
	conf, err := readConfigFile(logger)
	if err != nil {
		logger.Fatal("%v", err)
	}

	logger.Debug("applying environment variable config overrides")
	envConfigOverrides(conf)

	repo := constructOrFail(ctx, logger, func() (database.Repository, error) {
		if conf.Database.Use == config.Mysql {
			return mysql.NewMySQLConnector(conf.Database, logger)
		} else if conf.Database.Use == config.Inmemory {
			return inmemory.NewInMemoryProvider(), nil
		} else {
			return nil, errors.New("invalid configuration")
		}
	})

	if migrate {
		if err := repo.Migrate(); err != nil {
			logger.Fatal("%v", err)
		}
	}

	//playDatabase(ctx, repo)

	attClient := constructOrFail(ctx, logger, func() (attendeeservice.AttendeeService, error) {
		return attendeeservice.New(conf.Service.AttendeeService)
	})

	ccClient := constructOrFail(ctx, logger, func() (cncrdadapter.CncrdAdapter, error) {
		return cncrdadapter.New(conf.Service.ProviderAdapter, conf.Security.Fixed.Api)
	})

	_ = constructOrFail(ctx, logger, func() (authservice.AuthService, error) {
		return authservice.New()
	})

	i := constructOrFail(ctx, logger, func() (interaction.Interactor, error) {
		return interaction.NewServiceInteractor(repo, attClient, ccClient)
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

func parseArgs(logger logging.Logger) error {
	flag.BoolVar(&showHelp, "h", false, "Displays the help text")
	flag.StringVar(&configFilePath, "config", "", "The path to a configuration file")
	flag.BoolVar(&migrate, "migrate", false, "Performs database migrations before the service starts")

	flag.Parse()

	if showHelp {
		flag.PrintDefaults()
		return errHelpRequested
	}

	if configFilePath == "" {
		flag.PrintDefaults()
		return errors.New("no config file was provided")
	}

	return nil
}

func readConfigFile(logger logging.Logger) (*config.Application, error) {
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

func envConfigOverrides(conf *config.Application) {
	if dbPassword := os.Getenv(envDbPassword); dbPassword != "" {
		conf.Database.Password = dbPassword
	}
	if apiToken := os.Getenv(envApiToken); apiToken != "" {
		conf.Security.Fixed.Api = apiToken
	}
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
