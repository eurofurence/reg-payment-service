package main

import (
	"errors"
	"flag"
	"net/http"
	"path/filepath"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/repository/database"
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

	repo, err := database.NewMySQLProvider(conf.Database)
	if err != nil {
		logging.Ctx(ctx).Error(err)
	}

	if err := createFooTestBla(ctx, repo); err != nil {
		logging.Ctx(ctx).Error(err)
	}

	logging.Ctx(ctx).Debug("Setting up router")
	handler := server.CreateRouter(
		interaction.NewServiceInteractor(), conf.Service)

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
	var showHelp bool
	var configPath string

	flag.BoolVar(&showHelp, "h", false, "Displays the help text")
	flag.StringVar(&configPath, "config", "", "The path to a configuration file")
	// flag.StringVar(&configPath, "c", "", "The path to a configuration file")

	flag.Parse()

	if showHelp {
		flag.PrintDefaults()
		return nil, errHelpRequested
	}

	if configPath == "" {
		flag.PrintDefaults()
		return nil, errors.New("no config path was provided")
	}
	logging.NoCtx().Debug()

	fi, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return nil, errors.New("directory provided but yaml file was expected")
	}

	f, err := os.Open(filepath.Clean(configPath))
	if err != nil {
		return nil, err
	}

	return config.UnmarshalFromYamlConfiguration(f)
}

// TODO remove only for tests
func createFooTestBla(ctx context.Context, d database.Repository) error {
	f := entities.Foo{
		Age:  18,
		Name: "Hello World",
	}
	return d.CreateFoo(ctx, f)
}
