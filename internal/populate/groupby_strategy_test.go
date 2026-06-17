package populate

import (
	"context"
	"errors"
	"testing"

	"ms_home/internal/domain"
	"ms_home/internal/ports"
)

type fakeSearch struct {
	result *ports.GroupBySearchResult
	err    error
	calls  int
}

func (f *fakeSearch) SearchProductList(context.Context, ports.GroupBySearchConfig) (*ports.GroupBySearchResult, error) {
	f.calls++
	return f.result, f.err
}

func ctxWith(ri domain.RequestInfo) context.Context {
	return domain.WithRequestInfo(context.Background(), ri)
}

func recordsN(n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = map[string]any{"allMeta": map[string]any{"title": "p"}}
	}
	return out
}

func block() Block {
	return Block{
		"_content_type_uid": "products_list",
		"source_of_data":    "groupby",
		"products_data":     map[string]any{"category": "cat123"},
		"max_of_products":   "10",
		"min_of_products":   "4",
	}
}

func TestGroupByPopulate(t *testing.T) {
	t.Run("groupby flag off drops block", func(t *testing.T) {
		fs := &fakeSearch{}
		s := NewProductListGroupBy(fs)
		out, err := s.Populate(ctxWith(domain.RequestInfo{}), block())
		if err != nil || out != nil {
			t.Fatalf("want drop, got out=%v err=%v", out, err)
		}
		if fs.calls != 0 {
			t.Error("search should not be called when flag off")
		}
	})

	t.Run("missing category errors", func(t *testing.T) {
		s := NewProductListGroupBy(&fakeSearch{})
		b := block()
		b["products_data"] = map[string]any{}
		_, err := s.Populate(ctxWith(domain.RequestInfo{FeatureFlags: map[string]bool{"groupby": true}}), b)
		if !errors.Is(err, errNoCategory) {
			t.Fatalf("want errNoCategory, got %v", err)
		}
	})

	t.Run("below minimum drops", func(t *testing.T) {
		fs := &fakeSearch{result: &ports.GroupBySearchResult{Records: recordsN(2)}}
		s := NewProductListGroupBy(fs)
		out, err := s.Populate(ctxWith(domain.RequestInfo{FeatureFlags: map[string]bool{"groupby": true}}), block())
		if err != nil || out != nil {
			t.Fatalf("want drop, got out=%v err=%v", out, err)
		}
	})

	t.Run("populates products with index and productsListId", func(t *testing.T) {
		fs := &fakeSearch{result: &ports.GroupBySearchResult{Records: recordsN(5)}}
		s := NewProductListGroupBy(fs)
		out, err := s.Populate(ctxWith(domain.RequestInfo{FeatureFlags: map[string]bool{"groupby": true}}), block())
		if err != nil || out == nil {
			t.Fatalf("want populated block, got out=%v err=%v", out, err)
		}
		if out["productsListId"] != "cat123" {
			t.Errorf("productsListId = %v", out["productsListId"])
		}
		products, _ := out["products"].([]any)
		if len(products) != 5 {
			t.Fatalf("want 5 products, got %d", len(products))
		}
	})

	t.Run("audience logged drops anonymous", func(t *testing.T) {
		fs := &fakeSearch{result: &ports.GroupBySearchResult{Records: recordsN(5)}}
		s := NewProductListGroupBy(fs)
		b := block()
		b["audience_filter"] = "logged"
		out, _ := s.Populate(ctxWith(domain.RequestInfo{FeatureFlags: map[string]bool{"groupby": true}, LoggedIn: false}), b)
		if out != nil {
			t.Error("logged-only block should drop for anonymous user")
		}
	})

	t.Run("search error drops block", func(t *testing.T) {
		fs := &fakeSearch{err: errors.New("boom")}
		s := NewProductListGroupBy(fs)
		_, err := s.Populate(ctxWith(domain.RequestInfo{FeatureFlags: map[string]bool{"groupby": true}}), block())
		if err == nil {
			t.Error("want error from search")
		}
	})
}

func TestSlug(t *testing.T) {
	if got := slug("  Ofertas   Increíbles "); got != "ofertas_increíbles" {
		t.Errorf("slug = %q", got)
	}
}
