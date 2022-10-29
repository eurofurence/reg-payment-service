package common

import "time"

// APIError is the generic return type for any Failure
// during endpoint operations
type APIError struct {
	RequestID string `json:"requestid"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// NewAPIError creates a new instance of the `APIError` which will be returned
// to the client if an operation fails
func NewAPIError(reqID, message string) *APIError {
	return &APIError{
		RequestID: reqID,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}
}
