package product

// FromSalesfroce maps a Salesforce record (nested {value} attributes) to a Dto.
// Faithful port of ProductDto.fromSalesfroce. Note: digital_bff leaves some price
// fields as strings (listPrice); here they are parsed to numbers (documented divergence).
func FromSalesfroce(record map[string]any) Dto {
	attrs, _ := record["attributes"].(map[string]any)
	dims, _ := record["dimensions"].(map[string]any)

	return Dto{
		ProductID: str(record["id"]),
		Name:      valueString(attrs, "name"),
		PriceInfo: PriceInfo{
			MaximumListPrice:  parseFloatOrZero(valueStringOr(attrs, "maxListPrice", "0")),
			MaximumPromoPrice: parseFloatOrZero(valueStringOr(attrs, "maxPromoPrice", "0")),
			OriginalPrice:     valueNumber(attrs, "price"),
			PromoPrice:        parseFloatOrZero(valueString(attrs, "listPrice")),
			Price:             parseFloatOrZero(valueString(attrs, "listPrice")),
		},
		Categories: salesforceCategories(attrs["categories"]),
		Images:     []Image{{Type: "largeImage", URL: valueString(attrs, "imageUrl")}},
		Brand:      firstString(dims["Brand"]),
		Rating: Rating{
			Average: valueStringOr(attrs, "rating", "0"),
			Count:   "0",
		},
		Variants:      []any{},
		IsMarketplace: false,
	}
}

func salesforceCategories(v any) []Category {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]Category, 0, len(arr))
	for _, id := range arr {
		out = append(out, Category{ID: str(id)})
	}
	return out
}

// valueString reads attrs[key].value as a string (Salesforce {value} shape).
func valueString(attrs map[string]any, key string) string {
	field, ok := attrs[key].(map[string]any)
	if !ok {
		return ""
	}
	return str(field["value"])
}

func valueStringOr(attrs map[string]any, key, fallback string) string {
	if s := valueString(attrs, key); s != "" {
		return s
	}
	return fallback
}

// valueNumber reads attrs[key].value as a float64.
func valueNumber(attrs map[string]any, key string) float64 {
	field, ok := attrs[key].(map[string]any)
	if !ok {
		return 0
	}
	return num(field["value"])
}
