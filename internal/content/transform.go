package content

// RenameKeys moves truthy template[old] to template[new] (port of renameKeys,
// content.utils.ts). Nil/absent values are skipped, matching the TS `!!` guard.
func RenameKeys(template map[string]any, keys map[string]string) {
	for oldKey, newKey := range keys {
		if v, ok := template[oldKey]; ok && v != nil {
			template[newKey] = v
			delete(template, oldKey)
		}
	}
}

// DeleteKeys removes the given keys from the template (port of deleteKeys).
func DeleteKeys(template map[string]any, keys []string) {
	for _, k := range keys {
		delete(template, k)
	}
}
