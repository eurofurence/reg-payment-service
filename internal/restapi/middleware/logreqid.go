package middleware

import (
	"github.com/eurofurence/reg-payment-service/internal/logging"

	"net/http"
)

func logRequestIdHandler(next http.Handler) func(w http.ResponseWriter, r *http.Request) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		// example to log a request id
		ctx := r.Context()
		newCtx := logging.CreateContextWithLoggerForRequestId(ctx, GetRequestID(ctx))
		r = r.WithContext(newCtx)

		next.ServeHTTP(w, r)
	}
	return handlerFunc
}

// would not need this extra layer in the absence of parameters

func LogRequestIdMiddleware() func(http.Handler) http.Handler {
	middlewareCreator := func(next http.Handler) http.Handler {
		return http.HandlerFunc(logRequestIdHandler(next))
	}
	return middlewareCreator
}
