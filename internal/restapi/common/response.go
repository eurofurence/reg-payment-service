package common

// Response is the generic type for all responses.
type Response[T any] struct {
	Payload *T `json:"payload"`
}

// NewResponse creates a new response with the provided payload type.
func NewResponse[T any](payload *T) *Response[T] {
	return &Response[T]{
		Payload: payload,
	}
}
