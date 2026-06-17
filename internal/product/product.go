// Package product holds the pure ProductDto and its mappers, ported from
// digital_bff dto/product/product.dto.ts. No infrastructure.
package product

import (
	"encoding/json"
	"strconv"
	"strings"
)

// PriceInfo mirrors PriceInfoDto.
type PriceInfo struct {
	MinimumListPrice  float64 `json:"minimumListPrice"`
	MaximumListPrice  float64 `json:"maximumListPrice"`
	MinimumPromoPrice float64 `json:"minimumPromoPrice"`
	MaximumPromoPrice float64 `json:"maximumPromoPrice"`
	OriginalPrice     float64 `json:"originalPrice"`
	PromoPrice        float64 `json:"promoPrice"`
	Price             float64 `json:"price"`
}

// Category mirrors { id, label }.
type Category struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Image mirrors { type, url }.
type Image struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// Rating mirrors { average, count } (strings, as in digital_bff).
type Rating struct {
	Average string `json:"average"`
	Count   string `json:"count"`
}

// Dto is the ProductDto shape returned in populated carousels.
type Dto struct {
	Index               int        `json:"index"`
	ProductID           string     `json:"productId"`
	DiscountLabel       string     `json:"discountLabel,omitempty"` // jewel only
	Name                string     `json:"name"`
	PriceInfo           PriceInfo  `json:"priceInfo"`
	Categories          []Category `json:"categories"`
	Images              []Image    `json:"images"`
	Brand               string     `json:"brand"`
	Seller              string     `json:"seller"`
	SellerCode          string     `json:"sellerCode"`
	Rating              Rating     `json:"rating"`
	HasGift             bool       `json:"hasGift"`
	Variants            []any      `json:"variants"`
	IsMarketplace       bool       `json:"isMarketplace"`
	IsCollectionProduct bool       `json:"isCollectionProduct"`
}

// ToMap renders a Dto as a generic map (JSON field names), for embedding as a
// dynamic block field (e.g. hotspot image-group `details`).
func ToMap(d Dto) map[string]any {
	b, _ := json.Marshal(d)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// FromGroupBySearch maps a GroupBy search record (allMeta-shaped) to a Dto.
// Faithful port of ProductDto.fromGroupBySearch.
func FromGroupBySearch(record map[string]any) Dto {
	allMeta, _ := record["allMeta"].(map[string]any)
	attrs, _ := allMeta["attributes"].(map[string]any)
	priceInfo, _ := allMeta["priceInfo"].(map[string]any)

	return Dto{
		ProductID:           textAt(attrs, "productId", 0),
		IsCollectionProduct: stringBoolean(textAt(attrs, "isCollectionProduct", 0)),
		Name:                str(allMeta["title"]),
		PriceInfo: PriceInfo{
			MinimumListPrice:  parseFloatOrZero(textAt(attrs, "minimumListPrice", 0)),
			MaximumListPrice:  parseFloatOrZero(textAt(attrs, "maximumListPrice", 0)),
			MinimumPromoPrice: parseFloatOrZero(textAt(attrs, "minimumPromoPrice", 0)),
			MaximumPromoPrice: parseFloatOrZero(textAt(attrs, "maximumPromoPrice", 0)),
			OriginalPrice:     num(priceInfo["originalPrice"]),
			PromoPrice:        num(priceInfo["price"]),
			Price:             num(priceInfo["price"]),
		},
		Categories: parseBreadcrumbs(textAt(attrs, "categoryBreadCrumbs", 0), ">", "#"),
		Images:     parseImages(allMeta["images"]),
		Brand:      firstString(allMeta["brands"]),
		Seller:     textAt(attrs, "sellernames", 0),
		SellerCode: "",
		Rating: Rating{
			Average: defaultZero(textAt(attrs, "productAvgRating", 0)),
			Count:   defaultZero(textAt(attrs, "productRatingCount", 0)),
		},
		HasGift:       false,
		Variants:      []any{},
		IsMarketplace: textAt(attrs, "isMarketPlace", 0) == "true",
	}
}

// textAt reads attrs[key].text[i] as a string (GroupBy {text:[]} shape).
func textAt(attrs map[string]any, key string, i int) string {
	field, ok := attrs[key].(map[string]any)
	if !ok {
		return ""
	}
	arr, ok := field["text"].([]any)
	if !ok || i >= len(arr) {
		return ""
	}
	return str(arr[i])
}

func parseBreadcrumbs(s, outer, inner string) []Category {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, outer)
	out := make([]Category, 0, len(parts))
	for _, p := range parts {
		id, label := splitPair(p, inner)
		out = append(out, Category{ID: id, Label: label})
	}
	return out
}

// splitPair returns (first, second) of s split on sep; second is "" if absent.
func splitPair(s, sep string) (string, string) {
	kv := strings.SplitN(s, sep, 2)
	if len(kv) == 2 {
		return kv[0], kv[1]
	}
	return kv[0], ""
}

// parseImages maps allMeta.images[].uri "type##...##url" to {type,url}.
func parseImages(v any) []Image {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]Image, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		comps := strings.Split(str(m["uri"]), "##")
		img := Image{Type: comps[0]}
		img.URL = comps[len(comps)-1]
		out = append(out, img)
	}
	return out
}

func firstString(v any) string {
	if arr, ok := v.([]any); ok && len(arr) > 0 {
		return str(arr[0])
	}
	return ""
}

func defaultZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// num coerces a JSON number (float64) to float64; 0 otherwise.
func num(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func parseFloatOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// parseIntOrZero mirrors parseInt(s, 10) coerced to float64 (0 on failure).
func parseIntOrZero(s string) float64 {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return float64(n)
}

// formatNumber renders a JS-like number string ("10", "10.5"), matching `${n}`.
func formatNumber(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func stringBoolean(s string) bool { return s == "true" }
