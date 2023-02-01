package downstreams

import (
	"context"
	"errors"
	"net/http"
	"time"

	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"
	"github.com/go-chi/chi/v5/middleware"

	aurestbreaker "github.com/StephanHCB/go-autumn-restclient-circuitbreaker/implementation/breaker"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	"github.com/go-http-utils/headers"

	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

// nolint
const apiKeyHeader = "X-Api-Key"

var (
	ErrDownStreamUnavailable = errors.New("downstream unavailable - see log for details")
)

func requestIDFromContext(ctx context.Context) string {
	if reqID, ok := ctx.Value(logging.RequestIdKey).(string); ok {
		return reqID
	}

	return "ffffffff"
}

func ApiTokenRequestManipulator(fixedApiToken string) aurestclientapi.RequestManipulatorCallback {
	return func(ctx context.Context, r *http.Request) {
		r.Header.Add(apiKeyHeader, fixedApiToken)
		r.Header.Add(middleware.RequestIDHeader, requestIDFromContext(ctx))
	}
}

func JwtForwardingRequestManipulator() aurestclientapi.RequestManipulatorCallback {
	return func(ctx context.Context, r *http.Request) {
		jwt, ok := ctx.Value(common.CtxKeyToken{}).(string)
		if ok {
			r.Header.Add(headers.Authorization, "Bearer "+jwt)
		}
		r.Header.Add(middleware.RequestIDHeader, requestIDFromContext(ctx))
	}
}

func ClientWith(requestManipulator aurestclientapi.RequestManipulatorCallback, circuitBreakerName string) (aurestclientapi.Client, error) {
	httpClient, err := auresthttpclient.New(0, nil, requestManipulator)
	if err != nil {
		return nil, err
	}

	requestLoggingClient := aurestlogging.New(httpClient)

	circuitBreakerClient := aurestbreaker.New(requestLoggingClient,
		circuitBreakerName,
		10,
		2*time.Minute,
		30*time.Second,
		15*time.Second,
	)

	return circuitBreakerClient, nil
}

func ErrByStatus(err error, status int) error {
	if err != nil {
		return err
	}
	if status >= 300 {
		return ErrDownStreamUnavailable
	}
	return nil
}
