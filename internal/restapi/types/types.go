package types

import (
	"encoding/json"
	"net/http"
)

type (
	Result[T any] struct {
		Payload    *T `json:"payload"`
		StatusCode int
	}
)

func NewResult[T any](payload T, statusCode int) *Result[T] {
	return &Result[T]{
		Payload:    &payload,
		StatusCode: statusCode,
	}
}

func (r *Result[T]) EncodeToJson(w http.ResponseWriter) error {
	w.WriteHeader(r.StatusCode)
	enc := json.NewEncoder(w)

	if r.Payload != nil {
		err := enc.Encode(r.Payload)
		return err
	}

	return nil
}
