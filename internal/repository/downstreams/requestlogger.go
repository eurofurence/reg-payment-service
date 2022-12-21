package downstreams

import (
	"context"
	"time"

	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"

	"github.com/eurofurence/reg-payment-service/internal/logging"
)

// custom implementation because this service isn't using go-autumn-logging

type RequestLoggingImpl struct {
	Wrapped aurestclientapi.Client
}

func NewRequestLoggingWrapper(wrapped aurestclientapi.Client) aurestclientapi.Client {
	return &RequestLoggingImpl{
		Wrapped: wrapped,
	}
}

func (c *RequestLoggingImpl) Perform(ctx context.Context, method string, requestUrl string, requestBody interface{}, response *aurestclientapi.ParsedResponse) error {
	before := time.Now()
	err := c.Wrapped.Perform(ctx, method, requestUrl, requestBody, response)
	millis := time.Now().Sub(before).Milliseconds()
	if err != nil {
		logging.LoggerFromContext(ctx).Warn("downstream %s %s -> %d FAILED (%d ms): %s", method, requestUrl, response.Status, millis, err.Error())
	} else {
		logging.LoggerFromContext(ctx).Info("downstream %s %s -> %d OK (%d ms)", method, requestUrl, response.Status, millis)
	}
	return err
}
