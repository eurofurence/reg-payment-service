package middleware

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-http-utils/headers"
	"github.com/golang-jwt/jwt/v4"

	"github.com/eurofurence/reg-payment-service/internal/config"
	"github.com/eurofurence/reg-payment-service/internal/logging"
	"github.com/eurofurence/reg-payment-service/internal/restapi/common"
)

const apiKeyHeader = "X-Api-Key"
const bearerPrefix = "Bearer"

func parseAuthCookie(r *http.Request, cookieName string) string {
	if cookieName == "" {
		return ""
	}

	authCookie, err := r.Cookie(cookieName)
	if err != nil && errors.Is(err, http.ErrNoCookie) {
		return ""
	}

	return fmt.Sprintf("%s %s", bearerPrefix, authCookie.Value)
}

func parseBearerToken(r *http.Request, conf *config.SecurityConfig) string {
	token := r.Header.Get(headers.Authorization)
	if token != "" {
		return token
	}

	return parseAuthCookie(r, conf.Oidc.TokenCookieName)
}

func getApiKeyFromHeader(r *http.Request) string {
	return r.Header.Get(apiKeyHeader)
}

// --- middleware validating the values and adding to context values ---

func keyFuncForKey(rsaPublicKey *rsa.PublicKey) func(token *jwt.Token) (interface{}, error) {
	return func(token *jwt.Token) (interface{}, error) {
		return rsaPublicKey, nil
	}
}

// TODO example - no idea if this matches the idp claims structure - compare to room service!

func CheckRequestAuthorization(conf *config.SecurityConfig) func(http.Handler) http.Handler {
	parsedPEMs := make([]*rsa.PublicKey, len(conf.Oidc.TokenPublicKeysPEM))

	for i, publicKey := range conf.Oidc.TokenPublicKeysPEM {
		rsaKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
		if err != nil {
			panic("Couldn't parse configured pem " + publicKey)
		}

		parsedPEMs[i] = rsaKey
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			reqID := logging.GetRequestID(ctx)
			logger := logging.LoggerFromContext(ctx)

			// check for api key first
			if token := getApiKeyFromHeader(r); token != "" {
				if token == conf.Fixed.Api {
					ctx = context.WithValue(ctx, common.CtxKeyAPIKey{}, token)
					r = r.WithContext(ctx)
					next.ServeHTTP(w, r)
				} else {
					common.SendUnauthorizedResponse(w, reqID, logger, "Token doesn't match the configured value")
				}

				return
			}

			// get bearer token
			token := parseBearerToken(r, conf)

			if token == "" {
				common.SendUnauthorizedResponse(w, reqID, logger, "Token is missing")
				return
			}

			if !strings.HasPrefix(token, bearerPrefix) {
				common.SendUnauthorizedResponse(w, reqID, logger, "value of Authorization header did not start with 'Bearer '")
				return
			}

			split := strings.Split(token, " ")
			if len(split) != 2 {
				common.SendUnauthorizedResponse(w, reqID, logger, "invalid structure for authorization header")
				return
			}

			tokenString := split[1]

			for _, key := range parsedPEMs {
				claims := common.AllClaims{}
				token, err := jwt.ParseWithClaims(tokenString, &claims, keyFuncForKey(key), jwt.WithValidMethods([]string{"RS256", "RS512"}))
				if err != nil {
					logger.Debug("Couldn't parse token, [reason]: %s", err.Error())
					continue
				}

				if token.Valid {

					// check if claims are set already. If the logic doesn't set the claims variable, defer to parse them from the
					// generic interface

					// set the context values for the claims and token

					if claims.Subject == "" {
						common.SendUnauthorizedResponse(w, reqID, logger, "No subject was supplied in the token")
						return
					}

					ctx = context.WithValue(ctx, common.CtxKeyToken{}, tokenString)
					ctx = context.WithValue(ctx, common.CtxKeyClaims{}, &claims)
					r = r.WithContext(ctx)
					next.ServeHTTP(w, r)
					return
				}
			}

			common.SendUnauthorizedResponse(w, reqID, logger, "")
		})
	}
}
