package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// testIDP spins up an in-memory JWKS server and signs tokens with a test RSA key.
type testIDP struct {
	key    *rsa.PrivateKey
	kid    string
	server *httptest.Server
}

func newTestIDP(t *testing.T) *testIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	idp := &testIDP{key: key, kid: "test-kid"}
	idp.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jwk := map[string]any{
			"keys": []any{map[string]any{
				"kty": "RSA", "kid": idp.kid, "alg": "RS256", "use": "sig",
				"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
			}},
		}
		_ = json.NewEncoder(w).Encode(jwk)
	}))
	t.Cleanup(idp.server.Close)
	return idp
}

func (idp *testIDP) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = idp.kid
	s, err := tok.SignedString(idp.key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestVerifyValidToken(t *testing.T) {
	idp := newTestIDP(t)
	v := NewVerifier(idp.server.URL, "issuer-x", "aud-x", 5*time.Second)

	token := idp.sign(t, jwt.MapClaims{
		"iss": "issuer-x", "aud": "aud-x", "exp": time.Now().Add(time.Hour).Unix(),
		"profileId": "p123", "dateOfBirth": float64(1),
	})

	claims, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims["profileId"] != "p123" {
		t.Errorf("profileId claim = %v", claims["profileId"])
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	idp := newTestIDP(t)
	v := NewVerifier(idp.server.URL, "", "", 5*time.Second)
	token := idp.sign(t, jwt.MapClaims{"exp": time.Now().Add(-time.Hour).Unix()})
	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Error("expected expired token to be rejected")
	}
}

func TestVerifyRejectsWrongIssuer(t *testing.T) {
	idp := newTestIDP(t)
	v := NewVerifier(idp.server.URL, "issuer-x", "", 5*time.Second)
	token := idp.sign(t, jwt.MapClaims{"iss": "evil", "exp": time.Now().Add(time.Hour).Unix()})
	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Error("expected wrong issuer to be rejected")
	}
}

func TestVerifyRejectsNoneAlg(t *testing.T) {
	idp := newTestIDP(t)
	v := NewVerifier(idp.server.URL, "", "", 5*time.Second)
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()})
	s, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := v.Verify(context.Background(), s); err == nil {
		t.Error("alg=none must be rejected")
	}
}

func TestVerifyRejectsUnknownKid(t *testing.T) {
	idp := newTestIDP(t)
	v := NewVerifier(idp.server.URL, "", "", 5*time.Second)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()})
	tok.Header["kid"] = "other-kid"
	s, _ := tok.SignedString(idp.key)
	if _, err := v.Verify(context.Background(), s); err == nil {
		t.Error("unknown kid must be rejected")
	}
}
