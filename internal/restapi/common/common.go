package common

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/eurofurence/reg-payment-service/internal/logging"
)

type ctxKeyRequestID int

const RequestIDKey ctxKeyRequestID = 0

func EncodeToJSON(w http.ResponseWriter, obj interface{}, logger logging.Logger) {
	enc := json.NewEncoder(w)

	if obj != nil {
		err := enc.Encode(obj)

		if err != nil {
			logger.Error("Could not encode response. [error]: %v", err)
		}
	}
}

func SendUnauthorizedResponse(w http.ResponseWriter, reqID string, logger logging.Logger) {
	SendResponseWithStatusAndMessage(w, http.StatusUnauthorized, reqID, AuthUnauthorizedMessage, logger)
}

func SendBadRequestResponse(w http.ResponseWriter, reqID string, logger logging.Logger) {
	SendResponseWithStatusAndMessage(w, http.StatusBadRequest, reqID, RequestParseErrorMessage, logger)
}

func SendStatusNotFoundResponse(w http.ResponseWriter, reqID string, logger logging.Logger) {
	SendResponseWithStatusAndMessage(w, http.StatusNotFound, reqID, TransactionIDNotFoundMessage, logger)
}

func SendInternalServerError(w http.ResponseWriter, reqID string, message APIErrorMessage, logger logging.Logger) {
	SendResponseWithStatusAndMessage(w, http.StatusInternalServerError, reqID, message, logger)
}

func SendResponseWithStatusAndMessage(w http.ResponseWriter, status int, reqID string, message APIErrorMessage, logger logging.Logger) {
	if reqID == "" {
		logger.Debug("request id is empty")
	}

	w.WriteHeader(status)
	apiErr := NewAPIError(reqID, message)
	EncodeToJSON(w, apiErr, logger)
}

func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return "00000000"
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return "ffffffff"
}
