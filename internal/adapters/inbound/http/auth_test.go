package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeExchanger struct {
	claims     map[string]any
	err        error
	lastClient string
}

func (f *fakeExchanger) ExchangeToken(_ context.Context, _, client string) (map[string]any, error) {
	f.lastClient = client
	return f.claims, f.err
}

func reqWithCookie(name, val string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/content/page/es-mx/", nil)
	if val != "" {
		r.AddCookie(&http.Cookie{Name: name, Value: val})
	}
	r.Header.Set("x-brand-id", "lp")
	return r
}

func TestOpaqueAuthenticator(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("logged-in session", func(t *testing.T) {
		ex := &fakeExchanger{claims: map[string]any{"prn": "p1", "isAnonymous": false}}
		a := NewOpaqueAuthenticator(ex, "SessionId", "LP", discard)
		p, loggedIn, claims := a.Authenticate(context.Background(), reqWithCookie("SessionId", "s1"))
		if p != "p1" || !loggedIn || claims["prn"] != "p1" {
			t.Errorf("got p=%s loggedIn=%v claims=%v", p, loggedIn, claims)
		}
		if ex.lastClient != "LP" { // brand upper-cased
			t.Errorf("client = %q, want LP", ex.lastClient)
		}
	})

	t.Run("anonymous token → not logged in", func(t *testing.T) {
		ex := &fakeExchanger{claims: map[string]any{"prn": "anon", "isAnonymous": true}}
		a := NewOpaqueAuthenticator(ex, "SessionId", "LP", discard)
		_, loggedIn, _ := a.Authenticate(context.Background(), reqWithCookie("SessionId", "s1"))
		if loggedIn {
			t.Error("anonymous must not be logged in")
		}
	})

	t.Run("no cookie → anonymous, no exchange", func(t *testing.T) {
		ex := &fakeExchanger{claims: map[string]any{"prn": "p1"}}
		a := NewOpaqueAuthenticator(ex, "SessionId", "LP", discard)
		p, loggedIn, _ := a.Authenticate(context.Background(), reqWithCookie("SessionId", ""))
		if p != "" || loggedIn {
			t.Error("missing cookie should yield anonymous")
		}
	})

	t.Run("exchange error → anonymous", func(t *testing.T) {
		ex := &fakeExchanger{err: errors.New("boom")}
		a := NewOpaqueAuthenticator(ex, "SessionId", "LP", discard)
		_, loggedIn, _ := a.Authenticate(context.Background(), reqWithCookie("SessionId", "s1"))
		if loggedIn {
			t.Error("exchange failure should yield anonymous")
		}
	})
}

type okVerifier struct{ claims map[string]any }

func (v okVerifier) Verify(context.Context, string) (map[string]any, error) { return v.claims, nil }

func TestJWTAuthenticator(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := NewJWTAuthenticator(okVerifier{claims: map[string]any{"profileId": "p9"}}, "profileId", discard)

	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Header.Set("Authorization", "Bearer tok")
	p, loggedIn, claims := a.Authenticate(context.Background(), r)
	if p != "p9" || !loggedIn || claims["profileId"] != "p9" {
		t.Errorf("got p=%s loggedIn=%v", p, loggedIn)
	}

	// No bearer → anonymous.
	if p, loggedIn, _ := a.Authenticate(context.Background(), httptest.NewRequest(http.MethodGet, "/x", nil)); p != "" || loggedIn {
		t.Error("no bearer should be anonymous")
	}
}
