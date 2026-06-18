package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"ms_home/pkg/httpclient"
)

// ExchangeAdapter exchanges an opaque session cookie for decoded token claims at
// the Auth service. Mirrors digital_bff AuthProvider.exchangeToken
// (GET /v2/auth/exchange-token?{cookieName}={sessionId}, header x-brand-id=<client>).
type ExchangeAdapter struct {
	http       *httpclient.Client
	baseURL    string
	cookieName string
}

// NewExchange builds the adapter. baseURL is the Auth service base.
func NewExchange(client *httpclient.Client, baseURL, cookieName string) *ExchangeAdapter {
	return &ExchangeAdapter{http: client, baseURL: strings.TrimRight(baseURL, "/"), cookieName: cookieName}
}

// ExchangeToken validates the session and returns the decoded access-token claims
// (decodeAccessToken: prn, isAnonymous, isSignUp, …). client is the Auth0 client
// (brand, upper-cased by the caller).
func (a *ExchangeAdapter) ExchangeToken(ctx context.Context, sessionID, client string) (map[string]any, error) {
	q := url.Values{a.cookieName: {sessionID}}
	endpoint := a.baseURL + "/v2/auth/exchange-token?" + q.Encode()

	resp, err := a.http.Do(ctx, http.MethodGet, endpoint, nil, map[string]string{"x-brand-id": client})
	if err != nil {
		return nil, fmt.Errorf("auth: exchange-token: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("auth: exchange-token: unexpected status %d", resp.Status)
	}

	var parsed struct {
		DecodeAccessToken map[string]any `json:"decodeAccessToken"`
	}
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("auth: exchange-token decode: %w", err)
	}
	return parsed.DecodeAccessToken, nil
}
