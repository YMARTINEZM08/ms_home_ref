package product

import "testing"

func TestFromSearchFacadeProduct(t *testing.T) {
	record := map[string]any{
		"allMeta": map[string]any{
			"productId":           "P1",
			"title":               "Licuadora",
			"minimumListPrice":    float64(1000),
			"maximumListPrice":    float64(1200),
			"minimumPromoPrice":   float64(800),
			"maximumPromoPrice":   float64(900),
			"brands":              []any{"LP"},
			"categories":          []any{"c1#Hogar > c2#Cocina"},
			"productImages":       []any{map[string]any{"imageType": "large", "imageUrl": "https://img/p1.jpg"}},
			"ratingInfo":          map[string]any{"ratingInfo_productAvgRating": "4.8", "ratingInfo_productRatingCount": "20"},
			"isMarketPlace":       "true",
			"isCollectionProduct": "false",
		},
	}
	dto := FromSearchFacadeProduct(record)

	if dto.ProductID != "P1" || dto.Name != "Licuadora" || dto.Brand != "LP" {
		t.Fatalf("basic fields wrong: %+v", dto)
	}
	// originalPrice = maximumListPrice; promoPrice/price = minimumPromoPrice.
	if dto.PriceInfo.OriginalPrice != 1200 || dto.PriceInfo.PromoPrice != 800 || dto.PriceInfo.Price != 800 {
		t.Errorf("priceInfo wrong: %+v", dto.PriceInfo)
	}
	if len(dto.Categories) != 2 || dto.Categories[0].ID != "c1" || dto.Categories[0].Label != "Hogar" {
		t.Errorf("categories wrong (expect ' > ' split): %+v", dto.Categories)
	}
	if dto.Images[0].Type != "large" || dto.Images[0].URL != "https://img/p1.jpg" {
		t.Errorf("images wrong: %+v", dto.Images)
	}
	if dto.Rating.Average != "4.8" || dto.Rating.Count != "20" {
		t.Errorf("rating wrong: %+v", dto.Rating)
	}
	if !dto.IsMarketplace || dto.IsCollectionProduct {
		t.Errorf("flags wrong: marketplace=%v collection=%v", dto.IsMarketplace, dto.IsCollectionProduct)
	}
}
