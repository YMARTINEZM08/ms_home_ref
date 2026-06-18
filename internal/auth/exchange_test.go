package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ms_home/pkg/httpclient"
)

func TestExchangeToken(t *testing.T) {
	var gotSession, gotBrand, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSession = r.URL.Query().Get("SessionId")
		gotBrand = r.Header.Get("x-brand-id")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"accessToken": "x",
			"decodeAccessToken": map[string]any{
				"prn": "p123", "isAnonymous": false, "isSignUp": true, "phoneHash": "abc",
			},
		})
	}))
	defer srv.Close()

	c := httpclient.New(2*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ex := NewExchange(c, srv.URL, "SessionId")

	claims, err := ex.ExchangeToken(context.Background(), "sess-1", "LP")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if gotPath != "/v2/auth/exchange-token" || gotSession != "sess-1" || gotBrand != "LP" {
		t.Errorf("request wrong: path=%s session=%s brand=%s", gotPath, gotSession, gotBrand)
	}
	if claims["prn"] != "p123" || claims["isAnonymous"] != false || claims["isSignUp"] != true {
		t.Errorf("decodeAccessToken claims wrong: %v", claims)
	}
}

func TestExchangeTokenErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	c := httpclient.New(2*time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ex := NewExchange(c, srv.URL, "SessionId")
	if _, err := ex.ExchangeToken(context.Background(), "bad", "LP"); err == nil {
		t.Error("expected error on non-2xx exchange")
	}
}
