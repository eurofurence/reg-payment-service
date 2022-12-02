package middleware

import (
	"net/http"

	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

// nolint
const tokenHeaderKey = "X-API-TOKEN"

func tokenHandlerMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerToken := r.Header.Get(tokenHeaderKey)
		logger := logging.LoggerFromContext(r.Context())
		if token != headerToken {
			logger.Error("invalid token provided")
			common.SendBadRequestResponse(w, common.GetRequestID(r.Context()), logger)
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
