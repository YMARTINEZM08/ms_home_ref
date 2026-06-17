package populate

import (
	"testing"

	"ms_home/internal/domain"
)

func bffEvent(eventType string) map[string]any {
	return map[string]any{
		"customData": []any{
			map[string]any{"source": "bff", "type": eventType},
		},
	}
}

func TestProcessEventsIndex(t *testing.T) {
	state := domain.NewRequestState()
	blocks := []any{
		map[string]any{"events": []any{bffEvent("index")}},
		map[string]any{"events": []any{bffEvent("index")}},
	}
	processEvents(state, blocks)

	got := []int{
		blocks[0].(map[string]any)["events"].([]any)[0].(map[string]any)["customData"].([]any)[0].(map[string]any)["value"].(int),
		blocks[1].(map[string]any)["events"].([]any)[0].(map[string]any)["customData"].([]any)[0].(map[string]any)["value"].(int),
	}
	// tag_index starts at 1 and increments; order between the two is deterministic here (sequential).
	if got[0] != 1 || got[1] != 2 {
		t.Errorf("index values = %v, want [1 2]", got)
	}
}

func TestProcessEventsSelectedStore(t *testing.T) {
	state := domain.NewRequestState()
	state.SetSelectedStore(&domain.SelectedStore{ID: 42, Name: "Centro"})
	block := map[string]any{"events": []any{bffEvent("selected_store.name"), bffEvent("selected_store.code")}}
	processEvents(state, []any{block})

	events := block["events"].([]any)
	name := events[0].(map[string]any)["customData"].([]any)[0].(map[string]any)["value"]
	code := events[1].(map[string]any)["customData"].([]any)[0].(map[string]any)["value"]
	if name != "Centro" || code != 42 {
		t.Errorf("store values = %v / %v, want Centro / 42", name, code)
	}
}

func TestProcessEventsNonArrayBecomesEmpty(t *testing.T) {
	state := domain.NewRequestState()
	block := map[string]any{"events": map[string]any{"some": "object"}}
	processEvents(state, []any{block})
	if arr, ok := block["events"].([]any); !ok || len(arr) != 0 {
		t.Errorf("non-array event list should become [], got %v", block["events"])
	}
}

func TestProcessEventsIgnoresNonBff(t *testing.T) {
	state := domain.NewRequestState()
	event := map[string]any{"customData": []any{map[string]any{"source": "client-side", "type": "index"}}}
	block := map[string]any{"events": []any{event}}
	processEvents(state, []any{block})
	if _, hasValue := event["customData"].([]any)[0].(map[string]any)["value"]; hasValue {
		t.Error("non-bff customData should be untouched")
	}
}

func TestProcessEventsNilStateNoop(t *testing.T) {
	block := map[string]any{"events": []any{bffEvent("index")}}
	processEvents(nil, []any{block}) // must not panic
}
