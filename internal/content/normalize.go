// Package content holds pure, deterministic CMS transforms ported from
// digital_bff (block normalization, content-type gating, key rename/delete).
// No infrastructure, no external calls.
package content

// NormalizeDoubleBlocks flattens Contentstack modular blocks. Faithful port of
// normalizeDoubleBlocks (libs/providers/src/utils/block.ts). Each block is a
// single-key object {id: data}; nested {id: data[id]} arrays are unwrapped.
func NormalizeDoubleBlocks(blocks []any) []any {
	out := make([]any, 0, len(blocks))
	for _, raw := range blocks {
		block, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		id, data := firstEntry(block)
		dm, dataIsMap := data.(map[string]any)

		if dataIsMap {
			if nested, has := dm[id]; has {
				// { ...nestedData[0], ...rest }
				rest := mapExcept(dm, id)
				merged := mergeMaps(firstOfArray(nested), rest)
				if id == "container_grid" {
					out = append(out, flattenContainerGrid(merged))
				} else {
					out = append(out, merged)
				}
				continue
			}
		}

		// Fallback: { _content_type_uid: id, uid: data._metadata?.uid, ...data }
		res := map[string]any{"_content_type_uid": id}
		if dataIsMap {
			if meta, ok := dm["_metadata"].(map[string]any); ok {
				res["uid"] = meta["uid"]
			}
			for k, v := range dm {
				res[k] = v // data spread overrides the fields above
			}
		}
		out = append(out, res)
	}
	return out
}

// flattenContainerGrid mirrors flattenContainerGrid (block.ts).
func flattenContainerGrid(data map[string]any) map[string]any {
	grid, _ := data["item_grid"].([]any)
	flat := make([]any, 0, len(grid))
	for _, gi := range grid {
		gim, ok := gi.(map[string]any)
		if !ok {
			flat = append(flat, gi)
			continue
		}
		itemType, itemData := firstEntry(gim)
		idm, _ := itemData.(map[string]any)
		res := map[string]any{"item_grid_type": itemType}
		if blockArr, ok := idm["block"].([]any); ok && len(blockArr) > 0 {
			if bm, ok := blockArr[0].(map[string]any); ok {
				for k, v := range bm {
					res[k] = v
				}
			}
		}
		for k, v := range mapExcept(idm, "block") {
			res[k] = v
		}
		flat = append(flat, res)
	}
	out := cloneMap(data)
	out["item_grid"] = flat
	return out
}

// firstEntry returns the single key/value of a modular block (insertion order is
// irrelevant because CMS modular blocks carry exactly one key).
func firstEntry(m map[string]any) (string, any) {
	for k, v := range m {
		return k, v
	}
	return "", nil
}

// firstOfArray returns nested[0] as a map, or an empty map (mirrors spread of
// possibly-undefined nestedData[0]).
func firstOfArray(v any) map[string]any {
	if arr, ok := v.([]any); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

func mapExcept(m map[string]any, except string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if k == except {
			continue
		}
		out[k] = v
	}
	return out
}

// mergeMaps returns base overlaid by over (over wins), matching {...base, ...over}.
func mergeMaps(base, over map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
