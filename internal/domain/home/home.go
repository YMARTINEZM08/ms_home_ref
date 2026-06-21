package home

// Layout is the ordered list of blocks for the home page.
// Order is always preserved from the content-service response (Rule 18).
type Layout struct {
	Blocks []Block
}

// Block is a discriminated union: either a StaticBlock or a DynamicBlock.
// Use the Kind field to determine which concrete type is populated.
type Block struct {
	Kind    BlockKind
	Static  *StaticBlock
	Dynamic *DynamicBlock
}

// BlockKind discriminates the two variants of Block.
type BlockKind string

const (
	KindStatic  BlockKind = "static"
	KindDynamic BlockKind = "dynamic"
)

// StaticBlock carries fully resolved Contentstack content.
// It has no session dependency and is eligible for long-lived caching.
type StaticBlock struct {
	ID      string
	Type    BlockType
	Content map[string]any // raw Contentstack fields passed through as-is
}

// DynamicBlock is a placeholder returned to the frontend.
// The frontend calls ResolveEndpoint when it needs the actual content.
// This satisfies Rule 18: Home orchestrates layout only, never personalization.
type DynamicBlock struct {
	ID              string
	Type            BlockType
	ResolveEndpoint string // e.g. /home/blocks/products_list
	Fallback        string // frontend fallback when the endpoint is unavailable
	FeatureFlagID   string
	Enabled         bool
}

// HomeRequest carries the validated, sanitised inputs for a home page composition.
type HomeRequest struct {
	Locale     string
	Brand      string
	Channel    string // optional: pocket | kiosk | mpos
	Preview    bool
	IsLoggedIn bool   // true when the caller is an authenticated user
}
