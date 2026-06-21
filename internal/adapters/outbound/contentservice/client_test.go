package contentservice_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/YMARTINEZM08/ms_home_ref/internal/adapters/outbound/contentservice"
	"github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/breaker"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/httpx"
)

var testLog = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

func openBreakerSettings() breaker.Settings {
	return breaker.Settings{
		FailureRatio: 0.01,  // trips at 1% — effectively any failure
		MinRequests:  1,
		OpenTimeout:  5 * time.Second,
	}
}

func defaultSettings() breaker.Settings {
	return breaker.Settings{
		FailureRatio: 0.99,
		MinRequests:  100,
		OpenTimeout:  5 * time.Second,
	}
}

func newClient(t *testing.T, serverURL string, bs breaker.Settings) *contentservice.Client {
	t.Helper()
	return contentservice.NewClient(
		contentservice.Config{
			BaseURL:         serverURL,
			HomePageID:      "tienda/home",
			Timeout:         5 * time.Second,
			BreakerSettings: bs,
		},
		httpx.NewClient(5*time.Second),
		testLog,
	)
}

func TestClient_FetchLayout_Success(t *testing.T) {
	payload := map[string]any{
		"uid": "entry1",
		"template": map[string]any{
			"layout": map[string]any{
				"blocks": []any{
					map[string]any{"_content_type_uid": "banner", "uid": "b1"},
					map[string]any{"_content_type_uid": "products_list", "uid": "p1", "source_of_data": "groupby"},
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	client := newClient(t, srv.URL, defaultSettings())
	blocks, appErr := client.FetchLayout(context.Background(), home.HomeRequest{
		Locale: "es-mx",
		Brand:  "LP",
	})

	if appErr != nil {
		t.Fatalf("unexpected error: %v", appErr)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != "banner" {
		t.Errorf("blocks[0].Type = %q, want banner", blocks[0].Type)
	}
	if blocks[1].SourceOfData != "groupby" {
		t.Errorf("blocks[1].SourceOfData = %q, want groupby", blocks[1].SourceOfData)
	}
}

func TestClient_FetchLayout_Returns404AsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newClient(t, srv.URL, defaultSettings())
	_, appErr := client.FetchLayout(context.Background(), home.HomeRequest{Locale: "es-mx", Brand: "LP"})

	if appErr == nil {
		t.Fatal("expected error for 404")
	}
	if appErr.Code != home.ErrCodeNotFound {
		t.Errorf("error code = %q, want %q", appErr.Code, home.ErrCodeNotFound)
	}
}

func TestClient_FetchLayout_BreakerOpenReturnsServiceUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newClient(t, srv.URL, openBreakerSettings())

	// First call trips the breaker (1 request, 1 failure = 100% > 1%)
	_, _ = client.FetchLayout(context.Background(), home.HomeRequest{Locale: "es-mx", Brand: "LP"})

	// Second call — breaker is open
	_, appErr := client.FetchLayout(context.Background(), home.HomeRequest{Locale: "es-mx", Brand: "LP"})
	if appErr == nil {
		t.Fatal("expected error when breaker is open")
	}
	if appErr.Code != home.ErrCodeServiceUnavailable {
		t.Errorf("error code = %q, want SERVICE_UNAVAILABLE", appErr.Code)
	}
	if !appErr.Retryable {
		t.Error("SERVICE_UNAVAILABLE should be retryable")
	}
}

func TestClient_FetchLayout_InvalidLocaleBadRequest(t *testing.T) {
	client := newClient(t, "http://localhost:9999", defaultSettings())

	_, appErr := client.FetchLayout(context.Background(), home.HomeRequest{
		Locale: "../etc/passwd", // invalid
		Brand:  "LP",
	})
	if appErr == nil {
		t.Fatal("expected error for invalid locale")
	}
	if appErr.Code != home.ErrCodeBadRequest {
		t.Errorf("error code = %q, want BAD_REQUEST", appErr.Code)
	}
}

func TestClient_FetchLayout_URLHostAlwaysFromConfig(t *testing.T) {
	var receivedHost string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uid": "e1",
			"template": map[string]any{
				"layout": map[string]any{"blocks": []any{}},
			},
		})
	}))
	defer srv.Close()

	client := newClient(t, srv.URL, defaultSettings())
	_, _ = client.FetchLayout(context.Background(), home.HomeRequest{Locale: "es-mx", Brand: "LP"})

	// The request must go to the configured server, not anywhere derived from input
	if receivedHost == "" {
		t.Error("server received no request — host is not from config")
	}
}

func TestClient_FetchLayout_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := newClient(t, srv.URL, defaultSettings())
	_, appErr := client.FetchLayout(ctx, home.HomeRequest{Locale: "es-mx", Brand: "LP"})
	if appErr == nil {
		t.Fatal("expected error for cancelled context")
	}
}
