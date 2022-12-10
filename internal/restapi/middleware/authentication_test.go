package middleware

import (
	"crypto/rsa"
	"net/http"
	"testing"

	"github.com/go-http-utils/headers"
	"github.com/golang-jwt/jwt/v4"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
	"github.com/stretchr/testify/require"
)

const valid_JWT_is_not_staff = `eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZ2xvYmFsIjp7Im5hbWUiOiJKb2huIERvZSIsInJvbGVzIjpbXX0sImlhdCI6MTUxNjIzOTAyMn0.ove6_7BWQRe9HQyphwDdbiaAchgn9ynC4-2EYEXFeVTDADC4P3XYv5uLisYg4Mx8BZOnkWX-5L82pFO1mUZM147gLKMsYlc-iMKXy4sKZPzhQ_XKnBR-EBIf5x_ZD1wpva9ti7Yrvd0vDi8YSFdqqf7R4RA11hv9kg-_gg1uea6sK-Q_eEqoet7ocqGVLu-ghhkZdVLxu9tWJFPNueILWv8vW1Y_u9fDtfOhw7Ugf5ysI9RXiO-tXEHKN2HnFPCkwccnMFt4PJRzU1VoOldz0xzzZRb-j2tlbjLqcQkjMwLEoPQpC4Wbl8DgkaVdTi2aNyH7EbWMynlSOZIYK0AFvQ`
const valid_JWT_is_not_staff_sub101 = `eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMDEiLCJnbG9iYWwiOnsibmFtZSI6Ik5vcm1hbG8gMTAxIiwicm9sZXMiOltdfSwiaWF0IjoxNTE2MjM5MDIyfQ.btbaXOuIP23GpDQH3yRM82h4VoKG6HFLsIs4oh9fNKgb_P6exEOc2jeRSQXkpXjOst-xDGzAy7QtvK_ZN7ckPJAWWo5EhH4ujJxtzIGe-q013ST6q_54S887Cvdyf3EpIE9vV4ZNK0agFApghW4B62vrJuO00jwLS-V6wRSqkN6GAYQPbX3aAVBS7dPZgKxxHSDyOMRG-hHrc6BExMGQr89fMAHR7QkwWx0AeFDYJZ7AkI0XlYNVG1kVlKLbHYCbx6I4XTcHqMsHqlYJ9qVtss3GjVIfF3OPld3Ni5kR--51wFIZs2-47vLxUAGr5EHsblreZIjLYsDJO01ZfwURdw`

const valid_JWT_is_staff = `eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZ2xvYmFsIjp7Im5hbWUiOiJKb2huIERvZSIsInJvbGVzIjpbInN0YWZmIl19LCJpYXQiOjE1MTYyMzkwMjJ9.PNO4vV6V6iRg4-LcvJsRHyTSx7-6lDmqh6GrUWM4_OrhmmUWh2W4KF6sOfUco7sJ_I0PFOrnPGqREYAPuG1oAkHfitq5GdkYHCnJuHXXWo5d982GPs7zI-l9SxAgcUDdytesmSbq9Ktoad94OUL5bR8Uln0DPTlZvXDTAuCmAMW_89a4C-i71bsCYaFgL0RsJQ4yR4f3ez2M4hG4mNBjwaU4Ke77qdQIjx_9pP5ph37X8Z7twsC1yBH-Hev-293Naj3FZS8y63Zb6VGG3w8WW69eN_apoGRo26ZyaiDChAzOI-c1xkbMC5KYbnFQl5Ubdgk8sQgmp20RHHTV1R8Bcg`
const valid_JWT_is_staff_sub202 = `eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIyMDIiLCJnbG9iYWwiOnsibmFtZSI6IkpvaG4gU3RhZmYiLCJyb2xlcyI6WyJzdGFmZiJdfSwiaWF0IjoxNTE2MjM5MDIyfQ.aZPHUaia1SBvAu06DKDURIpcqmexk7MkHCzYvrdj4l9H3QbXeCBfA3WZvcw1bN5C-aEN3GmJeiaCpK4m1Loi7oJxJgxEL1iUp4zW_tglPd0QNquLpZNNxDLa-99PpWDLw1EYslqYWd74lB2xnlZvrxmTpciDJeBIWRZA1bAISZQLRGDCv4VD_qZrkEHl66dOTp7kjYeQ9hme9ckeFu06MoOj0p8EdM9GPXlGQFlXYiKbwan4guvJNtIOnbERlUfhWKdL3GffAY7_zO1Xu0lipm9bGHbI0OH3-HQDnKBGyhPvRg829LMfZZW7qrwu-UW4hKgS9L4e8ltcGoHL3wlD3w`

