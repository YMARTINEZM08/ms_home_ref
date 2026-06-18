package http

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// Authenticator resolves the caller's identity from the request. Implementations:
// jwt (local validation), opaque (digital_bff-style cookie exchange). A nil
// Authenticator means dev mode (identity from the x-profile-id header).
type Authenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (profileID string, loggedIn bool, claims map[string]any)
}

// TokenVerifier validates a bearer JWT and returns its claims (auth.Verifier).
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (map[string]any, error)
}

// TokenExchanger exchanges an opaque session for decoded claims (auth.ExchangeAdapter).
type TokenExchanger interface {
	ExchangeToken(ctx context.Context, sessionID, client string) (map[string]any, error)
}

// jwtAuthenticator validates a Bearer JWT locally.
type jwtAuthenticator struct {
	verifier     TokenVerifier
	profileClaim string
	log          *slog.Logger
}

// NewJWTAuthenticator builds the local-JWT authenticator.
func NewJWTAuthenticator(v TokenVerifier, profileClaim string, log *slog.Logger) Authenticator {
	return jwtAuthenticator{verifier: v, profileClaim: profileClaim, log: log}
}

func (a jwtAuthenticator) Authenticate(ctx context.Context, r *http.Request) (string, bool, map[string]any) {
	token := bearerToken(r)
	if token == "" {
		return "", false, nil
	}
	claims, err := a.verifier.Verify(ctx, token)
	if err != nil {
		a.log.DebugContext(ctx, "jwt verification failed", slog.String("error", err.Error()))
		return "", false, nil
	}
	p := claimString(claims, a.profileClaim)
	return p, p != "", claims
}

// opaqueAuthenticator exchanges the session cookie for claims at the Auth service.
type opaqueAuthenticator struct {
	exchange     TokenExchanger
	cookieName   string
	defaultBrand string
	log          *slog.Logger
}

// NewOpaqueAuthenticator builds the cookie-exchange authenticator (digital_bff parity).
func NewOpaqueAuthenticator(ex TokenExchanger, cookieName, defaultBrand string, log *slog.Logger) Authenticator {
	return opaqueAuthenticator{exchange: ex, cookieName: cookieName, defaultBrand: defaultBrand, log: log}
}

func (a opaqueAuthenticator) Authenticate(ctx context.Context, r *http.Request) (string, bool, map[string]any) {
	c, err := r.Cookie(a.cookieName)
	if err != nil || c.Value == "" {
		return "", false, nil
	}
	brand := r.Header.Get("x-brand-id")
	if brand == "" {
		brand = a.defaultBrand
	}
	claims, err := a.exchange.ExchangeToken(ctx, c.Value, strings.ToUpper(brand))
	if err != nil {
		a.log.DebugContext(ctx, "token exchange failed", slog.String("error", err.Error()))
		return "", false, nil
	}
	// profileId = prn; isLoggedIn = !isAnonymous (mirrors OpaqueTokenMiddleware).
	prn, _ := claims["prn"].(string)
	anon, _ := claims["isAnonymous"].(bool)
	return prn, prn != "" && !anon, claims
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):]
	}
	return ""
}

// claimString reads a claim as a string (numbers are formatted without decimals).
func claimString(claims map[string]any, key string) string {
	switch v := claims[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}
