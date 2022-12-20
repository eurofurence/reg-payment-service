package interaction

import (
	"context"
	"testing"

	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestNewIdentityManager(t *testing.T) {

	type args struct {
		inputJWT    string
		inputAPIKey string
		inputClaims *common.AllClaims
	}

	type expected struct {
		subject          string
		roles            []string
		isAdmin          bool
		isAPITokenCall   bool
		isRegisteredUser bool
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "Should create manager with valid API token",
			args: args{
				inputJWT:    "",
				inputAPIKey: "api-token",
				inputClaims: nil,
			},
			expected: expected{
				isAPITokenCall: true,
			},
		},
		{
			name: "Should create manager with admin role",
			args: args{
				inputJWT:    "valid",
				inputAPIKey: "",
				inputClaims: &common.AllClaims{
					RegisteredClaims: jwt.RegisteredClaims{
						Subject: "123456",
					},
					CustomClaims: common.CustomClaims{
						Global: common.GlobalClaims{
							Roles: []string{"admin", "test"},
							Name:  "Peter",
							EMail: "peter@peter.eu",
						},
					},
				},
			},
			expected: expected{
				isAdmin: true,
				subject: "123456",
				roles:   []string{"admin", "test"},
			},
		},
		{
			name: "Should create manager with registered user role",
			args: args{
				inputJWT:    "valid",
				inputAPIKey: "",
				inputClaims: &common.AllClaims{
					RegisteredClaims: jwt.RegisteredClaims{
						Subject: "123456",
					},
					CustomClaims: common.CustomClaims{
						Global: common.GlobalClaims{
							Roles: []string{"staff", "test"},
							Name:  "Peter",
							EMail: "peter@peter.eu",
						},
					},
				},
			},
			expected: expected{
				isRegisteredUser: true,
				subject:          "123456",
				roles:            []string{"staff", "test"},
			},
		},
		{
			name: "Should return empty manager if no tokens provided",
			args: args{
				inputJWT:    "",
				inputAPIKey: "",
				inputClaims: nil,
			},
			expected: expected{},
		},
		{
			name: "Should return empty manager if no tokens provided",
			args: args{
				inputJWT:    "",
				inputAPIKey: "",
				inputClaims: nil,
			},
			expected: expected{},
		},
		{
			name: "Should be invalid if registered user has no subject assigned",
			args: args{
				inputJWT:    "valid",
				inputAPIKey: "",
				inputClaims: &common.AllClaims{
					CustomClaims: common.CustomClaims{
						Global: common.GlobalClaims{
							Roles: []string{""},
						},
					},
				},
			},
			expected: expected{
				roles: []string{""},
			},
		},
		{
			name: "API key should dominate over JWT token",
			args: args{
				inputJWT:    "valid",
				inputAPIKey: "also valid",
				inputClaims: &common.AllClaims{
					CustomClaims: common.CustomClaims{
						Global: common.GlobalClaims{
							Roles: []string{""},
						},
					},
				},
			},
			expected: expected{
				isAPITokenCall: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.args.inputAPIKey != "" {
				ctx = context.WithValue(ctx, common.CtxKeyAPIKey{}, tt.args.inputAPIKey)
			}

			if tt.args.inputJWT != "" {
				ctx = context.WithValue(ctx, common.CtxKeyToken{}, tt.args.inputJWT)
			}

			if tt.args.inputClaims != nil {
				ctx = context.WithValue(ctx, common.CtxKeyClaims{}, tt.args.inputClaims)
			}

			mgr := NewIdentityManager(ctx)

			require.Equal(t, tt.expected.isAdmin, mgr.IsAdmin())
			require.Equal(t, tt.expected.isAPITokenCall, mgr.IsAPITokenCall())
			require.Equal(t, tt.expected.isRegisteredUser, mgr.IsRegisteredUser())
			require.Equal(t, tt.expected.roles, mgr.roles)
			require.Equal(t, tt.expected.subject, mgr.subject)

		})
	}
}