const valid_JWT_is_admin = `eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZ2xvYmFsIjp7Im5hbWUiOiJKb2huIERvZSIsInJvbGVzIjpbImFkbWluIl19LCJpYXQiOjE1MTYyMzkwMjJ9.sriAGCekreVU3nlQHc8Di7BqqI4Tut7tVNMWYa3kEpRi39Em5lOQ0b7w69idZEKT-MJfBGLVicnkw7Q4l8pUpJuHZMnja5YBIp7FDTg-KKbX__oOSSOnLhjaIGNFR_Xk_DanGrolQMKSYIfQs8MSgRO1bq-ZccCp1iJ4sdOOS4PenXj9h6xSe_lidGp8Wk47qwzRAFHYURaHFl_TCPMNDrYbM5MMIv8Lkye_duLxLo3zc9bnwWinhyD00p7ASwKgMc6vtWeTu_h000OOuviKoc2XKzOjUurqtm9Cird5rDAgAYtT_nTI_N4IzWFiRRPqX1IODe2zlqvKucv_FjzE8g`

const pemPublicKeyRS512 = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu1SU1LfVLPHCozMxH2Mo
4lgOEePzNm0tRgeLezV6ffAt0gunVTLw7onLRnrq0/IzW7yWR7QkrmBL7jTKEn5u
+qKhbwKfBstIs+bMY2Zkp18gnTxKLxoS2tFczGkPLPgizskuemMghRniWaoLcyeh
kd3qqGElvW/VDL5AaWTg0nLVkjRo9z+40RQzuVaE8AkAFmxZzow3x+VJYKdjykkJ
0iT9wCS0DRTXu269V264Vf/3jvredZiKRkgwlL9xNAwxXFg0x/XFw005UWVRIkdg
cKWTjpBP2dPwVZ4WWC+9aGVd+Gyn1o0CLelf4rEjGoXbAAEgAqeGUxrcIlbjXfbc
mwIDAQAB
-----END PUBLIC KEY-----
`

const valid_JWT_is_admin_256 = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZ2xvYmFsIjp7Im5hbWUiOiJKb2huIERvZSIsInJvbGVzIjpbImFkbWluIl19LCJpYXQiOjE1MTYyMzkwMjJ9.L8CNx5rE9TQSdd1II7UythBlo5o2lhIYvXG6eDGrkMNYBWEcYBShgTCJvOMrxXIOF16H6HVlBYLNBBGesCgsao3ffXsJZkDJML_9mC31mdtqVS5-L0Ka7xuZTc7OXyCWqVmNLG0IthY3Pa8QfOol5OOrynJVNF6tbAHVZ_Kxn5u2edMT1Cn2ngPTV5OXqHArhNvb8PbcxyV5U4VOwSAHy6pxBjxaV_IQrLkPi2f1aV4Mr9tYqXf8yEFNi70WH_pI0mXMWIbwWmBP9ESJAvrQIiSdfIURIk2u5-HcNiHMBCy4CrnCz3_xJjI6GVyJYNZNjtppGWx7QHmDNIhZuzCIAg`
const valid_JWT_is_admin_256_no_claims = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiaWF0IjoxNTE2MjM5MDIyfQ.ajUvZZVkUQIsqLn_nEj-py1n6HWek7KCuyFlYMEe5D37XHI7Ydujcs4MAGKNVI7vCY_oyQHtL32lxKDinTiT-wgWLxwJtSXCxfu6aOTFnpC4JOGTFGhjzWjOSB4djW2fKthkS0xR_0NEOWMF3RjqMsneiZDKRobZhkH3VLnNgUhAM1Msy6laPvxwUf-qeqH0LZOhRJ21_TstII7xDKpilkwiBCoHFoQTlNECHqCiC8B69fCVlUo6Ri--a5WhV6p_t4SKEtP2bVRXjyIA8e6tG0qsL9ki6UaT4AejK7UvA4dIRwu2dRVFPJGDegcbB0OSVSPbTSJI_-ygi-Z6Cj03fg`
const invalid_JWT_is_admin_no_subject_256 = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJnbG9iYWwiOnsibmFtZSI6IkpvaG4gRG9lIiwicm9sZXMiOlsiYWRtaW4iXX0sImlhdCI6MTUxNjIzOTAyMn0.qNvWt_hp357DUZMCZLXOzWwpC0eeYReipcXQhkIzKkBO6m0xgO3MmOU4GEZFnA69d9Hi-0b0FhnwrenhIKNLjixwQ4zaO5BicptoPw-giQLQkutAcBglmi6v55dGGqS0zikE8w2tgK5HfLPmvNm2ZEj_FPipSyeK9O1JJw2F_cHEBmrRONp69Qdybfk1gsrTwQx7hZSHOv8q0F58dr4tctbySQerdlvInbYPMIgOqQ8PCj5t5bHA4-dwHOSxz8gqG3oTBZ50o8RbLqh7tsatqRVo64wTI86g4evKxRnsBlpcy4BLID6lQ-_2d7w5bFBNw9ZW-4dA-CNc347hKw59cQ`

