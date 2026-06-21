package blocks

import (
	"context"
	"fmt"
	"log/slog"

	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// Resolver is the per-block contract: given runtime params, return the block's
// detail payload. Each block type has exactly one Resolver registered.
// Implementations must be stateless and safe for concurrent use.
type Resolver interface {
	Resolve(ctx context.Context, params map[string]string) (map[string]any, *domain.AppError)
}

// ResolveUseCase is the inbound port consumed by the HTTP block handler.
type ResolveUseCase interface {
	ResolveBlock(ctx context.Context, blockType domain.BlockType, params map[string]string) (map[string]any, *domain.AppError)
}

// Registry dispatches block resolution to the Resolver registered for each
// BlockType. Unregistered types return NOT_FOUND. One breaker per resolver
// is the responsibility of the concrete Resolver implementation.
type Registry struct {
	resolvers map[domain.BlockType]Resolver
	log       *slog.Logger
}

// Compile-time check.
var _ ResolveUseCase = (*Registry)(nil)

// NewRegistry constructs an empty Registry. Use Register to add resolvers.
func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{
		resolvers: make(map[domain.BlockType]Resolver),
		log:       log,
	}
}

// Register adds a Resolver for blockType. Panics on duplicate registration —
// duplicates are always a programmer error caught at startup, not at runtime.
func (r *Registry) Register(blockType domain.BlockType, resolver Resolver) {
	if _, exists := r.resolvers[blockType]; exists {
		panic(fmt.Sprintf("blocks.Registry: duplicate resolver for block type %q", blockType))
	}
	r.resolvers[blockType] = resolver
}

// ResolveBlock implements ResolveUseCase.
// Errors from resolvers are returned as-is — the resolver is responsible for
// logging its own error exactly once (logger-handler: log once at highest layer).
func (r *Registry) ResolveBlock(
	ctx context.Context,
	blockType domain.BlockType,
	params map[string]string,
) (map[string]any, *domain.AppError) {
	resolver, ok := r.resolvers[blockType]
	if !ok {
		return nil, domain.ErrNotFound(fmt.Sprintf("resolver for block type %q", blockType))
	}

	result, appErr := resolver.Resolve(ctx, params)
	if appErr != nil {
		return nil, appErr
	}

	r.log.InfoContext(ctx, "block resolved",
		"block_type", blockType,
		"locale", params["locale"],
		"brand", params["brand"],
	)
	return result, nil
}

// StubResolver is a development-time stand-in used until a real outbound
// adapter is wired for a given block type. It returns a clearly-marked
// placeholder payload so the service is runnable end-to-end from day one.
type StubResolver struct {
	BlockType domain.BlockType
}

func (s *StubResolver) Resolve(_ context.Context, params map[string]string) (map[string]any, *domain.AppError) {
	return map[string]any{
		"block_type": string(s.BlockType),
		"stub":       true,
		"message":    fmt.Sprintf("resolver for %q is not yet implemented", s.BlockType),
		"params":     params,
	}, nil
}
