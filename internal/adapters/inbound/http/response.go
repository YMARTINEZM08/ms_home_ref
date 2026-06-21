package handler

import (
	"encoding/json"
	"net/http"

	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// layoutResponse is the JSON contract for GET /home.
// Field names are stable and intentionally decoupled from the domain struct
// names so the API contract can evolve independently.
type layoutResponse struct {
	Blocks []blockResponse `json:"blocks"`
}

// blockResponse represents one block in the layout. Static and dynamic
// variants share the same envelope; fields absent for a given kind are omitted.
type blockResponse struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Type string `json:"type"`

	// Static-only
	Content map[string]any `json:"content,omitempty"`

	// Dynamic-only (placeholder contract — Rule 18)
	ResolveEndpoint string `json:"resolve_endpoint,omitempty"`
	Fallback        string `json:"fallback,omitempty"`
	FeatureFlagID   string `json:"feature_flag_id,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

func toLayoutResponse(layout *domain.Layout) layoutResponse {
	blocks := make([]blockResponse, 0, len(layout.Blocks))
	for _, b := range layout.Blocks {
		blocks = append(blocks, toBlockResponse(b))
	}
	return layoutResponse{Blocks: blocks}
}

func toBlockResponse(b domain.Block) blockResponse {
	switch b.Kind {
	case domain.KindStatic:
		return blockResponse{
			Kind:    string(b.Kind),
			ID:      b.Static.ID,
			Type:    string(b.Static.Type),
			Content: b.Static.Content,
		}
	default: // KindDynamic
		enabled := b.Dynamic.Enabled
		return blockResponse{
			Kind:            string(b.Kind),
			ID:              b.Dynamic.ID,
			Type:            string(b.Dynamic.Type),
			ResolveEndpoint: b.Dynamic.ResolveEndpoint,
			Fallback:        b.Dynamic.Fallback,
			FeatureFlagID:   b.Dynamic.FeatureFlagID,
			Enabled:         &enabled,
		}
	}
}

// writeJSON serialises v as JSON and writes it with the given status code.
// Sets Content-Type and baseline security response headers.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
