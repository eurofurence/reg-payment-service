package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/google/uuid"

	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

var RequestIDHeader = "X-Request-Id"

var ValidRequestIdRegex = regexp.MustCompile("^[0-9a-f]{8}$")

func createReqIdHandler(next http.Handler) func(w http.ResponseWriter, r *http.Request) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		reqUuidStr := r.Header.Get(RequestIDHeader)
		if !ValidRequestIdRegex.MatchString(reqUuidStr) {
			reqUuid, err := uuid.NewRandom()
			if err == nil {
				reqUuidStr = reqUuid.String()[:8]
			} else {
				// this should not normally ever happen, but continue with this fixed requestId
				reqUuidStr = "ffffffff"
			}
		}
		ctx := r.Context()
		newCtx := context.WithValue(ctx, common.CtxKeyRequestID{}, reqUuidStr)
		r = r.WithContext(newCtx)

		next.ServeHTTP(w, r)
	}
	return handlerFunc
}

// would not need this extra layer in the absence of parameters

func RequestIdMiddleware() func(http.Handler) http.Handler {
	middlewareCreator := func(next http.Handler) http.Handler {
		return http.HandlerFunc(createReqIdHandler(next))
	}
	return middlewareCreator
}
