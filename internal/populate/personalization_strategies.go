package populate

import (
	"context"

	"ms_home/internal/domain"
)

// ContainerGuest — port of ContainerGuestPopulateStrategy. Shown only to guests
// (anonymous) when personalization is on; dropped for logged-in users.
type ContainerGuest struct{}

func (ContainerGuest) Supports(b Block) bool { return b["_content_type_uid"] == "container_guest" }

func (ContainerGuest) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("personalization") {
		return nil, nil
	}
	if !ri.LoggedIn {
		return b, nil // guest → keep
	}
	return nil, nil // logged in → drop
}

// ContainerShortcuts — port of ContainerShortcutsPopulateStrategy. Logged-in only;
// flattens shortcut_items (primary/custom) into {type, ...data}.
type ContainerShortcuts struct{}

func (ContainerShortcuts) Supports(b Block) bool {
	return b["_content_type_uid"] == "container_shortcuts"
}

func (ContainerShortcuts) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.LoggedIn {
		return nil, nil
	}
	if items, ok := b["shortcut_items"].([]any); ok {
		mapped := make([]any, 0, len(items))
		for _, entry := range items {
			em, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if p, ok := em["primary"].(map[string]any); ok {
				mapped = append(mapped, withType("primary", p))
			} else if c, ok := em["custom"].(map[string]any); ok {
				mapped = append(mapped, withType("custom", c))
			}
		}
		b["shortcut_items"] = mapped
	} else {
		delete(b, "shortcut_items")
	}
	return b, nil
}

// withType returns {type, ...data}.
func withType(kind string, data map[string]any) map[string]any {
	out := make(map[string]any, len(data)+1)
	out["type"] = kind
	for k, v := range data {
		out[k] = v
	}
	return out
}
