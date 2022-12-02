// This is an example file for handling web service routes.
// When implementing the real server, make sure to create an instance of `http.Server`,
// provide a valid configuration and apply a base context

package server

import (
	"fmt"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/interaction"
	"github.com/eurofurence/reg-payment-service/internal/restapi/middleware"
	v1health "github.com/eurofurence/reg-payment-service/internal/restapi/v1/health"
	v1transactions "github.com/eurofurence/reg-payment-service/internal/restapi/v1/transactions"

	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewServer(ctx context.Context, conf *config.ServerConfig, router http.Handler) *http.Server {

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", conf.BaseAddress, conf.Port),
		Handler:      router,
		ReadTimeout:  time.Second * time.Duration(conf.ReadTimeout),
		WriteTimeout: time.Second * time.Duration(conf.WriteTimeout),
		IdleTimeout:  time.Second * time.Duration(conf.IdleTimeout),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
}

func CreateRouter(i interaction.Interactor, conf config.SecurityConfig) chi.Router {
	router := chi.NewRouter()

	router.Use(chimiddleware.Recoverer)
	router.Use(middleware.RequestIdMiddleware())
	router.Use(middleware.LogRequestIdMiddleware())
	router.Use(middleware.CorsHeadersMiddleware())

	setupV1Routes(router, i, conf)

	return router
}

func setupV1Routes(router chi.Router, i interaction.Interactor, conf config.SecurityConfig) {
	v1health.Create(router)

	router.Route("/api/rest/v1", func(r chi.Router) {
		r.Use(middleware.TokenHandlerMiddleware(conf.Fixed.Api))
		v1transactions.Create(r, i)
	})
}
