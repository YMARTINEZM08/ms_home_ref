package home

// BlockType identifies the content type of a block returned by the content-service.
// Add new types here without touching existing ones (Open/Closed Principle).
type BlockType string

const (
	// Static blocks — no session dependency, cacheable.
	BlockTypeBanner        BlockType = "banner"
	BlockTypeCarousel      BlockType = "carousel"
	BlockTypeHeroBanner    BlockType = "hero_banner"
	BlockTypePromoBar      BlockType = "promo_bar"
	BlockTypeStaticContent BlockType = "static_content"
	BlockTypeForm          BlockType = "form"
	BlockTypeComparePage   BlockType = "comparepage"
	BlockTypeSearchBanners BlockType = "search_banners"
	BlockTypeCountdown     BlockType = "countdown"

	// Dynamic blocks — session/runtime dependent, returned as placeholders.
	BlockTypeProductsList    BlockType = "products_list"
	BlockTypeBannerProducts  BlockType = "banner_products"
	BlockTypeContainerGrid   BlockType = "container_grid"
	BlockTypeGreeting        BlockType = "container_greeting"
	BlockTypeGuestContainer  BlockType = "container_guest"
	BlockTypeShortcuts       BlockType = "container_shortcuts"
	BlockTypeRecommendations BlockType = "recommendation_product_list"
	BlockTypeProductCards    BlockType = "products_cards"
)

// dynamicBlockTypes is the authoritative allowlist of block types that must be
// returned as placeholders. Evaluated in classify.go.
var dynamicBlockTypes = map[BlockType]bool{
	BlockTypeProductsList:    true,
	BlockTypeBannerProducts:  true,
	BlockTypeGreeting:        true,
	BlockTypeGuestContainer:  true,
	BlockTypeShortcuts:       true,
	BlockTypeRecommendations: true,
	BlockTypeProductCards:    true,
}

// IsDynamic reports whether the block type requires runtime/session resolution.
func IsDynamic(t BlockType) bool {
	return dynamicBlockTypes[t]
}

// allowedBlockTypes is the inbound allowlist for {blockType} path parameters.
var allowedBlockTypes = map[BlockType]bool{
	BlockTypeProductsList:    true,
	BlockTypeBannerProducts:  true,
	BlockTypeGreeting:        true,
	BlockTypeGuestContainer:  true,
	BlockTypeShortcuts:       true,
	BlockTypeRecommendations: true,
	BlockTypeProductCards:    true,
}

// IsAllowedResolveType reports whether a blockType string is a valid, known
// resolve endpoint target. Prevents raw user input from reaching adapters.
func IsAllowedResolveType(raw string) (BlockType, bool) {
	t := BlockType(raw)
	return t, allowedBlockTypes[t]
}
