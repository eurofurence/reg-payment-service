package interaction

import (
	"context"
	"github.com/eurofurence/reg-payment-service/internal/config"

	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

type IdentityManager struct {
	subject          string
	groups           []string
	isAdmin          bool
	isAPITokenCall   bool
	isRegisteredUser bool
}

func (i *IdentityManager) IsAdmin() bool {
	return i.isAdmin
}

func (i *IdentityManager) IsAPITokenCall() bool {
	return i.isAPITokenCall
}

func (i *IdentityManager) IsRegisteredUser() bool {
	return i.isRegisteredUser && i.subject != ""
}

func (i *IdentityManager) Subject() string {
	return i.subject
}

func NewIdentityManager(ctx context.Context) (*IdentityManager, error) {
	manager := &IdentityManager{}

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
			if group == conf.Security.Oidc.AdminGroup {
				manager.isRegisteredUser = false
				manager.isAdmin = true
				break
			}
		}
	}

	return manager, nil
}
