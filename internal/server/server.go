// This is an example file for handling web service routes.
// When implementing the real server, make sure to create an instance of `http.Server`,
// provide a valid configuration and apply a base context

package server

import (
	"github.com/eurofurence/reg-payment-service/internal/restapi/middleware"
	v1health "github.com/eurofurence/reg-payment-service/internal/restapi/v1/health"

	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// quick and dirty method of handling the server. 
// do not export as global variable in productive code
var srv *http.Server

func Create() chi.Router {
	server := chi.NewRouter()

	server.Use(middleware.RequestIdMiddleware())
	server.Use(middleware.LogRequestIdMiddleware())
	server.Use(middleware.CorsHeadersMiddleware())

	v1health.Create(server)
	// add your controllers here
	return server
}

func Serve(ctx context.Context, server chi.Router) {
	const address = ":8080"
	srv = &http.Server{
		Addr:         address,
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("Failed...")
	}
}

func Shutdown(ctx context.Context) error {
	if srv != nil {
		return srv.Shutdown(ctx)
	}

	return nil
}
