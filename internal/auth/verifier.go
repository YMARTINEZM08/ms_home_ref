// Package auth validates inbound JWTs (service-side, RS256) against a JWKS,
// returning the decoded claims. It is the auth boundary for ms_home.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwksRefreshThrottle bounds how often a missing key id triggers a JWKS refetch.
const jwksRefreshThrottle = 30 * time.Second

// Verifier validates RS256 JWTs using cached JWKS public keys.
type Verifier struct {
	jwksURL  string
	issuer   string
	audience string
	client   *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	lastFetch time.Time
}

// NewVerifier builds a verifier. issuer/audience are validated only when non-empty.
func NewVerifier(jwksURL, issuer, audience string, timeout time.Duration) *Verifier {
	return &Verifier{
		jwksURL:  jwksURL,
		issuer:   issuer,
		audience: audience,
		client:   &http.Client{Timeout: timeout},
		keys:     make(map[string]*rsa.PublicKey),
	}
}

// Verify parses and validates the token, returning its claims. Only RS256 is
// accepted (alg=none and HMAC are rejected), guarding against algorithm confusion.
func (v *Verifier) Verify(ctx context.Context, tokenString string) (map[string]any, error) {
	opts := []jwt.ParserOption{jwt.WithValidMethods([]string{"RS256"})}
	if v.issuer != "" {
		opts = append(opts, jwt.WithIssuer(v.issuer))
	}
	if v.audience != "" {
		opts = append(opts, jwt.WithAudience(v.audience))
	}

	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(tokenString, claims, v.keyfunc(ctx), opts...); err != nil {
		return nil, fmt.Errorf("auth: verify: %w", err)
	}
	return map[string]any(claims), nil
}

func (v *Verifier) keyfunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if k := v.lookup(kid); k != nil {
			return k, nil
		}
		if err := v.refresh(ctx); err != nil {
			return nil, fmt.Errorf("auth: jwks refresh: %w", err)
		}
		if k := v.lookup(kid); k != nil {
			return k, nil
		}
		return nil, fmt.Errorf("auth: unknown key id %q", kid)
	}
}

func (v *Verifier) lookup(kid string) *rsa.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.keys[kid]
}

// refresh refetches the JWKS, throttled to avoid hammering on unknown key ids.
func (v *Verifier) refresh(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.keys) > 0 && time.Since(v.lastFetch) < jwksRefreshThrottle {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jwks status %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pk, err := rsaPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pk
	}
	v.keys = keys
	v.lastFetch = time.Now()
	return nil
}

// rsaPublicKey builds an RSA public key from base64url JWK modulus/exponent.
func rsaPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}
