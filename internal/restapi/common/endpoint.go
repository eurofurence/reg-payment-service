package common

import (
	"context"
	"net/http"

	"github.com/eurofurence/reg-payment-service/internal/apierrors"
	"github.com/eurofurence/reg-payment-service/internal/logging"
)

type RequestHandler[Req any] func(r *http.Request) (*Req, error)
type ResponseHandler[Res any] func(ctx context.Context, res *Res, w http.ResponseWriter) error
type Endpoint[Req, Res any] func(ctx context.Context, request *Req, logger logging.Logger) (*Res, error)

func CreateHandler[Req, Res any](endpoint Endpoint[Req, Res],
	requestHandler RequestHandler[Req],
	responseHandler ResponseHandler[Res]) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := GetRequestID(ctx)
		logger := logging.WithRequestID(ctx, reqID)

		defer func() {
			err := r.Body.Close()
			if err != nil {
				logger.Error("Error when closing the request body. [error]: %v", err)
			}
		}()

		if requestHandler == nil {
			logger.Error("No request handler supplied")
			SendInternalServerError(w, reqID, UnknownErrorMessage, logger, "")
			return
		}

		if responseHandler == nil {
			logger.Error("No response handler supplied")
			SendInternalServerError(w, reqID, UnknownErrorMessage, logger, "")
			return
		}

		request, err := requestHandler(r)
		if err != nil {
			logger.Error("An error occurred while parsing the request. [error]: %v", err)
			SendBadRequestResponse(w, reqID, logger, "")
			return
		}

		response, err := endpoint(ctx, request, logger)

		if err != nil {
			logger.Error("An error occurred during the request. [error]: %v", err)

			// check if the error is a `StatusError`
			if status := apierrors.AsAPIStatus(err); status != nil {

				// TODO enhance
				switch {
				case apierrors.IsBadRequestError(err):
					SendBadRequestResponse(w, reqID, logger, status.Status().Details)
				case apierrors.IsUnauthorizedError(err):
					SendUnauthorizedResponse(w, reqID, logger, status.Status().Details)
				case apierrors.IsForbiddenError(err):
					SendForbiddenResponse(w, reqID, logger, status.Status().Details)
				case apierrors.IsNotFoundError(err):
					SendStatusNotFoundResponse(w, reqID, logger, status.Status().Details)
				case apierrors.IsConflictError(err):
					SendConflictResponse(w, reqID, logger, status.Status().Details)
				case apierrors.IsInternalServerError(err):
					SendInternalServerError(w,
						reqID,
						APIErrorMessage(status.Status().Message),
						logger,
						status.Status().Details,
					)
				}

				return
			}

			SendInternalServerError(w, reqID, APIErrorMessage(err.Error()), logger, "")
			return
		}

		if err := responseHandler(ctx, response, w); err != nil {
			logger.Error("An error occurred during the handling of the response. [error]: %v", err)
			SendInternalServerError(w, reqID, UnknownErrorMessage, logger, "")
			return
		}

	})
}