const valid_JWT_is_non_staff_subject_256 = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZ2xvYmFsIjp7Im5hbWUiOiJKb2huIERvZSIsInJvbGVzIjpbXX0sImlhdCI6MTUxNjIzOTAyMn0.phzC1-ExNKJyYoh2MD-3N0Kd6gUjPUXuOplf6bvkb2qyhDDcVJFQtx9lcLLz03KhYBZf2LFp8tLu0Ezu9KacJXNSKmN2Vl_EMbbsxmDCe0JnDZ-UZQXlE6Z43dQPmKVXSKzYMSNPPEN0UsSxS_DYzHkYG2kUwjeI51y_zQ5Yis6M5XO4erh6ji6Lf_XYZseR-MG6PxkO4AdtOSijSj_12z_17QiuYImqljrAp2pmvALhyQzgCIRRyCeBY_T2NQVr7SkTR8ljAB9nv0b1DlZHE-N3qBrPHXjY83VZ4avabeugBOWxLSlfZwz7YqQdYNQVlXfW57aT1OCs0HQEmdtkqg`
const invalid_JWT_is_non_staff_no_subject_256 = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJnbG9iYWwiOnsibmFtZSI6IkpvaG4gRG9lIiwicm9sZXMiOltdfSwiaWF0IjoxNTE2MjM5MDIyfQ.hNVhIDkL2CWLwmRDuQdJIpXdNTMpa1oIjZ6jBT169UFpaCSreNNzeA5zaqMd1Kj0Tx6i2VjwPRHOpzqPpaY3kIWyjc1zfvh-tQ9KiCnNXFYQ3voXu0pj4u1l3HqXgjzX7oghGT-UdkujaBYito9Bd85JHVERedkUWCcmFXM_T-kzn-_br2-ODP2NWhTsv8-VZm5EZYqCa2hH31QPsQF5MBl67bA0HgKGFgoRvaEH72fVoJHnQ7ZwtcvEMFOA6ag-Q7PELN3LNNH-I-RQtScwJu1uJP2sPYnrRX0N-sNr_8vLag_RcWbFZcnZiRmCmg91vQzr44zKXjmDFAsSXwbSZA`

