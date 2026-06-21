package home_test

import (
	"testing"

	"github.com/YMARTINEZM08/ms_home_ref/internal/domain/home"
)

func TestIsDynamic(t *testing.T) {
	tests := []struct {
		blockType home.BlockType
		want      bool
	}{
		{home.BlockTypeProductList, true},
		{home.BlockTypeBannerProducts, true},
		{home.BlockTypeGreeting, true},
		{home.BlockTypeGuestContainer, true},
		{home.BlockTypeShortcuts, true},
		{home.BlockTypeRecommendations, true},
		{home.BlockTypeProductCards, true},
		// Static types
		{home.BlockTypeBanner, false},
		{home.BlockTypeHeroBanner, false},
		{home.BlockTypeCarousel, false},
		{home.BlockTypePromoBar, false},
		{home.BlockTypeCountdown, false},
		{"unknown_type", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.blockType), func(t *testing.T) {
			if got := home.IsDynamic(tt.blockType); got != tt.want {
				t.Errorf("IsDynamic(%q) = %v, want %v", tt.blockType, got, tt.want)
			}
		})
	}
}

func TestIsAllowedResolveType(t *testing.T) {
	allowed := []string{
		"product_list", "banner_products", "container_greeting",
		"container_guest", "container_shortcuts",
		"recommendation_product_list", "products_cards",
	}
	for _, raw := range allowed {
		t.Run("allowed/"+raw, func(t *testing.T) {
			_, ok := home.IsAllowedResolveType(raw)
			if !ok {
				t.Errorf("IsAllowedResolveType(%q) should be allowed", raw)
			}
		})
	}

	rejected := []string{
		"", "../etc/passwd", "page", "banner", "unknown",
		"product_list; DROP TABLE blocks;", "<script>",
	}
	for _, raw := range rejected {
		t.Run("rejected/"+raw, func(t *testing.T) {
			_, ok := home.IsAllowedResolveType(raw)
			if ok {
				t.Errorf("IsAllowedResolveType(%q) should be rejected", raw)
			}
		})
	}
}
