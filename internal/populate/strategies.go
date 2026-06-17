package populate

import (
	"context"
	"time"
)

// DefaultStrategies returns the deterministic (no external dependency) strategies
// ported so far. GroupBy/Salesforce/Middleware-backed strategies are added in
// later phases (see docs/todos.md).
func DefaultStrategies() []Strategy {
	return []Strategy{
		Container{},
		Countdown{},
		ContainerGuest{},
		ContainerShortcuts{},
	}
}

// Container — port of ContainerPopulateStrategy. Defaults columns_mobile_small to 2.
type Container struct{}

func (Container) Supports(b Block) bool { return b["_content_type_uid"] == "container" }

func (Container) Populate(_ context.Context, b Block) (Block, error) {
	if v, ok := b["columns_mobile_small"]; !ok || v == nil {
		b["columns_mobile_small"] = 2
	}
	return b, nil
}

// Countdown — port of CountdownPopulateStrategy. Drops inactive or expired timers.
type Countdown struct{}

func (Countdown) Supports(b Block) bool { return b["_content_type_uid"] == "countdown" }

func (Countdown) Populate(_ context.Context, b Block) (Block, error) {
	if active, _ := b["is_active"].(bool); !active {
		return nil, nil // drop
	}
	// TS: +new Date(timer) < Date.now(). An unparseable timer yields NaN (never <),
	// so we only drop when the timer parses and is in the past.
	if timer, ok := b["timer"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timer); err == nil && t.Before(time.Now()) {
			return nil, nil // drop
		}
	}
	return b, nil
}
