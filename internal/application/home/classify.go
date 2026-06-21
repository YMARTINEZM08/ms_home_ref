package home

import (
	"fmt"

	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

const resolveEndpointPrefix = "/home/blocks"

// classify converts a RawBlock into the domain Block variant.
// Static blocks carry their content inline; dynamic blocks become placeholders
// carrying only the metadata the frontend needs to resolve them independently.
// Classification is purely in-memory — it never panics or returns an error.
func classify(raw domain.RawBlock) domain.Block {
	if !isDynamic(raw) {
		return domain.Block{
			Kind: domain.KindStatic,
			Static: &domain.StaticBlock{
				ID:      raw.ID,
				Type:    raw.Type,
				Content: raw.Fields,
			},
		}
	}

	return domain.Block{
		Kind: domain.KindDynamic,
		Dynamic: &domain.DynamicBlock{
			ID:              raw.ID,
			Type:            raw.Type,
			ResolveEndpoint: fmt.Sprintf("%s/%s", resolveEndpointPrefix, raw.Type),
			Fallback:        fallbackFrom(raw.Fields),
			FeatureFlagID:   raw.FeatureFlagID,
			Enabled:         raw.Enabled,
		},
	}
}

// isDynamic returns true when the block must be resolved at runtime.
// A block is dynamic when:
//   - its type is in the dynamic allowlist, OR
//   - its source_of_data requires a live call (groupby, salesforce, etc.), OR
//   - its handle is "client-side" (legacy BFF signal meaning: skip server population)
func isDynamic(raw domain.RawBlock) bool {
	if domain.IsDynamic(raw.Type) {
		return true
	}
	switch raw.SourceOfData {
	case "groupby", "salesforce", "recently_viewed", "jewel", "lob":
		return true
	}
	return raw.Handle == "client-side"
}

// filterByAudience removes blocks that are gated to a specific audience.
// container_greeting is for logged-in users only; container_guest is for guests
// only. All other blocks are passed through regardless of auth state.
func filterByAudience(blocks []domain.RawBlock, isLoggedIn bool) []domain.RawBlock {
	out := make([]domain.RawBlock, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case domain.BlockTypeGreeting:
			if !isLoggedIn {
				continue
			}
		case domain.BlockTypeGuestContainer:
			if isLoggedIn {
				continue
			}
		}
		out = append(out, b)
	}
	return out
}

// fallbackFrom extracts an optional "fallback" string that the frontend should
// render when the resolve endpoint is unavailable. Defaults to empty.
func fallbackFrom(fields map[string]any) string {
	v, _ := fields["fallback"].(string)
	return v
}
