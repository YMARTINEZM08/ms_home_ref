package populate

import (
	"context"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/internal/product"
)

// ProductListJewel populates `jewel` carousels from the Jewel model service.
// Faithful port of JewelProductRecommendationStrategy.
type ProductListJewel struct {
	jewel ports.JewelPort
}

// NewProductListJewel builds the strategy.
func NewProductListJewel(jewel ports.JewelPort) ProductListJewel {
	return ProductListJewel{jewel: jewel}
}

func (ProductListJewel) Supports(b Block) bool {
	return b["_content_type_uid"] == "products_list" && b["source_of_data"] == "jewel"
}

func (s ProductListJewel) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	cfg, ok := jewelModelConfig(b)
	if !ok || !jewelShouldPopulate(b, ri, cfg) {
		return nil, nil
	}

	min := atoiOr(str(b["min_of_products"]), 3)
	max := atoiOr(str(b["max_of_products"]), 15)

	products, err := s.jewel.GetProductsFromModel(ctx, cfg, min, max)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, nil
	}

	mapped := make([]any, 0, len(products))
	for i, p := range products {
		if dto, ok := product.FromJewel(p, i); ok {
			mapped = append(mapped, dto)
		}
	}
	if len(mapped) < min {
		return nil, nil
	}

	b["products"] = mapped
	return b, nil
}

func jewelShouldPopulate(b Block, ri domain.RequestInfo, cfg ports.JewelModelConfig) bool {
	if !ri.Flag("jewel") || !ri.Flag("personalization") {
		return false
	}
	switch str(b["audience_filter"]) {
	case "logged":
		if !ri.LoggedIn {
			return false
		}
	case "guest":
		if ri.LoggedIn {
			return false
		}
	}
	if cfg.RequiresUserID && ri.JewelUserID == "" {
		return false
	}
	if cfg.RequiresDeviceID && ri.JewelDeviceID == "" {
		return false
	}
	if ri.Source == domain.SourcePocket {
		return boolDefault(b["enable_on_apps"], true)
	}
	return boolDefault(b["enable_on_web"], true)
}

func jewelModelConfig(b Block) (ports.JewelModelConfig, bool) {
	mc, ok := b["jewel_model_config"].(map[string]any)
	if !ok {
		return ports.JewelModelConfig{}, false
	}
	return ports.JewelModelConfig{
		Model:            str(mc["jewel_model"]),
		RequiresDeviceID: boolDefault(mc["jewel_requires_device_id"], false),
		RequiresUserID:   boolDefault(mc["jewel_requires_user_id"], false),
	}, true
}

// atoiOr parses s, returning fallback when empty, unparseable, or zero
// (mirrors `parseInt(x) || fallback`).
func atoiOr(s string, fallback int) int {
	if n := atoi(s); n != 0 {
		return n
	}
	return fallback
}
