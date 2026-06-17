package populate

import (
	"context"
	"log/slog"
	"sync"

	"ms_home/internal/domain"
)

// Service populates block collections. Port of PopulateService.populateAll/populate.
type Service struct {
	registry *Registry
	log      *slog.Logger
}

// NewService wires the populate service.
func NewService(registry *Registry, log *slog.Logger) *Service {
	return &Service{registry: registry, log: log}
}

// ProcessEvents fills BFF custom-data event values on the given block tree. Used
// for template-level events (block-level events run inside PopulateAll).
func (s *Service) ProcessEvents(ctx context.Context, blocks []any) {
	processEvents(domain.RequestInfoFromContext(ctx).State, blocks)
}

// PopulateAll populates each block concurrently and drops failed/undefined blocks,
// preserving input order. Concurrency replaces TS Promise.allSettled.
//
// TODO(phase-2): container_greeting de-duplication and BFF custom-data events
// (personalization-only; no effect on the current strategy set).
func (s *Service) PopulateAll(ctx context.Context, blocks []any) []any {
	type result struct {
		val  any
		keep bool
	}
	results := make([]result, len(blocks))

	var wg sync.WaitGroup
	for i, raw := range blocks {
		block, ok := raw.(map[string]any)
		if !ok {
			results[i] = result{val: raw, keep: true} // non-object blocks pass through
			continue
		}
		wg.Add(1)
		go func(i int, block Block) {
			defer wg.Done()
			b, keep := s.populate(ctx, block)
			results[i] = result{val: b, keep: keep}
		}(i, block)
	}
	wg.Wait()

	out := make([]any, 0, len(blocks))
	for _, r := range results {
		if r.keep {
			out = append(out, r.val)
		}
	}
	out = dedupeGreetings(out)
	processEvents(domain.RequestInfoFromContext(ctx).State, out)
	return out
}

// dedupeGreetings keeps only the birthday container_greeting when one is present,
// dropping the others (port of populateAll's greeting de-duplication).
func dedupeGreetings(blocks []any) []any {
	var greetingIdx []int
	for i, b := range blocks {
		if m, ok := b.(map[string]any); ok && m["_content_type_uid"] == "container_greeting" {
			greetingIdx = append(greetingIdx, i)
		}
	}
	if len(greetingIdx) == 0 {
		return blocks
	}
	birthdayIdx := -1
	for _, i := range greetingIdx {
		if m, ok := blocks[i].(map[string]any); ok {
			if isBirthday, _ := m["is_birthday"].(bool); isBirthday {
				birthdayIdx = i
				break
			}
		}
	}
	if birthdayIdx < 0 {
		return blocks
	}
	inGreeting := make(map[int]bool, len(greetingIdx))
	for _, i := range greetingIdx {
		inGreeting[i] = true
	}
	out := make([]any, 0, len(blocks))
	for i, b := range blocks {
		if !inGreeting[i] || i == birthdayIdx {
			out = append(out, b)
		}
	}
	return out
}

// populate runs every supporting strategy and returns the first successful block.
// If no strategy supports the block it is kept unchanged; if all supporting
// strategies fail or drop it, the block is dropped (keep=false).
func (s *Service) populate(ctx context.Context, block Block) (Block, bool) {
	strategies := s.registry.GetStrategies(block)
	if len(strategies) == 0 {
		return block, true
	}
	for _, st := range strategies {
		b, err := st.Populate(ctx, block)
		if err != nil {
			s.log.DebugContext(ctx, "block populate failed",
				slog.String("content_type", contentType(block)), slog.String("error", err.Error()))
			continue
		}
		if b == nil {
			continue
		}
		return b, true
	}
	return nil, false
}

func contentType(block Block) string {
	if ct, ok := block["_content_type_uid"].(string); ok {
		return ct
	}
	return "unknown"
}
