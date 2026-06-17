package domain

import (
	"errors"
	"sync"
	"testing"
)

func TestSalesforceActionMemoizesPerAction(t *testing.T) {
	st := NewRequestState()
	var calls int32
	var mu sync.Mutex
	compute := func(out map[string]any) func() (map[string]any, error) {
		return func() (map[string]any, error) {
			mu.Lock()
			calls++
			mu.Unlock()
			return out, nil
		}
	}

	// Same action computed concurrently → single underlying call.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = st.SalesforceAction("A", compute(map[string]any{"k": "A"}))
		}()
	}
	wg.Wait()

	// Different action → its own call.
	if d, _ := st.SalesforceAction("B", compute(map[string]any{"k": "B"})); d["k"] != "B" {
		t.Errorf("action B result wrong: %v", d)
	}
	if calls != 2 {
		t.Errorf("expected 2 computes (A once, B once), got %d", calls)
	}

	// Cached A returns the original value.
	if d, _ := st.SalesforceAction("A", compute(map[string]any{"k": "OTHER"})); d["k"] != "A" {
		t.Errorf("cached A should return first value, got %v", d)
	}
}

func TestSalesforceActionCachesError(t *testing.T) {
	st := NewRequestState()
	want := errors.New("boom")
	calls := 0
	fn := func() (map[string]any, error) { calls++; return nil, want }

	_, err1 := st.SalesforceAction("X", fn)
	_, err2 := st.SalesforceAction("X", fn)
	if !errors.Is(err1, want) || !errors.Is(err2, want) || calls != 1 {
		t.Errorf("error should be cached once: calls=%d err1=%v err2=%v", calls, err1, err2)
	}
}

func TestNextTagIndex(t *testing.T) {
	st := NewRequestState()
	if st.NextTagIndex() != 1 || st.NextTagIndex() != 2 {
		t.Error("tag index should start at 1 and increment")
	}
}
