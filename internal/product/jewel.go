package product

import "strings"

// FromJewel maps a Jewel product to a Dto. Faithful port of ProductDto.fromJewel.
// Returns ok=false when no valid product id can be derived (TS returns null → skipped).
func FromJewel(record map[string]any, index int) (Dto, bool) {
	sf, _ := record["standard_features"].(map[string]any)

	productID, ok := productIDFromJewel(sf)
	if !ok {
		return Dto{}, false
	}

	dto := Dto{
		Index:     index,
		ProductID: productID,
		Name:      str(record["title"]),
		PriceInfo: PriceInfo{
			MinimumListPrice:  num(sf["min_list_price"]),
			MaximumListPrice:  num(sf["max_list_price"]),
			MinimumPromoPrice: num(sf["min_promo_price"]),
			MaximumPromoPrice: num(sf["max_promo_price"]),
			OriginalPrice:     num(sf["price"]),
			PromoPrice:        num(sf["sale_price"]),
			Price:             num(sf["sale_price"]),
		},
		Categories: parseBreadcrumbs(str(sf["item_type"]), ">", "#"),
		Images:     []Image{{Type: "thumbnailImage", URL: str(sf["image_url_src"])}},
		Brand:      str(sf["brand"]),
		Seller:     firstSplit(str(sf["seller_names"]), "|"),
		SellerCode: "",
		Rating: Rating{
			Average: ratingString(sf["rating"]),
			Count:   ratingString(sf["rating_count"]),
		},
		Variants:      sliceOrEmpty(sf["item_variants"]),
		IsMarketplace: false,
	}

	if d := num(sf["discount_amount"]); d > 0 {
		dto.DiscountLabel = "-" + formatNumber(d) + "%"
	}
	return dto, true
}

// productIDFromJewel mirrors: offer_id split('_')[0], else event_id, else null.
func productIDFromJewel(sf map[string]any) (string, bool) {
	if offer, ok := sf["offer_id"].(string); ok {
		return strings.SplitN(offer, "_", 2)[0], true
	}
	if ev := str(sf["event_id"]); ev != "" {
		return ev, true
	}
	return "", false
}

func firstSplit(s, sep string) string {
	if s == "" {
		return ""
	}
	return strings.SplitN(s, sep, 2)[0]
}

// ratingString mirrors `${value ?? '0'}` for a numeric rating.
func ratingString(v any) string {
	if f, ok := v.(float64); ok {
		return formatNumber(f)
	}
	return "0"
}

func sliceOrEmpty(v any) []any {
	if arr, ok := v.([]any); ok {
		return arr
	}
	return []any{}
}
