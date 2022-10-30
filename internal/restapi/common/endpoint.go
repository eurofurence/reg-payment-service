package common

import (
	"context"
	"net/http"

	"github.com/eurofurence/reg-payment-service/internal/logging"
)

type RequestHandler[Req any] func(r *http.Request) (*Req, error)
type ResponseHandler[Res any] func(res *Res, w http.ResponseWriter) error
type Endpoint[Req, Res any] func(ctx context.Context, request *Req) (*Res, error)

func CreateHandler[Req, Res any](ep Endpoint[Req, Res],
	requestHandler RequestHandler[Req],
	responseHandler ResponseHandler[Res]) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := GetRequestID(ctx)
		logger := logging.LoggerFromContext(ctx)

		if requestHandler == nil {
			logger.Fatal("No request provider supplied")
			return
		}

		if responseHandler == nil {
			logger.Fatal("No response handler supplied")
			return
		}

		request, err := requestHandler(r)
		if err != nil {
			logger.Error("An error occurred while parsing the request. [error]: %v", err)

			SendBadRequestResponse(w, reqID, logger)
		}

		response, err := ep(ctx, request)

		if err != nil {
			logger.Error("An error occurred during the request. [error]: %v", err)
			return
		}

		if err := responseHandler(response, w); err != nil {
			logger.Error("An error occurred during the handling of the response. [error]: %v", err)
		}
	})
}
