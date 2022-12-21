package attendeeservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"

	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams"
)

type Impl struct {
	paymentsChangedClient     aurestclientapi.Client
	listMyRegistrationsClient aurestclientapi.Client
	baseUrl                   string
}

func New(attendeeServiceBaseUrl string, fixedApiToken string) (AttendeeService, error) {
	if attendeeServiceBaseUrl == "" {
		return nil, errors.New("service.attendee_service not configured. This service cannot function without the attendee service, though you can run it in inmemory database mode for development.")
	}

	paymentsChangedClient, err := downstreams.ClientWith(
		downstreams.ApiTokenRequestManipulator(fixedApiToken),
		"attendee-service-webhook-breaker",
	)
	if err != nil {
		return nil, err
	}

	listMyRegistrationsClient, err := downstreams.ClientWith(
		downstreams.JwtForwardingRequestManipulator(),
		"attendee-service-breaker",
	)
	if err != nil {
		return nil, err
	}

	return &Impl{
		paymentsChangedClient:     paymentsChangedClient,
		listMyRegistrationsClient: listMyRegistrationsClient,
		baseUrl:                   attendeeServiceBaseUrl,
	}, nil
}

type AttendeeIdList struct {
	Ids []int64 `json:"ids"`
}

func (i *Impl) PaymentsChanged(ctx context.Context, debitorId uint) error {
	url := fmt.Sprintf("%s/api/rest/v1/attendees/%d/payments-changed", i.baseUrl, debitorId)
	response := aurestclientapi.ParsedResponse{}
	err := i.paymentsChangedClient.Perform(ctx, http.MethodGet, url, nil, &response)
	return downstreams.ErrByStatus(err, response.Status)
}

func (i *Impl) ListMyRegistrationIds(ctx context.Context) ([]int64, error) {
	url := fmt.Sprintf("%s/api/rest/v1/attendees", i.baseUrl)
	bodyDto := AttendeeIdList{
		Ids: make([]int64, 0),
	}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.listMyRegistrationsClient.Perform(ctx, http.MethodGet, url, nil, &response)
	return bodyDto.Ids, downstreams.ErrByStatus(err, response.Status)
}
