package authservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	aurestbreaker "github.com/StephanHCB/go-autumn-restclient-circuitbreaker/implementation/breaker"
	aurestclientapi "github.com/StephanHCB/go-autumn-restclient/api"
	auresthttpclient "github.com/StephanHCB/go-autumn-restclient/implementation/httpclient"
	aurestlogging "github.com/StephanHCB/go-autumn-restclient/implementation/requestlogging"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/repository/downstreams"
)

type Impl struct {
	client  aurestclientapi.Client
	baseUrl string
}

func newClient() (AuthService, error) {
	conf, err := config.GetApplicationConfig()
	if err != nil {
		return nil, err
	}

	requestManipulator := downstreams.CookiesOrAuthHeaderForwardingRequestManipulator(conf.Security)

	httpClient, err := auresthttpclient.New(0, nil, requestManipulator)
	if err != nil {
		return nil, err
	}

	requestLoggingClient := aurestlogging.New(httpClient)

	circuitBreakerClient := aurestbreaker.New(requestLoggingClient,
		"auth-service-breaker",
		10,
		2*time.Minute,
		30*time.Second,
		15*time.Second,
	)

	return &Impl{
		client:  circuitBreakerClient,
		baseUrl: conf.Security.Oidc.AuthService,
	}, nil
}

func errByStatus(err error, status int) error {
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized {
		return UnauthorizedError
	}
	if status >= 300 {
		return DownstreamError
	}
	return nil
}

func (i Impl) IsEnabled() bool {
	return true
}

func (i Impl) UserInfo(ctx context.Context) (UserInfoResponse, error) {
	url := fmt.Sprintf("%s/v1/userinfo", i.baseUrl)
	bodyDto := UserInfoResponse{}
	response := aurestclientapi.ParsedResponse{
		Body: &bodyDto,
	}
	err := i.client.Perform(ctx, http.MethodGet, url, nil, &response)
	return bodyDto, errByStatus(err, response.Status)
}
