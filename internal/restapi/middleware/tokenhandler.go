package middleware

import (
	"net/http"

	"github.com/eurofurence/reg-payment-service/internal/logging"
)

const tokenHeaderKey = "X-API-TOKEN"

func tokenHandlerMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerToken := r.Header.Get(tokenHeaderKey)
		if token != headerToken {
			logging.NoCtx().Error("invalid token provided")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error_message": "request was unauthorized", "status": 401}`))

			return
		}

		next.ServeHTTP(w, r)
	})
}

func TokenHandlerMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return tokenHandlerMiddleware(token, next)
	}
}
