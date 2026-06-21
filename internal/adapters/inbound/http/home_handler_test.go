package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	handler "github.com/YMARTINEZM08/ms_home_ref/internal/adapters/inbound/http"
	"github.com/YMARTINEZM08/ms_home_ref/internal/application/blocks"
	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

var silentLog = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.Level(1000)}))

// stubHomeUC implements domain.HomeUseCase for testing.
type stubHomeUC struct {
	layout *domain.Layout
	err    *domain.AppError
}

func (s *stubHomeUC) GetLayout(_ context.Context, _ domain.HomeRequest) (*domain.Layout, *domain.AppError) {
	return s.layout, s.err
}

func newTestRouter(uc domain.HomeUseCase) http.Handler {
	reg := blocks.NewRegistry(silentLog)
	return handler.NewRouter(uc, reg, silentLog, "test")
}

func TestHomeHandler_HappyPath(t *testing.T) {
	enabled := true
	layout := &domain.Layout{
		Blocks: []domain.Block{
			{Kind: domain.KindStatic, Static: &domain.StaticBlock{ID: "s1", Type: domain.BlockTypeBanner, Content: map[string]any{"title": "Hello"}}},
			{Kind: domain.KindDynamic, Dynamic: &domain.DynamicBlock{ID: "d1", Type: domain.BlockTypeProductList, ResolveEndpoint: "/home/blocks/products_list", Enabled: enabled}},
		},
	}
	router := newTestRouter(&stubHomeUC{layout: layout})

	req := httptest.NewRequest(http.MethodGet, "/home?locale=es-mx", nil)
	req.Header.Set("x-brand-id", "LP")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	blocksArr, _ := resp["blocks"].([]any)
	if len(blocksArr) != 2 {
		t.Errorf("blocks count = %d, want 2", len(blocksArr))
	}
}

func TestHomeHandler_InvalidChannel_Returns400(t *testing.T) {
	router := newTestRouter(&stubHomeUC{})

	req := httptest.NewRequest(http.MethodGet, "/home?locale=es-mx&channel=invalid_channel", nil)
	req.Header.Set("x-brand-id", "LP")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w, "BAD_REQUEST")
}

func TestHomeHandler_InvalidBrand_Returns400(t *testing.T) {
	router := newTestRouter(&stubHomeUC{})

	req := httptest.NewRequest(http.MethodGet, "/home?locale=es-mx", nil)
	req.Header.Set("x-brand-id", "<script>alert(1)</script>")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	assertErrorCode(t, w, "BAD_REQUEST")
}

func TestHomeHandler_UseCaseError_PropagatesErrorCode(t *testing.T) {
	appErr := domain.ErrServiceUnavailable("content-service", nil)
	router := newTestRouter(&stubHomeUC{err: appErr})

	req := httptest.NewRequest(http.MethodGet, "/home?locale=es-mx", nil)
	req.Header.Set("x-brand-id", "LP")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	assertErrorCode(t, w, "SERVICE_UNAVAILABLE")
}

func TestHomeHandler_ErrorResponse_NeverLeaksInternalDetail(t *testing.T) {
	appErr := domain.ErrUnexpected("internal op", nil)
	appErr.Detail = "secret internal stack trace"
	appErr.Cause = nil
	router := newTestRouter(&stubHomeUC{err: appErr})

	req := httptest.NewRequest(http.MethodGet, "/home?locale=es-mx", nil)
	req.Header.Set("x-brand-id", "LP")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	if contains(body, "secret internal stack trace") {
		t.Error("response body must not contain internal Detail field")
	}
	if contains(body, "cause") {
		t.Error("response body must not contain cause field")
	}
}

func TestHomeHandler_SecurityHeaders(t *testing.T) {
	layout := &domain.Layout{Blocks: []domain.Block{}}
	router := newTestRouter(&stubHomeUC{layout: layout})

	req := httptest.NewRequest(http.MethodGet, "/home?locale=es-mx", nil)
	req.Header.Set("x-brand-id", "LP")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options: nosniff")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options: DENY")
	}
}

func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp["error_code"] != wantCode {
		t.Errorf("error_code = %v, want %q", resp["error_code"], wantCode)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
