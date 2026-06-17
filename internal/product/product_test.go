package product

import "testing"

func TestFromGroupBySearch(t *testing.T) {
	record := map[string]any{
		"allMeta": map[string]any{
			"title":  "Camisa",
			"brands": []any{"LP"},
			"priceInfo": map[string]any{
				"originalPrice": float64(500),
				"price":         float64(400),
			},
			"images": []any{
				map[string]any{"uri": "thumbnailImage##x##https://img/1.jpg"},
			},
			"attributes": map[string]any{
				"productId":           map[string]any{"text": []any{"P1"}},
				"minimumListPrice":    map[string]any{"text": []any{"500"}},
				"maximumListPrice":    map[string]any{"text": []any{"500"}},
				"minimumPromoPrice":   map[string]any{"text": []any{"400"}},
				"maximumPromoPrice":   map[string]any{"text": []any{"400"}},
				"categoryBreadCrumbs": map[string]any{"text": []any{"c1#Ropa>c2#Camisas"}},
				"sellernames":         map[string]any{"text": []any{"Liverpool"}},
				"productAvgRating":    map[string]any{"text": []any{"4.5"}},
				"productRatingCount":  map[string]any{"text": []any{"10"}},
				"isMarketPlace":       map[string]any{"text": []any{"false"}},
				"isCollectionProduct": map[string]any{"text": []any{"true"}},
			},
		},
	}

	dto := FromGroupBySearch(record)

	if dto.ProductID != "P1" || dto.Name != "Camisa" || dto.Brand != "LP" {
		t.Fatalf("basic fields wrong: %+v", dto)
	}
	if dto.PriceInfo.OriginalPrice != 500 || dto.PriceInfo.Price != 400 || dto.PriceInfo.MinimumListPrice != 500 {
		t.Errorf("priceInfo wrong: %+v", dto.PriceInfo)
	}
	if len(dto.Categories) != 2 || dto.Categories[0].ID != "c1" || dto.Categories[0].Label != "Ropa" {
		t.Errorf("categories wrong: %+v", dto.Categories)
	}
	if len(dto.Images) != 1 || dto.Images[0].Type != "thumbnailImage" || dto.Images[0].URL != "https://img/1.jpg" {
		t.Errorf("images wrong: %+v", dto.Images)
	}
	if dto.Seller != "Liverpool" || dto.Rating.Average != "4.5" || dto.Rating.Count != "10" {
		t.Errorf("seller/rating wrong: %+v", dto)
	}
	if dto.IsMarketplace || !dto.IsCollectionProduct {
		t.Errorf("flags wrong: marketplace=%v collection=%v", dto.IsMarketplace, dto.IsCollectionProduct)
	}
}

func TestFromGroupBySearchDefaults(t *testing.T) {
	dto := FromGroupBySearch(map[string]any{}) // empty record
	if dto.ProductID != "" || dto.Rating.Average != "0" || dto.Rating.Count != "0" {
		t.Errorf("defaults wrong: %+v", dto)
	}
	if dto.Variants == nil {
		t.Error("variants should be non-nil empty slice")
	}
}
