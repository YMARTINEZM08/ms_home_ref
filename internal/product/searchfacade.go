package product

// FromSearchFacadeProduct maps a Search Facade product (allMeta-shaped) to a Dto.
// Faithful port of ProductDto.fromSearchFacadeProduct.
func FromSearchFacadeProduct(record map[string]any) Dto {
	allMeta, _ := record["allMeta"].(map[string]any)

	return Dto{
		ProductID: str(allMeta["productId"]),
		Name:      str(allMeta["title"]),
		PriceInfo: PriceInfo{
			MinimumListPrice:  num(allMeta["minimumListPrice"]),
			MaximumListPrice:  num(allMeta["maximumListPrice"]),
			MinimumPromoPrice: num(allMeta["minimumPromoPrice"]),
			MaximumPromoPrice: num(allMeta["maximumPromoPrice"]),
			OriginalPrice:     num(allMeta["maximumListPrice"]),
			PromoPrice:        num(allMeta["minimumPromoPrice"]),
			Price:             num(allMeta["minimumPromoPrice"]),
		},
		Categories: searchFacadeCategories(allMeta["categories"]),
		Images:     searchFacadeImages(allMeta["productImages"]),
		Brand:      firstString(allMeta["brands"]),
		Rating: Rating{
			Average: ratingInfo(allMeta, "ratingInfo_productAvgRating"),
			Count:   ratingInfo(allMeta, "ratingInfo_productRatingCount"),
		},
		Variants:            []any{},
		IsMarketplace:       str(allMeta["isMarketPlace"]) == "true",
		IsCollectionProduct: stringBoolean(str(allMeta["isCollectionProduct"])),
	}
}

// searchFacadeCategories splits allMeta.categories[0] on " > " then "#" → {id,label}.
func searchFacadeCategories(v any) []Category {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	first, ok := arr[0].(string)
	if !ok {
		return nil
	}
	return parseBreadcrumbs(first, " > ", "#")
}

func searchFacadeImages(v any) []Image {
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
		out = append(out, Image{Type: str(m["imageType"]), URL: str(m["imageUrl"])})
	}
	return out
}

func ratingInfo(allMeta map[string]any, key string) string {
	ri, ok := allMeta["ratingInfo"].(map[string]any)
	if !ok {
		return "0"
	}
	if s := str(ri[key]); s != "" {
		return s
	}
	return "0"
}
