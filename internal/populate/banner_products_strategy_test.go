package populate

import (
	"context"
	"testing"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
)

type fakeMultiProduct struct {
	records []map[string]any
	lastFav string
}

func (f *fakeMultiProduct) GetMultiProductDetails(_ context.Context, _ []string, fav string) (*ports.MultiProductResult, error) {
	f.lastFav = fav
	return &ports.MultiProductResult{Records: f.records}, nil
}

func bannerBlock() Block {
	return Block{
		"_content_type_uid": "banner_products",
		"hotspots_manager": map[string]any{
			"desktop_image_group": []any{
				map[string]any{"productId": float64(1)},
				map[string]any{"productId": float64(2)},
			},
			"mobile_image_group": []any{
				map[string]any{"productId": float64(1)},
			},
		},
	}
}

func bannerCtx() context.Context {
	st := domain.NewRequestState()
	st.SetSelectedStore(&domain.SelectedStore{ID: 7, Name: "Centro"})
	return domain.WithRequestInfo(context.Background(), domain.RequestInfo{
		FeatureFlags: map[string]bool{"groupby": true}, State: st,
	})
}

func TestCreateStringSkuArray(t *testing.T) {
	skus := createStringSkuArray(bannerBlock())
	if len(skus) != 2 || skus[0] != "1" || skus[1] != "2" {
		t.Fatalf("expected unique [1 2], got %v", skus)
	}
}

func TestBannerProductsCombines(t *testing.T) {
	search := &fakeMultiProduct{records: []map[string]any{
		{"allMeta": map[string]any{"productId": "1", "title": "P1"}},
	}}
	rec := &fakeRec{result: &ports.GroupByRecommendationResult{Products: []map[string]any{
		{"primaryProductId": "2", "title": "Similar2"},
	}}}
	s := NewBannerProducts(search, rec)

	out, err := s.Populate(bannerCtx(), bannerBlock())
	if err != nil || out == nil {
		t.Fatalf("want populated, got out=%v err=%v", out, err)
	}
	if search.lastFav != "7" {
		t.Errorf("favorite store not forwarded: %q", search.lastFav)
	}

	groups := imageGroups(out)
	// productId 1 → matched search detail; productId 2 → similar-item detail; mobile 1 → matched.
	d1, _ := groups[0]["details"].(map[string]any)
	d2, _ := groups[1]["details"].(map[string]any)
	dm, _ := groups[2]["details"].(map[string]any)
	if d1 == nil || d1["productId"] != "1" {
		t.Errorf("group 1 details wrong: %v", groups[0]["details"])
	}
	if d2 == nil || d2["productId"] != "2" {
		t.Errorf("group 2 details (similar) wrong: %v", groups[1]["details"])
	}
	if dm == nil || dm["productId"] != "1" {
		t.Errorf("mobile group details wrong: %v", groups[2]["details"])
	}
}

func TestBannerProductsNoSkusDrops(t *testing.T) {
	s := NewBannerProducts(&fakeMultiProduct{}, nil)
	out, _ := s.Populate(bannerCtx(), Block{"_content_type_uid": "banner_products"})
	if out != nil {
		t.Error("no image groups → drop")
	}
}

func TestBannerProductsGroupbyOffDrops(t *testing.T) {
	s := NewBannerProducts(&fakeMultiProduct{}, nil)
	ctx := domain.WithRequestInfo(context.Background(), domain.RequestInfo{State: domain.NewRequestState()})
	out, _ := s.Populate(ctx, bannerBlock())
	if out != nil {
		t.Error("groupby flag off → drop")
	}
}

func TestBannerProductsMissingDetailsNull(t *testing.T) {
	// search returns nothing, no rec port → all groups get details=null.
	s := NewBannerProducts(&fakeMultiProduct{}, nil)
	out, err := s.Populate(bannerCtx(), bannerBlock())
	if err != nil || out == nil {
		t.Fatalf("want block, got %v %v", out, err)
	}
	for _, g := range imageGroups(out) {
		if g["details"] != nil {
			t.Errorf("expected null details, got %v", g["details"])
		}
	}
}
