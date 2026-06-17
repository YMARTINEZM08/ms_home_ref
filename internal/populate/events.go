package populate

import "ms_home/internal/domain"

// eventListNames mirrors EventListNames (constant/event.ts).
var eventListNames = []string{"button_events", "events", "dot_events"}

// processEvents fills BFF-sourced custom-data values (index, selected store) in the
// event lists of a block tree. Faithful port of processBffCustomDataEvents.
func processEvents(state *domain.RequestState, blocks []any) {
	if state == nil {
		return
	}
	store := state.SelectedStore()
	for _, name := range eventListNames {
		for _, b := range blocks {
			recurseEvents(state, store, name, b)
		}
	}
}

func recurseEvents(state *domain.RequestState, store *domain.SelectedStore, name string, node any) {
	switch v := node.(type) {
	case []any:
		for _, item := range v {
			recurseEvents(state, store, name, item)
		}
	case map[string]any:
		if el, ok := v[name]; ok {
			if arr, ok := el.([]any); ok {
				for _, ev := range arr {
					em, ok := ev.(map[string]any)
					if !ok {
						continue
					}
					if cd, ok := em["customData"].([]any); ok {
						for _, d := range cd {
							if dm, ok := d.(map[string]any); ok && dm["source"] == "bff" {
								applyProcessor(state, store, dm)
							}
						}
					}
				}
			} else {
				v[name] = []any{} // non-array event list → []
			}
		}
		for _, val := range v {
			switch val.(type) {
			case map[string]any, []any:
				recurseEvents(state, store, name, val)
			}
		}
	}
}

// applyProcessor sets customEvent.value based on its type (processor()).
func applyProcessor(state *domain.RequestState, store *domain.SelectedStore, event map[string]any) {
	switch event["type"] {
	case "index":
		event["value"] = state.NextTagIndex()
	case "selected_store.name":
		if store != nil {
			event["value"] = store.Name
		} else {
			event["value"] = nil
		}
	case "selected_store.code":
		if store != nil {
			event["value"] = store.ID
		} else {
			event["value"] = nil
		}
	}
}
