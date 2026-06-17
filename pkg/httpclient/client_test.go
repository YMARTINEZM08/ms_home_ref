package httpclient

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestToCurlMasksSensitiveHeaders(t *testing.T) {
	got := toCurl(http.MethodGet, "https://x/y", map[string]string{
		"Authorization": "Bearer secret",
		"x-brand-id":    "LP",
	}, nil)

	if strings.Contains(got, "secret") {
		t.Errorf("cURL leaked secret: %s", got)
	}
	if !strings.Contains(got, "Authorization: ***") {
		t.Errorf("Authorization not masked: %s", got)
	}
	if !strings.Contains(got, "x-brand-id: LP") {
		t.Errorf("non-sensitive header dropped: %s", got)
	}
}

func TestDoContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if _, err := c.Do(ctx, http.MethodGet, srv.URL, nil, nil); err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}

func TestDoSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	resp, err := c.Do(context.Background(), http.MethodGet, srv.URL, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK || string(resp.Body) != `{"ok":true}` {
		t.Errorf("unexpected response: %d %s", resp.Status, resp.Body)
	}
}
