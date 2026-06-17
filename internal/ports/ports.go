// Package ports declares the interfaces the application depends on. Adapters
// implement them; the application never imports adapters (skill Rule 1).
package ports

import (
	"context"

	"ms_home/internal/domain"
)

// ContentPort fetches CMS content through the Content Service proxy.
// Mirrors digital_bff ContentProvider.getContent.
type ContentPort interface {
	GetContent(ctx context.Context, ct domain.ContentType, locale, id string) (domain.Document, error)
}

// GroupBySearchConfig is the strategy-supplied portion of a GroupBy search.
// The adapter adds the mandatory price/availability refinements, identity, and
// client-metadata fields (mirrors GroupBySearchProvider.searchProductList).
type GroupBySearchConfig struct {
	Category  string
	Component string
	PageSize  int
}

// GroupBySearchResult is the decoded GroupBy search response.
type GroupBySearchResult struct {
	Records  []map[string]any
	Metadata map[string]any
}

// GroupBySearchPort runs a GroupBy product-list search.
type GroupBySearchPort interface {
	SearchProductList(ctx context.Context, cfg GroupBySearchConfig) (*GroupBySearchResult, error)
}

// GroupByRecommendationConfig is the strategy-supplied portion of a recommendation
// request (the adapter adds identity + fields). Mirrors GroupByRecomendationConfig.
type GroupByRecommendationConfig struct {
	Name      string
	EventType string
	ProductID string
	Limit     int
}

// GroupByRecommendationResult is the decoded recommendation response.
type GroupByRecommendationResult struct {
	Products []map[string]any
}

// GroupByRecommendationsPort fetches GroupBy recommendations.
type GroupByRecommendationsPort interface {
	GetRecommendations(ctx context.Context, cfg GroupByRecommendationConfig) (*GroupByRecommendationResult, error)
}

// JewelModelConfig mirrors the block's jewel_model_config.
type JewelModelConfig struct {
	Model            string
	RequiresDeviceID bool
	RequiresUserID   bool
}

// JewelPort fetches products from a Jewel model.
type JewelPort interface {
	GetProductsFromModel(ctx context.Context, cfg JewelModelConfig, min, max int) ([]map[string]any, error)
}

// SalesforcePort fetches a personalized action payload for the current user.
// Mirrors SalesforceProvider.getActionFromUser.
type SalesforcePort interface {
	GetActionFromUser(ctx context.Context, action string) (map[string]any, error)
}

// CartHeaderPort fetches the ATG cart header details (login state, favorite store,
// last cart item). Mirrors LiverpoolATGProvider.getCartHeaderDetails — returns the
// decoded `cartHeaderDetails` object.
type CartHeaderPort interface {
	GetCartHeaderDetails(ctx context.Context) (map[string]any, error)
}
