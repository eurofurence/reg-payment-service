package common

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v4"

	"github.com/eurofurence/reg-payment-service/internal/logging"
)

type (
	CtxKeyRequestID struct{}
	CtxKeyToken     struct{}
	CtxKeyAPIKey    struct{}
	CtxKeyClaims    struct{}
)

type GlobalClaims struct {
	Name  string   `json:"name"`
	EMail string   `json:"email"`
	Roles []string `json:"roles"`
}

type CustomClaims struct {
	Global GlobalClaims `json:"global"`
}

type AllClaims struct {
	jwt.RegisteredClaims
	CustomClaims
}

func EncodeToJSON(w http.ResponseWriter, obj interface{}, logger logging.Logger) {
	enc := json.NewEncoder(w)

	if obj != nil {
		err := enc.Encode(obj)

		if err != nil {
			logger.Error("Could not encode response. [error]: %v", err)
		}
	}
}

func SendUnauthorizedResponse(w http.ResponseWriter, reqID string, logger logging.Logger, details string) {
	SendResponseWithStatusAndMessage(w, http.StatusUnauthorized, reqID, AuthUnauthorizedMessage, logger, details)
}

func SendBadRequestResponse(w http.ResponseWriter, reqID string, logger logging.Logger, details string) {
	SendResponseWithStatusAndMessage(w, http.StatusBadRequest, reqID, RequestParseErrorMessage, logger, details)
}

func SendStatusNotFoundResponse(w http.ResponseWriter, reqID string, logger logging.Logger, details string) {
	SendResponseWithStatusAndMessage(w, http.StatusNotFound, reqID, TransactionIDNotFoundMessage, logger, details)
}

func SendForbiddenResponse(w http.ResponseWriter, reqID string, logger logging.Logger, details string) {
	SendResponseWithStatusAndMessage(w, http.StatusForbidden, reqID, AuthForbiddenMessage, logger, details)
}

func SendConflictResponse(w http.ResponseWriter, reqID string, logger logging.Logger, details string) {
	SendResponseWithStatusAndMessage(w, http.StatusConflict, reqID, RequestConflictMessage, logger, details)
}

func SendInternalServerError(w http.ResponseWriter, reqID string, logger logging.Logger, details string) {
	SendResponseWithStatusAndMessage(w, http.StatusInternalServerError, reqID, InternalErrorMessage, logger, details)
}

func SendResponseWithStatusAndMessage(w http.ResponseWriter, status int, reqID string, message APIErrorMessage, logger logging.Logger, details string) {
	if reqID == "" {
		logger.Debug("request id is empty")
	}

	w.WriteHeader(status)

	var detailValues url.Values
	if details != "" {
		logger.Debug("Request was not successful: [error]: %s", details)
		detailValues = url.Values{"details": []string{details}}
	}

	apiErr := NewAPIError(reqID, message, detailValues)
	EncodeToJSON(w, apiErr, logger)
}

func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return "00000000"
	}
	if reqID, ok := ctx.Value(CtxKeyRequestID{}).(string); ok {
		return reqID
	}
	return "ffffffff"
}

// TODO SetRequestID func?
