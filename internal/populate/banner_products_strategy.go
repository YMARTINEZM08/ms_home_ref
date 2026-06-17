package populate

import (
	"context"
	"strconv"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/internal/product"
)

const sfSimilarItems = "liverpool_similar_items"

// BannerProducts populates a banner_products block's hotspot image groups with
// product details. Faithful port of BannerProductsPopulateStrategy.
//
// Note: the favorite store comes from RequestState (resolved in HomeService.loadSession).
// AI metrics are not part of this strategy.
type BannerProducts struct {
	search ports.SearchFacadePort
	rec    ports.GroupByRecommendationsPort // optional (similar-items fallback)
}

// NewBannerProducts builds the strategy. rec may be nil (no similar-items fallback).
func NewBannerProducts(search ports.SearchFacadePort, rec ports.GroupByRecommendationsPort) BannerProducts {
	return BannerProducts{search: search, rec: rec}
}

func (BannerProducts) Supports(b Block) bool { return b["_content_type_uid"] == "banner_products" }

func (s BannerProducts) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("groupby") {
		return nil, nil
	}

	skus := createStringSkuArray(b)
	if len(skus) == 0 {
		return nil, nil // TS returns undefined
	}

	favStore := favoriteStoreID(ri)
	res, err := s.search.GetMultiProductDetails(ctx, skus, favStore)
	if err != nil {
		return nil, err
	}

	// Map matched products by productId, plus similar-items for the missing ones.
	details := make(map[string]map[string]any)
	matched := make(map[string]bool, len(res.Records))
	idx := 0
	for _, rec := range res.Records {
		dto := product.FromSearchFacadeProduct(rec)
		details[dto.ProductID] = withIndex(dto, idx)
		matched[dto.ProductID] = true
		idx++
	}
	for _, sku := range s.similarItems(ctx, skus, matched) {
		details[sku.baseID] = withIndex(sku.dto, idx)
		idx++
	}

	combineInformation(b, details)
	return b, nil
}

type similarItem struct {
	baseID string
	dto    product.Dto
}

// similarItems fetches one GroupBy similar item per sku missing from the search
// results (mirrors getRecomendations). Returns nothing when no recs port is wired.
func (s BannerProducts) similarItems(ctx context.Context, skus []string, matched map[string]bool) []similarItem {
	if s.rec == nil || len(skus) <= len(matched) {
		return nil
	}
	var out []similarItem
	for _, sku := range skus {
		if matched[sku] {
			continue
		}
		resp, err := s.rec.GetRecommendations(ctx, ports.GroupByRecommendationConfig{
			Name:      sfSimilarItems,
			EventType: "detail-page-view",
			Limit:     1,
			ProductID: sku,
		})
		if err != nil || len(resp.Products) == 0 {
			continue
		}
		out = append(out, similarItem{baseID: sku, dto: product.FromGroupByRecomendation(resp.Products[0])})
	}
	return out
}

// createStringSkuArray returns the unique product ids from the block's hotspot
// image groups (combineImageGroups + dedupe).
func createStringSkuArray(b Block) []string {
	seen := make(map[string]bool)
	var out []string
	for _, g := range imageGroups(b) {
		id := numKey(g["productId"])
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// combineInformation attaches `details` to each hotspot image group, keyed by
// baseId ?? productId (mirrors combineInformation).
func combineInformation(b Block, details map[string]map[string]any) {
	for _, g := range imageGroups(b) {
		key := numKey(g["baseId"])
		if key == "" {
			key = numKey(g["productId"])
		}
		if d, ok := details[key]; ok {
			g["details"] = d
		} else {
			g["details"] = nil
		}
	}
}

// imageGroups returns the desktop/tablet/mobile hotspot image-group maps (live refs).
func imageGroups(b Block) []map[string]any {
	hm, ok := b["hotspots_manager"].(map[string]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, key := range []string{"desktop_image_group", "tablet_image_group", "mobile_image_group"} {
		for _, item := range sliceAt(hm, key) {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

// favoriteStoreID returns the resolved favorite store id as a string ("" if none).
func favoriteStoreID(ri domain.RequestInfo) string {
	if ri.State == nil {
		return ""
	}
	if store := ri.State.SelectedStore(); store != nil {
		return strconv.Itoa(store.ID)
	}
	return ""
}

// withIndex serializes a product Dto to a map and sets its combined-list index.
func withIndex(dto product.Dto, index int) map[string]any {
	dto.Index = index
	return product.ToMap(dto)
}

// numKey converts a JSON number/string id to its string form ("123", not "123.0").
func numKey(v any) string {
	switch n := v.(type) {
	case string:
		return n
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	case int:
		return strconv.Itoa(n)
	default:
		return ""
	}
}
