package interaction

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/config"

	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

type RBACValidator struct {
	subject          string
	groups           []string
	isAdmin          bool
	isAPITokenCall   bool
	isRegisteredUser bool
}

func (i *RBACValidator) IsAdmin() bool {
	return i.isAdmin
}

func (i *RBACValidator) IsAPITokenCall() bool {
	return i.isAPITokenCall
}

func (i *RBACValidator) IsRegisteredUser() bool {
	return i.isRegisteredUser && i.subject != ""
}

func (i *RBACValidator) Subject() string {
	return i.subject
}

func NewRBACValidator(ctx context.Context) (*RBACValidator, error) {
	manager := &RBACValidator{}

	conf, err := config.GetApplicationConfig()
	if err != nil {
		return nil, err
	}

	if _, ok := ctx.Value(common.CtxKeyAPIKey{}).(string); ok {
		manager.isAPITokenCall = true
		return manager, nil
	}

	if claims, ok := ctx.Value(common.CtxKeyClaims{}).(*common.AllClaims); ok {
		manager.subject = claims.Subject
		manager.groups = claims.Groups

		manager.isRegisteredUser = true

		for _, group := range claims.Groups {
			if group == conf.Security.Oidc.AdminGroup && hasValidAdminHeader(ctx) {
				manager.isRegisteredUser = false
				manager.isAdmin = true
				break
			}
		}
	}

	return manager, nil
}

// TODO remove after 2FA is available
// See reference https://github.com/eurofurence/reg-payment-service/issues/57
func hasValidAdminHeader(ctx context.Context) bool {
	adminHeaderValue, ok := ctx.Value(common.CtxKeyAdminHeader{}).(string)
	if !ok {
		return false
	}

	// legacy system implementation requires check against constant value "available"
	return adminHeaderValue == "available"
}
