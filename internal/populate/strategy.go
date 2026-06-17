// Package populate ports digital_bff's PopulateService + strategy registry:
// per-block strategies, parallel execution, and drop-on-failure semantics.
package populate

import "context"

// Block is a single content block (dynamic CMS shape).
type Block = map[string]any

// Strategy populates one kind of block. Mirrors PopulateStrategy (base.strategy.ts).
//
// Populate returns (nil, nil) to drop the block (TS `return undefined`), the
// block to keep it, or an error to signal failure (also dropped, logged at DEBUG).
type Strategy interface {
	Supports(block Block) bool
	Populate(ctx context.Context, block Block) (Block, error)
}
