package populate

// Shared accessors for navigating dynamic JSON maps.

func mapAt(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func sliceAt(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	return v
}

func firstMap(s []any) map[string]any {
	if len(s) > 0 {
		m, _ := s[0].(map[string]any)
		return m
	}
	return nil
}
