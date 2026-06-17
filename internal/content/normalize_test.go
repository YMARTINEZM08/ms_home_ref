package content

import (
	"reflect"
	"testing"
)

func TestNormalizeDoubleBlocks(t *testing.T) {
	t.Run("unwraps nested modular block", func(t *testing.T) {
		// { container: { container: [{ title: "x" }], extra: 1 } }
		in := []any{
			map[string]any{
				"container": map[string]any{
					"container": []any{map[string]any{"title": "x"}},
					"extra":     1,
				},
			},
		}
		got := NormalizeDoubleBlocks(in)
		want := []any{map[string]any{"title": "x", "extra": 1}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("fallback adds _content_type_uid and uid from _metadata", func(t *testing.T) {
		in := []any{
			map[string]any{
				"banner": map[string]any{
					"_metadata": map[string]any{"uid": "u1"},
					"title":     "hi",
				},
			},
		}
		got := NormalizeDoubleBlocks(in)[0].(map[string]any)
		if got["_content_type_uid"] != "banner" || got["uid"] != "u1" || got["title"] != "hi" {
			t.Errorf("unexpected fallback result: %v", got)
		}
	})

	t.Run("data uid overrides metadata uid", func(t *testing.T) {
		in := []any{
			map[string]any{
				"banner": map[string]any{
					"_metadata": map[string]any{"uid": "meta"},
					"uid":       "data",
				},
			},
		}
		got := NormalizeDoubleBlocks(in)[0].(map[string]any)
		if got["uid"] != "data" {
			t.Errorf("uid = %v, want data (data spread overrides metadata)", got["uid"])
		}
	})

	t.Run("flattens container_grid", func(t *testing.T) {
		in := []any{
			map[string]any{
				"container_grid": map[string]any{
					"container_grid": []any{map[string]any{
						"item_grid": []any{
							map[string]any{"card": map[string]any{"block": []any{map[string]any{"label": "a"}}, "size": 2}},
						},
					}},
				},
			},
		}
		got := NormalizeDoubleBlocks(in)[0].(map[string]any)
		grid := got["item_grid"].([]any)
		item := grid[0].(map[string]any)
		if item["item_grid_type"] != "card" || item["label"] != "a" || item["size"] != 2 {
			t.Errorf("unexpected grid item: %v", item)
		}
	})
}

func TestRenameAndDeleteKeys(t *testing.T) {
	tpl := map[string]any{"apps_products": []any{1}, "keep": true, "drop": "x"}
	RenameKeys(tpl, map[string]string{"apps_products": "products"})
	DeleteKeys(tpl, []string{"drop"})

	if _, ok := tpl["apps_products"]; ok {
		t.Error("apps_products should have been renamed away")
	}
	if _, ok := tpl["products"]; !ok {
		t.Error("products should exist after rename")
	}
	if _, ok := tpl["drop"]; ok {
		t.Error("drop should have been deleted")
	}
	if tpl["keep"] != true {
		t.Error("keep should remain")
	}
}
