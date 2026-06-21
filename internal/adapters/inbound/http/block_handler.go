package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/YMARTINEZM08/ms_home_ref/internal/application/blocks"
	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
	"github.com/YMARTINEZM08/ms_home_ref/pkg/httpx"
)

type blockHandler struct {
	registry blocks.ResolveUseCase
	log      *slog.Logger
}

func newBlockHandler(registry blocks.ResolveUseCase, log *slog.Logger) *blockHandler {
	return &blockHandler{registry: registry, log: log}
}

// ServeHTTP handles GET /home/blocks/{blockType}.
// Each block type is independently toggleable and failure-isolated — a failure
// here never affects the /home layout endpoint.
func (h *blockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawType := chi.URLParam(r, "blockType")

	// Security: validate against the fixed allowlist before any downstream call.
	blockType, ok := domain.IsAllowedResolveType(rawType)
	if !ok {
		h.log.WarnContext(r.Context(), "unknown block type requested",
			"block_type", rawType,
			"request_id", httpx.RequestIDFromCtx(r.Context()),
		)
		writeError(w, domain.ErrBadRequest("unknown block type: "+rawType))
		return
	}

	params := h.extractParams(r)

	result, appErr := h.registry.ResolveBlock(r.Context(), blockType, params)
	if appErr != nil {
		// Resolver already logged this. Do NOT re-log (log-once rule).
		writeError(w, appErr)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// extractParams collects query parameters and selected safe headers that
// resolvers may need for personalisation or filtering.
// Only an explicit, named set of headers is forwarded — never arbitrary ones.
func (h *blockHandler) extractParams(r *http.Request) map[string]string {
	params := map[string]string{
		"locale":  strings.ToLower(strings.TrimSpace(r.URL.Query().Get("locale"))),
		"brand":   strings.ToUpper(strings.TrimSpace(r.Header.Get("x-brand-id"))),
		"channel": strings.ToLower(strings.TrimSpace(r.URL.Query().Get("channel"))),
	}
	// Strip -PREVIEW from brand if present.
	params["brand"] = strings.TrimSuffix(params["brand"], "-PREVIEW")
	if params["locale"] == "" {
		params["locale"] = "es-mx"
	}
	if params["brand"] == "" {
		params["brand"] = "LP"
	}
	return params
}
