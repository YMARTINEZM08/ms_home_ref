package contentservice

// normalize flattens the raw block list returned by the content-service into
// a uniform []map[string]any, mirroring the contract of the legacy
// libs/providers/src/utils/block.utils.ts (rule 17/19: preserve external
// integration contract, never migrate business logic).
//
// Content-service layout items have the shape:
//
//	{ "<content_type_uid>": { "_content_type_uid": "...", "uid": "...", ... } }
//
// This function unwraps the outer key, infers _content_type_uid if absent,
// and recursively flattens container_grid and tabs_container.
func normalize(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, unwrap(m)...)
	}
	return out
}

// unwrap extracts the block from its wrapper key and handles special containers.
func unwrap(m map[string]any) []map[string]any {
	// If _content_type_uid is already at the top level the block is already
	// normalised — return as-is.
	if _, hasUID := m["_content_type_uid"]; hasUID {
		return handleContainer(m)
	}

	// Otherwise the block is wrapped: { "banner": { ... } }
	// There should be exactly one key.
	for key, val := range m {
		inner, ok := val.(map[string]any)
		if !ok {
			continue
		}
		if _, hasUID := inner["_content_type_uid"]; !hasUID {
			inner["_content_type_uid"] = key
		}
		return handleContainer(inner)
	}
	return nil
}

// handleContainer expands container_grid and tabs_container into their
// constituent blocks, or returns the block as a single-element slice.
func handleContainer(block map[string]any) []map[string]any {
	uid, _ := block["_content_type_uid"].(string)

	switch uid {
	case "container_grid":
		return flattenGrid(block)
	case "tabs_container":
		return flattenTabs(block)
	default:
		return []map[string]any{block}
	}
}

// flattenGrid expands container_grid by flattening its grid_items.
func flattenGrid(block map[string]any) []map[string]any {
	items, _ := block["grid_items"].([]any)
	if len(items) == 0 {
		return []map[string]any{block}
	}
	return normalize(items)
}

// flattenTabs normalises tabs_container by recursively normalising each tab's
// content, keeping the container block itself as the wrapper.
func flattenTabs(block map[string]any) []map[string]any {
	tabs, _ := block["tabs"].([]any)
	for i, t := range tabs {
		tab, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if content, ok := tab["content"].([]any); ok {
			tab["content"] = normalize(content)
		}
		tabs[i] = tab
	}
	block["tabs"] = tabs
	return []map[string]any{block}
}
