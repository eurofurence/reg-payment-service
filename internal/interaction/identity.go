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

func NewIdentityManager(ctx context.Context, conf *config.SecurityConfig) *IdentityManager {
	manager := &IdentityManager{}
	if _, ok := ctx.Value(common.CtxKeyAPIKey{}).(string); ok {
		manager.isAPITokenCall = true
		return manager
	}

	if claims, ok := ctx.Value(common.CtxKeyClaims{}).(*common.AllClaims); ok {
		manager.subject = claims.Subject
		manager.groups = claims.Groups

		manager.isRegisteredUser = true

		for _, group := range claims.Groups {
			if group == conf.Oidc.AdminGroup {
				manager.isRegisteredUser = false
				manager.isAdmin = true
				break
			}
		}
	}

	return manager
}