const pemPublicKeyRS256 = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu1SU1LfVLPHCozMxH2Mo
4lgOEePzNm0tRgeLezV6ffAt0gunVTLw7onLRnrq0/IzW7yWR7QkrmBL7jTKEn5u
+qKhbwKfBstIs+bMY2Zkp18gnTxKLxoS2tFczGkPLPgizskuemMghRniWaoLcyeh
kd3qqGElvW/VDL5AaWTg0nLVkjRo9z+40RQzuVaE8AkAFmxZzow3x+VJYKdjykkJ
0iT9wCS0DRTXu269V264Vf/3jvredZiKRkgwlL9xNAwxXFg0x/XFw005UWVRIkdg
cKWTjpBP2dPwVZ4WWC+9aGVd+Gyn1o0CLelf4rEjGoXbAAEgAqeGUxrcIlbjXfbc
mwIDAQAB
-----END PUBLIC KEY-----`

func TestParseAuthCookie(t *testing.T) {
	tests := []struct {
		name        string
		inputName   string
		inputCookie *http.Cookie
		expected    string
	}{
		{
			name:      "Should get value from cookie",
			inputName: "test-cookie",
			inputCookie: &http.Cookie{
				Name:  "test-cookie",
				Value: "cookie-value",
			},
			expected: "Bearer cookie-value",
		},
		{
			name:      "Should return empty string when cookie name doesn't match",
			inputName: "incorrect-name",
			inputCookie: &http.Cookie{
				Name:  "test-cookie",
				Value: "cookie-value",
			},
			expected: "",
		},
		{
			name:      "Should return empty string when cookie config name is empty",
			inputName: "",
			inputCookie: &http.Cookie{
				Name:  "test-cookie",
				Value: "cookie-value",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)
			r.AddCookie(tt.inputCookie)

			value := parseAuthCookie(r, tt.inputName)

			require.Equal(t, tt.expected, value)
		})
	}

}

func TestParseBearerToken(t *testing.T) {

	strPtr := func(s string) *string {
		return &s
	}

	tests := []struct {
		name                 string
		inputTokenCookieName string
		inputAuthHeaderValue *string
		inputCookie          *http.Cookie
		expected             string
	}{
		{
			name:                 "Header present, should get value from auth header",
			inputAuthHeaderValue: strPtr("Bearer header-value"),
			inputTokenCookieName: "doesn't matter",
			inputCookie:          nil,
			expected:             "Bearer header-value",
		},
		{
			name:                 "Header not present, should get cookie value",
			inputAuthHeaderValue: nil,
			inputTokenCookieName: "test-cookie",
			inputCookie: &http.Cookie{
				Name:  "test-cookie",
				Value: "cookie-value",
			},
			expected: "Bearer cookie-value",
		},
		{
			name:                 "Make sure TokenCookieName from config is honored",
			inputAuthHeaderValue: nil,
			inputTokenCookieName: "another-test-cookie",
			inputCookie: &http.Cookie{
				Name:  "another-test-cookie",
				Value: "cookie-value",
			},
			expected: "Bearer cookie-value",
		},
		{
			name:                 "Existing but empty header leads to the cookie being used",
			inputAuthHeaderValue: strPtr(""),
			inputTokenCookieName: "another-test-cookie",
			inputCookie: &http.Cookie{
				Name:  "another-test-cookie",
				Value: "cookie-value",
			},
			expected: "Bearer cookie-value",
		},
		// two more test cases are already covered via TestParseAuthCookie
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			if tt.inputAuthHeaderValue != nil {
				r.Header.Add(headers.Authorization, *tt.inputAuthHeaderValue)
			}
			if tt.inputCookie != nil {
				r.AddCookie(tt.inputCookie)
			}

			// We only need to fill a single field, the others are not used.
			securityConf := &config.SecurityConfig{
				Oidc: config.OpenIdConnectConfig{
					TokenCookieName: tt.inputTokenCookieName,
				},
			}

			value := parseBearerToken(r, securityConf)

			require.Equal(t, tt.expected, value)
		})
	}
}

func TestKeyFuncForKey(t *testing.T) {
	parseKey := func(t *testing.T, inputPem string) *rsa.PublicKey {
		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(inputPem))
		require.NoError(t, err)

		return key
	}

	tests := []struct {
		name          string
		inputKey      *rsa.PublicKey
		expectedError error
	}{
		{
			name:          "Should successfully parse RSA key and return no error",
			inputKey:      parseKey(t, pemPublicKeyRS512),
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rsaKey, err := keyFuncForKey(tt.inputKey)(nil)
			if tt.expectedError != nil {
				require.Error(t, err)
			}

			require.IsType(t, &rsa.PublicKey{}, rsaKey)
			require.Equal(t, tt.inputKey, rsaKey)

		})
	}
}

func TestCheckRequestAuthorization_ParsePEMs(t *testing.T) {
	require.Panics(t, func() {
		CheckRequestAuthorization(nil)
	})

	require.Panics(t, func() {
		CheckRequestAuthorization(&config.SecurityConfig{
			Oidc: config.OpenIdConnectConfig{
				TokenPublicKeysPEM: []string{"ABC123"},
			},
		})
	})

}

type statusCodeResponseWriter struct {
	statusCode int
}

func (s *statusCodeResponseWriter) Header() http.Header {
	return nil
}

func (s *statusCodeResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (s *statusCodeResponseWriter) WriteHeader(statusCode int) {
	s.statusCode = statusCode
}

func TestCheckRequestAuthorization(t *testing.T) {
	type args struct {
		xAPITokenHeader     string
		authorizationHeader string
	}

	type expected struct {
		xAPIToken  string
		jwt        string
		claims     *common.AllClaims
		shouldFail bool
	}

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "Should successfully retrieve API token from header",
			args: args{
				xAPITokenHeader:     "test-shared-secret",
				authorizationHeader: "",
			},
			expected: expected{
				xAPIToken:  "test-shared-secret",
				jwt:        "",
				claims:     nil,
				shouldFail: false,
			},
		},
		{
			name: "Should not proceed when API token doesn't match the configured value",
		},
		{
			name: "Should not proceed when both authorization header and cookie are missing",
		},
		{
			name: "Should fail validation when authorization header doesn't contain `Bearer ` prefix",
		},
		{
			name: "Should fail validation when only `Bearer ` exists without token",
		},
		{
			name: "Should fail validation when token contains blanks",
		},
		{
			name: "Should successfully parse JWT token against configured PEM RS256",
		},
		{
			name: "Should succeeed when no token claims are present",
		},
		{
			name: "Should fail when no subject was provided in the token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &statusCodeResponseWriter{}

			config := &config.SecurityConfig{
				Fixed: config.FixedTokenConfig{
					Api: "test-shared-secret",
				},
				Oidc: config.OpenIdConnectConfig{
					TokenPublicKeysPEM: []string{pemPublicKeyRS256, pemPublicKeyRS512},
				},
			}

			r, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			if tt.args.xAPITokenHeader != "" {
				r.Header.Add(apiKeyHeader, tt.args.xAPITokenHeader)
			}
			if tt.args.authorizationHeader != "" {
				r.Header.Add(headers.Authorization, tt.args.authorizationHeader)
			}

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expected.xAPIToken != "" {
					value, ok := r.Context().Value(common.CtxKeyAPIToken{}).(string)
					if !ok {
						require.FailNow(t, "expected type string")
					}
					require.Equal(t, tt.expected.xAPIToken, value)
				}
			})

			fn := CheckRequestAuthorization(config)

			fn(next).ServeHTTP(w, r)
		})
	}

}
