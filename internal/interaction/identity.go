package interaction

import (
	"context"

	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

type IdentityManager struct {
	subject        string
	roles          []string
	isAdmin        bool
	isAPITokenCall bool
	isAttendee     bool
}

func (i *IdentityManager) IsAdmin() bool {
	return i.isAdmin
}

func (i *IdentityManager) IsAPITokenCall() bool {
	return i.isAPITokenCall
}

func (i *IdentityManager) IsAttendee() bool {
	return i.isAttendee && i.subject != ""
}

func NewIdentityManager(ctx context.Context) *IdentityManager {
	manager := &IdentityManager{}
	if _, ok := ctx.Value(common.CtxKeyAPIKey{}).(string); ok {
		manager.isAPITokenCall = true
		return manager
	}

	if _, ok := ctx.Value(common.CtxKeyToken{}).(string); ok {
		if claims, ok := ctx.Value(common.CtxKeyClaims{}).(*common.AllClaims); ok {
			manager.subject = claims.Subject
			manager.roles = claims.Global.Roles

			manager.isAttendee = true

			for _, role := range claims.Global.Roles {
				if role == "admin" {
					manager.isAttendee = false
					manager.isAdmin = true
					break
				}
			}
		}
	}

	return manager
}
