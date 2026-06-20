package home

import "context"

// HomeUseCase is the inbound port: the contract the HTTP handler depends on.
// Only one implementation exists (application/home.Service); the interface
// keeps the handler decoupled from the application layer.
type HomeUseCase interface {
	GetLayout(ctx context.Context, req HomeRequest) (*Layout, *AppError)
}

// ContentPort is the outbound port: the contract the application layer uses
// to fetch raw block data. Implemented by adapters/outbound/contentservice.
type ContentPort interface {
	FetchLayout(ctx context.Context, req HomeRequest) ([]RawBlock, *AppError)
}

// BlockResolverPort is the outbound port for resolving a single dynamic block's
// detail data. Implemented per block type in adapters/outbound/<downstream>.
type BlockResolverPort interface {
	Resolve(ctx context.Context, blockType BlockType, params map[string]string) (map[string]any, *AppError)
}

// RawBlock is the unprocessed block shape returned by the content-service.
// It is an infrastructure DTO that lives at the domain boundary — adapters
// map it into StaticBlock or DynamicBlock inside the application layer.
type RawBlock struct {
	ID             string
	Type           BlockType
	FeatureFlagID  string
	Enabled        bool
	SourceOfData   string         // groupby | salesforce | recently_viewed | jewel | lob
	AudienceFilter string         // all | logged | guest
	Handle         string         // bff | client-side
	Fields         map[string]any // all other Contentstack fields
}
