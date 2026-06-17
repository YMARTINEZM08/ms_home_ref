package product

import "testing"

func TestFromSalesfroce(t *testing.T) {
	record := map[string]any{
		"id": "SF1",
		"attributes": map[string]any{
			"name":          map[string]any{"value": "Bolsa"},
			"price":         map[string]any{"value": float64(900)},
			"listPrice":     map[string]any{"value": "750"},
			"maxListPrice":  map[string]any{"value": "900"},
			"maxPromoPrice": map[string]any{"value": "750"},
			"imageUrl":      map[string]any{"value": "https://img/sf.jpg"},
			"rating":        map[string]any{"value": "4.2"},
			"categories":    []any{"cat1", "cat2"},
		},
		"dimensions": map[string]any{"Brand": []any{"LP"}},
	}
	dto := FromSalesfroce(record)

	if dto.ProductID != "SF1" || dto.Name != "Bolsa" || dto.Brand != "LP" {
		t.Fatalf("basic fields wrong: %+v", dto)
	}
	if dto.PriceInfo.OriginalPrice != 900 || dto.PriceInfo.PromoPrice != 750 || dto.PriceInfo.Price != 750 {
		t.Errorf("priceInfo wrong: %+v", dto.PriceInfo)
	}
	if dto.PriceInfo.MaximumListPrice != 900 || dto.PriceInfo.MaximumPromoPrice != 750 {
		t.Errorf("max prices wrong: %+v", dto.PriceInfo)
	}
	if len(dto.Categories) != 2 || dto.Categories[0].ID != "cat1" {
		t.Errorf("categories wrong: %+v", dto.Categories)
	}
	if dto.Images[0].Type != "largeImage" || dto.Images[0].URL != "https://img/sf.jpg" {
		t.Errorf("images wrong: %+v", dto.Images)
	}
	if dto.Rating.Average != "4.2" || dto.Rating.Count != "0" {
		t.Errorf("rating wrong: %+v", dto.Rating)
	}
}
