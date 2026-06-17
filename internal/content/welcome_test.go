package content

import "testing"

func TestLegacyWelcomeContainer(t *testing.T) {
	blocks := []any{
		map[string]any{
			"_content_type_uid": "container_shortcuts",
			"shortcut_items": []any{
				map[string]any{"type": "primary", "title": "Pedidos", "is_active": true, "primary_id": "orders"},
				map[string]any{"type": "primary", "title": "Inactivo", "is_active": false, "primary_id": "x"},
				map[string]any{"type": "custom", "title": "Custom"},
			},
		},
	}
	w := LegacyWelcomeContainer(blocks)

	if w["_content_type_uid"] != "container_welcome" || w["uid"] != "cwhrdcdduid" {
		t.Fatalf("header wrong: %v", w)
	}
	shortcuts := w["shortcuts"].([]any)
	if len(shortcuts) != 1 { // only active primary
		t.Fatalf("want 1 shortcut, got %d", len(shortcuts))
	}
	sc := shortcuts[0].(map[string]any)
	if sc["label"] != "Pedidos" || sc["value"] != "orders" || sc["image"] != nil {
		t.Errorf("shortcut wrong: %v", sc)
	}
}

func TestLegacyWelcomeContainerNoShortcuts(t *testing.T) {
	w := LegacyWelcomeContainer([]any{map[string]any{"_content_type_uid": "container"}})
	if len(w["shortcuts"].([]any)) != 0 {
		t.Error("expected empty shortcuts when no container_shortcuts present")
	}
}
