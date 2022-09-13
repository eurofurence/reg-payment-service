package types

import (
	"encoding/json"
	"net/http"
)

type (
	Response[T any] struct {
		Payload    *T  `json:"payload"`
		StatusCode int `json:"-"`
	}

	ErrorResponse struct {
		ErrorMessage string `json:"error_message"`
		Status       int    `json:"status"`
	}
)

func NewErrorResponse(err error, status int) *ErrorResponse {
	return &ErrorResponse{
		ErrorMessage: err.Error(),
		Status:       status,
	}
}

func (r *ErrorResponse) EncodeToJSON(w http.ResponseWriter) error {
	w.WriteHeader(r.Status)
	return encodeToJSON(w, r)
}

func NewResponse[T any](payload T, statusCode int) *Response[T] {
	return &Response[T]{
		Payload:    &payload,
		StatusCode: statusCode,
	}
}

func (r *Response[T]) EncodeToJSON(w http.ResponseWriter) error {
	w.WriteHeader(r.StatusCode)
	return encodeToJSON(w, r)
}

func encodeToJSON[T any](w http.ResponseWriter, element *T) error {

	enc := json.NewEncoder(w)

	if element != nil {
		err := enc.Encode(element)
		return err
	}

	return nil
}
