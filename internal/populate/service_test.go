package populate

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func newService(strategies ...Strategy) *Service {
	return NewService(NewRegistry(strategies...), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestPopulateAll(t *testing.T) {
	svc := newService(DefaultStrategies()...)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)

	blocks := []any{
		map[string]any{"_content_type_uid": "container"},                                     // populated, columns default
		map[string]any{"_content_type_uid": "countdown", "is_active": true, "timer": future}, // kept
		map[string]any{"_content_type_uid": "countdown", "is_active": true, "timer": past},   // dropped (expired)
		map[string]any{"_content_type_uid": "countdown", "is_active": false},                 // dropped (inactive)
		map[string]any{"_content_type_uid": "unknown_block", "x": 1},                         // kept unchanged (no strategy)
	}

	out := svc.PopulateAll(context.Background(), blocks)
	if len(out) != 3 {
		t.Fatalf("expected 3 surviving blocks, got %d: %v", len(out), out)
	}
	if c := out[0].(map[string]any); c["columns_mobile_small"] != 2 {
		t.Errorf("container default not applied: %v", c)
	}
}

type errStrategy struct{}

func (errStrategy) Supports(b Block) bool { return b["_content_type_uid"] == "boom" }
func (errStrategy) Populate(context.Context, Block) (Block, error) {
	return nil, context.DeadlineExceeded
}

func TestPopulateDropsOnError(t *testing.T) {
	svc := newService(errStrategy{})
	out := svc.PopulateAll(context.Background(), []any{
		map[string]any{"_content_type_uid": "boom"},
	})
	if len(out) != 0 {
		t.Errorf("errored block should be dropped, got %v", out)
	}
}

func TestPopulatePreservesOrder(t *testing.T) {
	svc := newService(DefaultStrategies()...)
	out := svc.PopulateAll(context.Background(), []any{
		map[string]any{"_content_type_uid": "a"},
		map[string]any{"_content_type_uid": "b"},
		map[string]any{"_content_type_uid": "c"},
	})
	for i, want := range []string{"a", "b", "c"} {
		if got := out[i].(map[string]any)["_content_type_uid"]; got != want {
			t.Errorf("order[%d] = %v, want %v", i, got, want)
		}
	}
}
