package content

// LegacyWelcomeContainer builds the container_welcome block injected for old Android
// versions. Faithful port of legacyWelcomeContainer: it derives legacy shortcuts from
// the active primary entries of an already-populated container_shortcuts block.
func LegacyWelcomeContainer(blocks []any) map[string]any {
	var shortcutItems []any
	for _, b := range blocks {
		m, ok := b.(map[string]any)
		if !ok || m["_content_type_uid"] != "container_shortcuts" {
			continue
		}
		if items, ok := m["shortcut_items"].([]any); ok {
			shortcutItems = items
		}
		break
	}

	shortcuts := make([]any, 0, len(shortcutItems))
	for _, s := range shortcutItems {
		sm, ok := s.(map[string]any)
		if !ok || sm["type"] != "primary" {
			continue
		}
		if active, _ := sm["is_active"].(bool); !active {
			continue
		}
		shortcuts = append(shortcuts, map[string]any{
			"label": sm["title"],
			"value": sm["primary_id"],
			"image": nil,
		})
	}

	return map[string]any{
		"_content_type_uid": "container_welcome",
		"uid":               "cwhrdcdduid",
		"title":             "",
		"events":            []any{},
		"shortcuts":         shortcuts,
	}
}
