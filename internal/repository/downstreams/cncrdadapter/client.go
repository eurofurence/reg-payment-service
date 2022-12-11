package cncrdadapter

import (
	"context"
	"errors"
	"fmt"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams"
	"net/http"
)

type Impl struct {
	client  aurestclientapi.Client
	baseUrl string
}

func New(cncrdBaseUrl string, fixedApiToken string) (CncrdAdapter, error) {
	if cncrdBaseUrl == "" {
		return nil, errors.New("service.provider_adapter not configured. This service cannot function without a provider adapter, though you can run it in local simulator mode for development")
	}

	client, err := downstreams.ClientWith(
		downstreams.ApiTokenRequestManipulator(fixedApiToken),
		"cncrd-adapter-breaker",
	)
	if err != nil {
		return nil, err
	}

	return &Impl{
		client:  client,
		baseUrl: cncrdBaseUrl,
	}, nil
}

func (i *Impl) CreatePaylink(ctx context.Context, request PaymentLinkRequestDto) (PaymentLinkDto, error) {
	url := fmt.Sprintf("%s/api/rest/v1/paylinks", i.baseUrl)
	bodyDto := PaymentLinkDto{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodPost, url, request, &response)
	return bodyDto, downstreams.ErrByStatus(err, response.Status)
}

func (i *Impl) GetPaylinkById(ctx context.Context, id uint) (PaymentLinkDto, error) {
	url := fmt.Sprintf("%s/api/rest/v1/paylinks/%d", i.baseUrl, id)
	bodyDto := PaymentLinkDto{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodGet, url, nil, &response)
	return bodyDto, downstreams.ErrByStatus(err, response.Status)
}
