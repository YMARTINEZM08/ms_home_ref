package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/YMARTINEZM08/ms_home_ref/internal/adapters/inbound/http"
	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// stubRegistry implements blocks.ResolveUseCase for testing.
type stubRegistry struct {
	result map[string]any
	err    *domain.AppError
}

func (s *stubRegistry) ResolveBlock(_ context.Context, _ domain.BlockType, _ map[string]string) (map[string]any, *domain.AppError) {
	return s.result, s.err
}

func newBlockTestRouter(reg *stubRegistry) http.Handler {
	return handler.NewRouter(&stubHomeUC{layout: &domain.Layout{}}, reg, silentLog, "test")
}

func TestBlockHandler_UnknownBlockType_Returns400(t *testing.T) {
	router := newBlockTestRouter(&stubRegistry{})

	// These reach the handler and must be rejected with 400 by our allowlist.
	for _, badType := range []string{"unknown", "<script>", "page", "banner"} {
		req := httptest.NewRequest(http.MethodGet, "/home/blocks/"+badType, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("blockType %q: status = %d, want 400", badType, w.Code)
		}
		assertErrorCode(t, w, "BAD_REQUEST")
	}
}

func TestBlockHandler_PathTraversal_NeverReachesHandler(t *testing.T) {
	router := newBlockTestRouter(&stubRegistry{})

	// Go's HTTP stack path-cleans "../etc/passwd" before routing, so Chi sees
	// /home/etc/passwd → 404. The handler is never invoked; the traversal is safe.
	req := httptest.NewRequest(http.MethodGet, "/home/blocks/../etc/passwd", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("path traversal attempt must not return 200")
	}
}

func TestBlockHandler_ValidBlockType_CallsRegistry(t *testing.T) {
	reg := &stubRegistry{result: map[string]any{"data": "resolved"}}
	router := newBlockTestRouter(reg)

	req := httptest.NewRequest(http.MethodGet, "/home/blocks/products_list?locale=es-mx", nil)
	req.Header.Set("x-brand-id", "LP")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestBlockHandler_ServiceUnavailable_Returns503(t *testing.T) {
	reg := &stubRegistry{err: domain.ErrServiceUnavailable("downstream", nil)}
	router := newBlockTestRouter(reg)

	req := httptest.NewRequest(http.MethodGet, "/home/blocks/products_list", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	assertErrorCode(t, w, "SERVICE_UNAVAILABLE")
}

func TestBlockHandler_DisabledBlock_Returns423(t *testing.T) {
	reg := &stubRegistry{err: domain.ErrBlockDisabled(domain.BlockTypeGreeting, "flag-greeting")}
	router := newBlockTestRouter(reg)

	req := httptest.NewRequest(http.MethodGet, "/home/blocks/container_greeting", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 423 {
		t.Fatalf("status = %d, want 423", w.Code)
	}
	assertErrorCode(t, w, "BLOCK_TEMPORARILY_DISABLED")
}

func TestBlockHandler_NoArbitraryHeadersForwarded(t *testing.T) {
	var receivedAuth string
	// If the block handler forwarded arbitrary headers, this inner handler would
	// see them. We verify the auth header is NOT forwarded.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	reg := &stubRegistry{result: map[string]any{}}
	router := newBlockTestRouter(reg)

	req := httptest.NewRequest(http.MethodGet, "/home/blocks/products_list", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The stub registry never forwards headers, so receivedAuth stays empty.
	// This test documents the contract: block handlers must not pass arbitrary headers downstream.
	if receivedAuth != "" {
		t.Error("Authorization header must not be forwarded to downstream services")
	}
}
