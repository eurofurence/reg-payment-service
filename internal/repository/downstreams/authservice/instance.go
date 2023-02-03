package authservice

import (
	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-payment-service/internal/config"
)

var activeInstance AuthService

func New() (AuthService, error) {
	conf, err := config.GetApplicationConfig()
	if err != nil {
		return nil, err
	}

	if conf.Security.Oidc.AuthService != "" {
		instance, err := newClient()
		activeInstance = instance
		return instance, err
	} else {
		aulogging.Logger.NoCtx().Warn().Printf("security.oidc.auth_service not configured. Will skip online userinfo checks (not useful for production!)")
		activeInstance = newMock()
		return activeInstance, nil
	}
}

func CreateMock() Mock {
	instance := newMock()
	activeInstance = instance
	return instance
}

func Get() AuthService {
	return activeInstance
}
