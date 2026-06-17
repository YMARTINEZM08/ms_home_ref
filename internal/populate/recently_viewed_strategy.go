package populate

import (
	"context"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/internal/product"
)

// groupByBrand maps a brand code to its GroupBy seller slug (brand.constant.ts).
var groupByBrand = map[string]string{
	"LP": "liverpool", "SB": "suburbia", "DCK": "dockers", "WE": "westelm",
	"WS": "williamssonoma", "PB": "potterybarn", "PBK": "potterybarnkids",
	"TRU": "toysrus", "BRU": "babiesrus", "GAP": "gap", "BR": "bananarepublic",
	"DPS": "dupuis", "FB": "fabletics",
}

// ProductListRecentlyViewed populates `recently_viewed` carousels from GroupBy
// recommendations. Faithful port of ProductListRecentlyViewedPopulateStrategy.
//
// TODO(phase-2): AI metrics (pushMetric) deferred.
type ProductListRecentlyViewed struct {
	rec ports.GroupByRecommendationsPort
}

// NewProductListRecentlyViewed builds the strategy.
func NewProductListRecentlyViewed(rec ports.GroupByRecommendationsPort) ProductListRecentlyViewed {
	return ProductListRecentlyViewed{rec: rec}
}

func (ProductListRecentlyViewed) Supports(b Block) bool {
	return b["_content_type_uid"] == "products_list" && b["source_of_data"] == "recently_viewed"
}

func (s ProductListRecentlyViewed) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("groupby") {
		return nil, nil
	}
	if !recentlyShouldPopulate(b, ri) {
		return nil, nil
	}

	res, err := s.rec.GetRecommendations(ctx, ports.GroupByRecommendationConfig{
		Name:  "liverpool_recently_viewed",
		Limit: atoi(str(b["max_of_products"])),
	})
	if err != nil {
		return nil, err
	}

	filtered := filterByBrand(res.Products, ri.Brand)
	if len(filtered) < atoi(str(b["min_of_products"])) {
		return nil, nil
	}

	b["productsListId"] = recordsModelID(b)
	products := make([]any, 0, len(filtered))
	for i, rec := range filtered {
		dto := product.FromGroupByRecomendation(rec)
		dto.Index = i
		products = append(products, dto)
	}
	b["products"] = products
	return b, nil
}

// filterByBrand drops collection products and products from other brands
// (mirrors the strategy's seller-name filter).
func filterByBrand(records []map[string]any, brand string) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		if coll := dottedTextAll(rec, "attributes.isCollectionProduct"); len(coll) > 0 && coll[0] == "true" {
			continue
		}
		sellers := dottedTextAll(rec, "attributes.sellernames")
		switch brand {
		case "LP":
			if contains(sellers, "suburbia") { // GroupByBrand.SB
				continue
			}
		case "SB":
			if !contains(sellers, "suburbia") {
				continue
			}
		default:
			if !contains(sellers, groupByBrand[brand]) {
				continue
			}
		}
		out = append(out, rec)
	}
	return out
}

func recentlyShouldPopulate(b Block, ri domain.RequestInfo) bool {
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
	if ri.ProfileID == "" || ri.VisitorID == "" {
		return false
	}
	if ri.Source == domain.SourcePocket {
		return boolDefault(b["enable_on_apps"], true)
	}
	return boolDefault(b["enable_on_web"], true)
}

// recordsModelID returns block.records.modelId ?? "".
func recordsModelID(b Block) string {
	if records, ok := b["records"].(map[string]any); ok {
		return str(records["modelId"])
	}
	return ""
}

// dottedTextAll reads record[key].text as a []string (GroupBy {text:[]} shape).
func dottedTextAll(record map[string]any, key string) []string {
	field, ok := record[key].(map[string]any)
	if !ok {
		return nil
	}
	arr, ok := field["text"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
