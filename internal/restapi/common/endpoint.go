package common

import (
	"context"
	"net/http"

	"github.com/eurofurence/reg-payment-service/internal/logging"
)

type RequestProvider[Req any] func(r *http.Request) (*Req, error)
type ResponseHandler[Res any] func(res *Res, w http.ResponseWriter) error
type Endpoint[Req, Res any] func(ctx context.Context, request *Req) (*Res, error)

func CreateHandler[Request, Response any](ep Endpoint[Request, Response],
	rp RequestProvider[Request],
	rh ResponseHandler[Response]) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := GetRequestID(ctx)
		logger := logging.LoggerFromContext(ctx)

		if rp == nil {
			logger.Error("No request provider supplied")
			return
		}

		request, err := rp(r)
		if err != nil {
			logger.Error("An error occurred while parsing the request. [error]: %v", err)

			SendBadRequestResponse(w, reqID, logger)
		}

		response, err := ep(ctx, request)

		if err != nil {
			logger.Error("An error occurred during the request. [error]: %v", err)
			return
		}

		if err := rh(response, w); err != nil {
			logger.Error("An error occurred during the handling of the response. [error]: %v", err)
		}

	})
}
