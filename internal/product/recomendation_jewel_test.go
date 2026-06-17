package product

import "testing"

func TestFromGroupByRecomendation(t *testing.T) {
	record := map[string]any{
		"primaryProductId":            "P9",
		"title":                       "Tenis",
		"brands":                      []any{"LP"},
		"categories":                  []any{"Ropa > Calzado"},
		"priceInfo.originalPrice":     float64(700),
		"images":                      []any{map[string]any{"uri": "largeImage##x##https://img/9.jpg"}},
		"attributes.promoPrice":       map[string]any{"numbers": []any{float64(600)}},
		"attributes.minimumListPrice": map[string]any{"text": []any{"700"}},
		"attributes.sellernames":      map[string]any{"text": []any{"liverpool"}},
		"attributes.isMarketPlace":    map[string]any{"text": []any{"false"}},
	}
	dto := FromGroupByRecomendation(record)

	if dto.ProductID != "P9" || dto.Name != "Tenis" || dto.Brand != "LP" {
		t.Fatalf("basic fields wrong: %+v", dto)
	}
	if dto.PriceInfo.OriginalPrice != 700 || dto.PriceInfo.PromoPrice != 600 || dto.PriceInfo.MinimumListPrice != 700 {
		t.Errorf("priceInfo wrong: %+v", dto.PriceInfo)
	}
	if len(dto.Categories) != 2 || dto.Categories[0].Label != "Ropa" || dto.Categories[1].Label != "Calzado" {
		t.Errorf("categories wrong: %+v", dto.Categories)
	}
	if dto.Seller != "liverpool" || dto.IsMarketplace {
		t.Errorf("seller/marketplace wrong: %+v", dto)
	}
}

func TestFromJewel(t *testing.T) {
	t.Run("maps via offer_id and discount", func(t *testing.T) {
		record := map[string]any{
			"title": "Reloj",
			"standard_features": map[string]any{
				"offer_id":        "OF123_x",
				"brand":           "LP",
				"discount_amount": float64(15),
				"min_list_price":  float64(1000),
				"sale_price":      float64(850),
				"price":           float64(1000),
				"image_url_src":   "https://img/r.jpg",
				"seller_names":    "Liverpool|Otro",
				"rating":          float64(4),
				"rating_count":    float64(12),
				"item_type":       "c1#Relojes",
			},
		}
		dto, ok := FromJewel(record, 2)
		if !ok {
			t.Fatal("expected ok")
		}
		if dto.ProductID != "OF123" || dto.Index != 2 {
			t.Errorf("productId/index wrong: %+v", dto)
		}
		if dto.DiscountLabel != "-15%" {
			t.Errorf("discountLabel = %q", dto.DiscountLabel)
		}
		if dto.PriceInfo.PromoPrice != 850 || dto.Brand != "LP" {
			t.Errorf("price/brand wrong: %+v", dto)
		}
		if dto.Seller != "Liverpool" || dto.Rating.Average != "4" || dto.Rating.Count != "12" {
			t.Errorf("seller/rating wrong: %+v", dto)
		}
		if dto.Images[0].Type != "thumbnailImage" || dto.Images[0].URL != "https://img/r.jpg" {
			t.Errorf("images wrong: %+v", dto.Images)
		}
	})

	t.Run("falls back to event_id", func(t *testing.T) {
		dto, ok := FromJewel(map[string]any{
			"standard_features": map[string]any{"event_id": "EV1"},
		}, 0)
		if !ok || dto.ProductID != "EV1" {
			t.Errorf("event_id fallback failed: ok=%v dto=%+v", ok, dto)
		}
		if dto.DiscountLabel != "" {
			t.Errorf("discountLabel should be empty, got %q", dto.DiscountLabel)
		}
	})

	t.Run("no id returns not ok", func(t *testing.T) {
		if _, ok := FromJewel(map[string]any{"standard_features": map[string]any{}}, 0); ok {
			t.Error("expected ok=false when no offer_id/event_id")
		}
	})
}
