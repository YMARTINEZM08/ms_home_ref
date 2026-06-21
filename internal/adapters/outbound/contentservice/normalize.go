package contentservice

// normalize flattens the raw block list returned by the content-service into
// a uniform []map[string]any, mirroring the contract of the legacy
// libs/providers/src/utils/block.utils.ts (rule 17/19: preserve external
// integration contract, never migrate business logic).
//
// The real content-service layout items use one of two wrapper shapes:
//
//  1. Map-wrapped:  { "banner": { "_content_type_uid": "...", "uid": "...", ... } }
//  2. List-wrapped: { "hero_banner_slider": [ item1, item2, ... ] }
//
// This function unwraps the outer key, infers _content_type_uid if absent,
// expands list-wrapped entries into one block per item, and recursively
// flattens container_grid and tabs_container.
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

// unwrap extracts one or more blocks from a wrapper object and handles special
// containers.  Two shapes are supported:
//
//   - Map-wrapped:  { "type": { ...fields... } } — produces one block
//   - List-wrapped: { "type": [ item1, item2 ] } — produces one block per item
func unwrap(m map[string]any) []map[string]any {
	// Already normalised — _content_type_uid is at the top level.
	if _, hasUID := m["_content_type_uid"]; hasUID {
		return handleContainer(m)
	}

	// Wrapped block: exactly one key whose value is either a map or a list.
	for key, val := range m {
		switch v := val.(type) {
		case map[string]any:
			// Map-wrapped: { "banner": { ...fields... } }
			if _, hasUID := v["_content_type_uid"]; !hasUID {
				v["_content_type_uid"] = key
			}
			return handleContainer(v)

		case []any:
			// List-wrapped: { "hero_banner_slider": [ item1, item2, ... ] }
			// Expand each list item into an independent block of this type.
			out := make([]map[string]any, 0, len(v))
			for _, raw := range v {
				item, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				if _, hasUID := item["_content_type_uid"]; !hasUID {
					item["_content_type_uid"] = key
				}
				out = append(out, handleContainer(item)...)
			}
			return out
		}
	}
	return nil
}

// handleContainer expands container_grid and tabs_container into their
// constituent blocks, normalises nested blocks inside container, or returns
// the block as a single-element slice for all other types.
func handleContainer(block map[string]any) []map[string]any {
	uid, _ := block["_content_type_uid"].(string)

	switch uid {
	case "container_grid":
		return flattenGrid(block)
	case "container":
		return flattenContainer(block)
	case "tabs_container":
		return flattenTabs(block)
	default:
		return []map[string]any{block}
	}
}

// flattenContainer normalises the nested blocks inside a container block,
// keeping the container itself as the top-level entry (mirrors flattenTabs).
// The real CMS container type stores its children under a "blocks" field.
func flattenContainer(block map[string]any) []map[string]any {
	sub, _ := block["blocks"].([]any)
	if len(sub) > 0 {
		block["blocks"] = normalize(sub)
	}
	return []map[string]any{block}
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
