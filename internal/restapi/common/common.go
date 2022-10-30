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
	SendResponseWithStatusAndMessage(w, http.StatusUnauthorized, reqID, "Request was unauthorized (wrong or no api token, invalid, expired or no bearer token)", logger)
}

func SendBadRequestResponse(w http.ResponseWriter, reqID string, logger logging.Logger) {
	SendResponseWithStatusAndMessage(w, http.StatusBadRequest, reqID, "Request validation failed", logger)
}

func SendStatusNotFoundResponse(w http.ResponseWriter, reqID string, logger logging.Logger) {
	message := "No Transactions available for this debitor given the visibility rules, or an error occured"
	SendResponseWithStatusAndMessage(w, http.StatusNotFound, reqID, message, logger)
}

func SendInternalServerError(w http.ResponseWriter, reqID string, message string, logger logging.Logger) {
	SendResponseWithStatusAndMessage(w, http.StatusInternalServerError, reqID, message, logger)
}

func SendResponseWithStatusAndMessage(w http.ResponseWriter, status int, reqID, message string, logger logging.Logger) {
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
