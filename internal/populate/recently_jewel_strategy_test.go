package populate

import (
	"context"
	"errors"
	"testing"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
)

// --- recently_viewed ---

type fakeRec struct {
	result *ports.GroupByRecommendationResult
	err    error
	calls  int
}

func (f *fakeRec) GetRecommendations(context.Context, ports.GroupByRecommendationConfig) (*ports.GroupByRecommendationResult, error) {
	f.calls++
	return f.result, f.err
}

func lpRecords(n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = map[string]any{
			"primaryProductId":       "P",
			"attributes.sellernames": map[string]any{"text": []any{"liverpool"}},
		}
	}
	return out
}

func recentlyBlock() Block {
	return Block{
		"_content_type_uid": "products_list",
		"source_of_data":    "recently_viewed",
		"max_of_products":   "10",
		"min_of_products":   "2",
	}
}

func loggedCtx() context.Context {
	return ctxWith(domain.RequestInfo{
		FeatureFlags: map[string]bool{"groupby": true},
		Brand:        "LP",
		ProfileID:    "u1",
		VisitorID:    "v1",
		LoggedIn:     true,
	})
}

func TestRecentlyViewed(t *testing.T) {
	t.Run("anonymous (no profile/visitor) drops", func(t *testing.T) {
		fr := &fakeRec{result: &ports.GroupByRecommendationResult{Products: lpRecords(5)}}
		s := NewProductListRecentlyViewed(fr)
		out, _ := s.Populate(ctxWith(domain.RequestInfo{FeatureFlags: map[string]bool{"groupby": true}, Brand: "LP"}), recentlyBlock())
		if out != nil {
			t.Error("should drop without profileId/visitorId")
		}
		if fr.calls != 0 {
			t.Error("should not call recommendations")
		}
	})

	t.Run("populates and filters other-brand products", func(t *testing.T) {
		recs := lpRecords(3)
		recs = append(recs, map[string]any{ // suburbia product filtered out for LP
			"primaryProductId":       "SBP",
			"attributes.sellernames": map[string]any{"text": []any{"suburbia"}},
		})
		fr := &fakeRec{result: &ports.GroupByRecommendationResult{Products: recs}}
		s := NewProductListRecentlyViewed(fr)
		out, err := s.Populate(loggedCtx(), recentlyBlock())
		if err != nil || out == nil {
			t.Fatalf("want populated, got out=%v err=%v", out, err)
		}
		if products, _ := out["products"].([]any); len(products) != 3 {
			t.Fatalf("want 3 (suburbia filtered), got %d", len(products))
		}
	})

	t.Run("below minimum drops", func(t *testing.T) {
		fr := &fakeRec{result: &ports.GroupByRecommendationResult{Products: lpRecords(1)}}
		s := NewProductListRecentlyViewed(fr)
		out, _ := s.Populate(loggedCtx(), recentlyBlock())
		if out != nil {
			t.Error("should drop below minimum")
		}
	})
}

// --- jewel ---

type fakeJewel struct {
	products []map[string]any
	err      error
}

func (f *fakeJewel) GetProductsFromModel(context.Context, ports.JewelModelConfig, int, int) ([]map[string]any, error) {
	return f.products, f.err
}

func jewelBlock() Block {
	return Block{
		"_content_type_uid":  "products_list",
		"source_of_data":     "jewel",
		"min_of_products":    "2",
		"max_of_products":    "10",
		"jewel_model_config": map[string]any{"jewel_model": "m1"},
	}
}

func jewelProducts(n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = map[string]any{"standard_features": map[string]any{"event_id": "E"}}
	}
	return out
}

func jewelCtx(flags map[string]bool) context.Context {
	return ctxWith(domain.RequestInfo{FeatureFlags: flags})
}

func TestJewel(t *testing.T) {
	t.Run("requires jewel + personalization flags", func(t *testing.T) {
		s := NewProductListJewel(&fakeJewel{products: jewelProducts(5)})
		out, _ := s.Populate(jewelCtx(map[string]bool{"jewel": true}), jewelBlock()) // personalization off
		if out != nil {
			t.Error("should drop without personalization flag")
		}
	})

	t.Run("populates when flags on", func(t *testing.T) {
		s := NewProductListJewel(&fakeJewel{products: jewelProducts(5)})
		out, err := s.Populate(jewelCtx(map[string]bool{"jewel": true, "personalization": true}), jewelBlock())
		if err != nil || out == nil {
			t.Fatalf("want populated, got out=%v err=%v", out, err)
		}
		if products, _ := out["products"].([]any); len(products) != 5 {
			t.Fatalf("want 5 products, got %d", len(products))
		}
	})

	t.Run("below minimum drops", func(t *testing.T) {
		s := NewProductListJewel(&fakeJewel{products: jewelProducts(1)})
		out, _ := s.Populate(jewelCtx(map[string]bool{"jewel": true, "personalization": true}), jewelBlock())
		if out != nil {
			t.Error("should drop below minimum")
		}
	})

	t.Run("search error drops via error", func(t *testing.T) {
		s := NewProductListJewel(&fakeJewel{err: errors.New("boom")})
		_, err := s.Populate(jewelCtx(map[string]bool{"jewel": true, "personalization": true}), jewelBlock())
		if err == nil {
			t.Error("want error")
		}
	})
}
