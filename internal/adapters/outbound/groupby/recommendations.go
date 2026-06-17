package groupby

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/pkg/httpclient"
)

// recordsFields mirrors groupByGetRecordsFields (liverpool.constant.ts).
var recordsFields = []string{
	"primaryProductId", "title", "images", "priceInfo.originalPrice", "priceInfo.price",
	"id", "isMarketPlace", "isHybridProduct", "isCollectionProduct", "productImages",
	"minimumListPrice", "maximumListPrice", "minimumPromoPrice", "maximumPromoPrice",
	"variants.prices", "variants.sellernames", "productType", "isVariant", "availability",
	"groupType", "relatedProducts", "brand", "categories", "hybridData", "miniPagosSkuPrice",
	"callRecommendCarousel", "attributes.isMarketPlace", "attributes.isHybridProduct",
	"attributes.isCollectionProduct", "attributes.minimumListPrice", "attributes.maximumListPrice",
	"attributes.minimumPromoPrice", "attributes.maximumPromoPrice", "attributes.isVariant",
	"attributes.skuId", "attributes.productType", "attributes.groupType", "attributes.relatedProducts",
	"brands", "attributes.hybridData", "attributes.miniPagosSkuPrice", "attributes.callRecommendCarousel",
	"attributes.PDPDescription", "attributes.brandId", "attributes.dynamicFacets_brandname",
	"attributes.salePrice", "attributes.promoPrice", "attributes.discountPercentage", "attributes.sellernames",
}

// RecommendationsAdapter implements ports.GroupByRecommendationsPort.
type RecommendationsAdapter struct {
	http *httpclient.Client
	url  string
}

// NewRecommendations builds the adapter. url is the full recommendations endpoint.
func NewRecommendations(client *httpclient.Client, url string) *RecommendationsAdapter {
	return &RecommendationsAdapter{http: client, url: strings.TrimRight(url, "/")}
}

// GetRecommendations POSTs identity + fields + config and decodes products.
func (a *RecommendationsAdapter) GetRecommendations(ctx context.Context, cfg ports.GroupByRecommendationConfig) (*ports.GroupByRecommendationResult, error) {
	ri := domain.RequestInfoFromContext(ctx)

	eventType := cfg.EventType
	if eventType == "" {
		eventType = "home-page-view" // GroupByRecomendationEventType.HOME_PAGE_VIEW default
	}

	body := map[string]any{
		"visitorId": ri.VisitorID,
		"loginId":   ri.ProfileID,
		"fields":    recordsFields,
		"name":      cfg.Name,
		"eventType": eventType,
		"limit":     cfg.Limit,
	}
	if cfg.ProductID != "" {
		body["productID"] = cfg.ProductID
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("groupby recommendations: marshal: %w", err)
	}
	headers := map[string]string{"Content-Type": "application/json"}
	if ri.CorrelationID != "" {
		headers["x-correlation-id"] = ri.CorrelationID
	}

	resp, err := a.http.Do(ctx, http.MethodPost, a.url, payload, headers)
	if err != nil {
		return nil, fmt.Errorf("groupby recommendations: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("groupby recommendations: unexpected status %d", resp.Status)
	}

	var parsed struct {
		Products []map[string]any `json:"products"`
	}
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("groupby recommendations: decode: %w", err)
	}
	return &ports.GroupByRecommendationResult{Products: parsed.Products}, nil
}
