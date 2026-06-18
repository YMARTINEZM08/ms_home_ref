package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"ms_home/internal/application"
	"ms_home/internal/domain"
	"ms_home/internal/populate"
)

type fakeContent struct {
	docs map[domain.ContentType]domain.Document
}

func (f fakeContent) GetContent(_ context.Context, ct domain.ContentType, _, _ string) (domain.Document, error) {
	src := f.docs[ct]
	out := make(domain.Document, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out, nil
}

type fakeCart struct{ chd map[string]any }

func (f fakeCart) GetCartHeaderDetails(context.Context) (map[string]any, error) { return f.chd, nil }

type fakeVerifier struct{ claims map[string]any }

func (f fakeVerifier) Verify(context.Context, string) (map[string]any, error) { return f.claims, nil }

func TestHandlerJWTIdentityAndMe(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	content := fakeContent{docs: map[domain.ContentType]domain.Document{
		domain.ContentTypePage:   {"_content_type_uid": "page"},
		domain.ContentTypeGlobal: {"feature_flags": map[string]any{"personalization": true}},
	}}
	cart := fakeCart{chd: map[string]any{
		"firstName": "Ana", "login": "ana@x.com",
		"favoriteStore": map[string]any{"id": "42", "storeName": "Centro"},
	}}
	pop := populate.NewService(populate.NewRegistry(), discard)
	home := application.NewHomeService(content, pop, cart, true, discard)

	verifier := fakeVerifier{claims: map[string]any{"profileId": "p9", "lastPasswordReset": "2026"}}
	authn := NewJWTAuthenticator(verifier, "profileId", discard)
	mux := NewRouter(NewHandler(home, authn, "LP", discard), "test")

	req := httptest.NewRequest(http.MethodGet, "/content/page/es-mx/", nil)
	req.Header.Set("Authorization", "Bearer any-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	me, ok := resp["me"].(map[string]any)
	if !ok {
		t.Fatalf("me missing: %v", resp)
	}
	if me["profileId"] != "p9" {
		t.Errorf("profileId from JWT = %v", me["profileId"])
	}
	if me["lastPasswordReset"] != "2026" {
		t.Errorf("token claim lastPasswordReset = %v", me["lastPasswordReset"])
	}
	if me["email"] != "ana@x.com" || me["firstName"] != "Ana" {
		t.Errorf("cart-header fields wrong: %v", me)
	}
}

func TestHandlerNoVerifierUsesHeader(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	content := fakeContent{docs: map[domain.ContentType]domain.Document{
		domain.ContentTypePage:   {"_content_type_uid": "page"},
		domain.ContentTypeGlobal: {"feature_flags": map[string]any{"personalization": true}},
	}}
	pop := populate.NewService(populate.NewRegistry(), discard)
	home := application.NewHomeService(content, pop, nil, true, discard)
	mux := NewRouter(NewHandler(home, nil, "LP", discard), "test") // dev mode

	req := httptest.NewRequest(http.MethodGet, "/content/page/es-mx/", nil)
	req.Header.Set("x-profile-id", "dev-1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	// dev fallback authenticates → personalization merge runs (shortcuts/me attach path).
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if _, hasGlobal := resp["globalData"]; !hasGlobal {
		t.Errorf("expected globalData on web page response: %v", resp)
	}
}
