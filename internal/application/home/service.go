package home

import (
	"context"
	"log/slog"

	domain "github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

// Compile-time check: Service must satisfy the inbound port.
var _ domain.HomeUseCase = (*Service)(nil)

// Service implements domain.HomeUseCase. It fetches raw blocks from the
// content outbound port, classifies each as static or dynamic, and assembles
// the ordered Layout. It never reorders blocks (Rule 18).
type Service struct {
	content domain.ContentPort
	log     *slog.Logger
}

// NewService constructs a Service with its required dependencies.
func NewService(content domain.ContentPort, log *slog.Logger) *Service {
	return &Service{content: content, log: log}
}

// GetLayout implements domain.HomeUseCase.
//
// Error contract:
//   - Returns the AppError from FetchLayout as-is. The adapter already logged
//     it — this layer must not re-log (logger-handler: log exactly once).
//   - On success, logs one INFO line with layout summary for operators.
func (s *Service) GetLayout(ctx context.Context, req domain.HomeRequest) (*domain.Layout, *domain.AppError) {
	rawBlocks, appErr := s.content.FetchLayout(ctx, req)
	if appErr != nil {
		return nil, appErr
	}

	blocks := make([]domain.Block, 0, len(rawBlocks))
	staticCount, dynamicCount := 0, 0

	for _, raw := range rawBlocks {
		b := classify(raw)
		blocks = append(blocks, b)
		if b.Kind == domain.KindStatic {
			staticCount++
		} else {
			dynamicCount++
		}
	}

	s.log.InfoContext(ctx, "home layout composed",
		"total_blocks", len(blocks),
		"static_blocks", staticCount,
		"dynamic_blocks", dynamicCount,
		"locale", req.Locale,
		"brand", req.Brand,
		"channel", req.Channel,
	)

	return &domain.Layout{Blocks: blocks}, nil
}
