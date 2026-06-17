package populate

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
	"ms_home/internal/product"
)

// errNoCategory mirrors ContentCategoryNotInBlockError (block is dropped).
var errNoCategory = errors.New("populate: products_list missing products_data.category")

// ProductListGroupBy populates `products_list` carousels sourced from GroupBy.
// Faithful port of ProductListGroupByPopulateStrategy.
//
// TODO(phase-1b): blacklist filtering currently no-ops (empty restricted set,
// matching digital_bff's "continue with empty cache" fallback) — wire the Search
// Facade restricted-products endpoint. AI metrics (pushMetric) deferred to Phase 2.
type ProductListGroupBy struct {
	search ports.GroupBySearchPort
}

// NewProductListGroupBy builds the strategy.
func NewProductListGroupBy(search ports.GroupBySearchPort) ProductListGroupBy {
	return ProductListGroupBy{search: search}
}

func (ProductListGroupBy) Supports(b Block) bool {
	return b["_content_type_uid"] == "products_list" && b["source_of_data"] == "groupby"
}

func (s ProductListGroupBy) Populate(ctx context.Context, b Block) (Block, error) {
	ri := domain.RequestInfoFromContext(ctx)
	if !ri.Flag("groupby") {
		return nil, nil // drop
	}

	productsData, _ := b["products_data"].(map[string]any)
	category := str(productsData["category"])
	if category == "" {
		return nil, errNoCategory
	}
	if !shouldPopulate(b, ri) {
		return nil, nil // drop (audience / surface toggle)
	}

	res, err := s.search.SearchProductList(ctx, ports.GroupBySearchConfig{
		Category:  category,
		PageSize:  atoi(str(b["max_of_products"])),
		Component: slug(strOr(str(b["products_list_title"]), "unknown_component")),
	})
	if err != nil {
		return nil, err
	}
	if len(res.Records) < atoi(str(b["min_of_products"])) {
		return nil, nil // drop (below minimum)
	}

	b["productsListId"] = category
	products := make([]any, 0, len(res.Records))
	for i, rec := range res.Records {
		dto := product.FromGroupBySearch(rec)
		dto.Index = i
		products = append(products, dto)
	}
	b["products"] = products
	return b, nil
}

// shouldPopulate ports the strategy's audience + surface gating.
func shouldPopulate(b Block, ri domain.RequestInfo) bool {
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
	if ri.Source == domain.SourcePocket {
		return boolDefault(b["enable_on_apps"], true)
	}
	return boolDefault(b["enable_on_web"], true)
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func strOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func boolDefault(v any, fallback bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return fallback
}

// slug lowercases, trims, and collapses whitespace runs to "_" (\s+ -> _).
func slug(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), "_")
}
