package product

import "strings"

// FromGroupByRecomendation maps a GroupBy recommendation record (flat dotted keys)
// to a Dto. Faithful port of ProductDto.fromGroupByRecomendation.
func FromGroupByRecomendation(record map[string]any) Dto {
	return Dto{
		ProductID:           str(record["primaryProductId"]),
		Name:                str(record["title"]),
		IsCollectionProduct: stringBoolean(dottedText(record, "attributes.isCollectionProduct", 0)),
		PriceInfo: PriceInfo{
			MaximumListPrice:  parseIntOrZero(dottedTextOr(record, "attributes.maximumListPrice", 0, "0")),
			MinimumListPrice:  parseIntOrZero(dottedTextOr(record, "attributes.minimumListPrice", 0, "0")),
			MaximumPromoPrice: parseIntOrZero(dottedTextOr(record, "attributes.maximumPromoPrice", 0, "0")),
			MinimumPromoPrice: parseIntOrZero(dottedTextOr(record, "attributes.minimumPromoPrice", 0, "0")),
			OriginalPrice:     num(record["priceInfo.originalPrice"]),
			PromoPrice:        dottedNumber(record, "attributes.promoPrice", 0),
			Price:             num(record["priceInfo.originalPrice"]),
		},
		Categories:    recomendationCategories(record["categories"]),
		Images:        parseImages(record["images"]),
		Brand:         firstString(record["brands"]),
		Seller:        dottedText(record, "attributes.sellernames", 0),
		Rating:        Rating{Average: "0", Count: "0"},
		Variants:      []any{},
		IsMarketplace: dottedText(record, "attributes.isMarketPlace", 0) == "true",
	}
}

// recomendationCategories splits categories[0] on ">" into {label(trimmed)}.
func recomendationCategories(v any) []Category {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	first, ok := arr[0].(string)
	if !ok {
		return nil
	}
	parts := strings.Split(first, ">")
	out := make([]Category, 0, len(parts))
	for _, p := range parts {
		out = append(out, Category{Label: strings.TrimSpace(p)})
	}
	return out
}

// dottedText reads record[key].text[i] (GroupBy recommendation {text:[]} shape).
func dottedText(record map[string]any, key string, i int) string {
	field, ok := record[key].(map[string]any)
	if !ok {
		return ""
	}
	arr, ok := field["text"].([]any)
	if !ok || i >= len(arr) {
		return ""
	}
	return str(arr[i])
}

func dottedTextOr(record map[string]any, key string, i int, fallback string) string {
	if s := dottedText(record, key, i); s != "" {
		return s
	}
	return fallback
}

// dottedNumber reads record[key].numbers[i] as float64.
func dottedNumber(record map[string]any, key string, i int) float64 {
	field, ok := record[key].(map[string]any)
	if !ok {
		return 0
	}
	arr, ok := field["numbers"].([]any)
	if !ok || i >= len(arr) {
		return 0
	}
	return num(arr[i])
}
