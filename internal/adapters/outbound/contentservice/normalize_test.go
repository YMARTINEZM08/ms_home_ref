package contentservice

import (
	"testing"
)

func TestNormalize_UnwrapsKeyWrapper(t *testing.T) {
	input := []any{
		map[string]any{
			"banner": map[string]any{"_content_type_uid": "banner", "uid": "b1"},
		},
	}
	got := normalize(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 block, got %d", len(got))
	}
	if got[0]["_content_type_uid"] != "banner" {
		t.Errorf("_content_type_uid = %v, want banner", got[0]["_content_type_uid"])
	}
}

func TestNormalize_InfersUIDFromKey(t *testing.T) {
	input := []any{
		map[string]any{
			"hero_banner": map[string]any{"uid": "h1"}, // no _content_type_uid
		},
	}
	got := normalize(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 block, got %d", len(got))
	}
	if got[0]["_content_type_uid"] != "hero_banner" {
		t.Errorf("_content_type_uid = %v, want hero_banner", got[0]["_content_type_uid"])
	}
}

func TestNormalize_AlreadyNormalisedPassesThrough(t *testing.T) {
	input := []any{
		map[string]any{"_content_type_uid": "products_list", "uid": "p1"},
	}
	got := normalize(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 block, got %d", len(got))
	}
	if got[0]["uid"] != "p1" {
		t.Errorf("uid = %v, want p1", got[0]["uid"])
	}
}

func TestNormalize_ContainerGridIsFlattened(t *testing.T) {
	input := []any{
		map[string]any{
			"_content_type_uid": "container_grid",
			"grid_items": []any{
				map[string]any{"banner": map[string]any{"_content_type_uid": "banner", "uid": "g1"}},
				map[string]any{"products_list": map[string]any{"_content_type_uid": "products_list", "uid": "g2"}},
			},
		},
	}
	got := normalize(input)
	if len(got) != 2 {
		t.Fatalf("container_grid should flatten to 2 blocks, got %d", len(got))
	}
	if got[0]["_content_type_uid"] != "banner" {
		t.Errorf("first block = %v, want banner", got[0]["_content_type_uid"])
	}
	if got[1]["_content_type_uid"] != "products_list" {
		t.Errorf("second block = %v, want products_list", got[1]["_content_type_uid"])
	}
}

func TestNormalize_TabsContainerPreservesContainer(t *testing.T) {
	input := []any{
		map[string]any{
			"_content_type_uid": "tabs_container",
			"tabs": []any{
				map[string]any{
					"title":   "Tab 1",
					"content": []any{map[string]any{"_content_type_uid": "banner", "uid": "t1"}},
				},
			},
		},
	}
	got := normalize(input)
	// tabs_container itself is kept as a single block (not flattened)
	if len(got) != 1 {
		t.Fatalf("tabs_container should produce 1 block, got %d", len(got))
	}
	if got[0]["_content_type_uid"] != "tabs_container" {
		t.Errorf("_content_type_uid = %v, want tabs_container", got[0]["_content_type_uid"])
	}
}

func TestNormalize_OrderingPreserved(t *testing.T) {
	input := []any{
		map[string]any{"_content_type_uid": "banner", "uid": "first"},
		map[string]any{"_content_type_uid": "carousel", "uid": "second"},
		map[string]any{"_content_type_uid": "products_list", "uid": "third"},
	}
	got := normalize(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(got))
	}
	uids := []string{"first", "second", "third"}
	for i, uid := range uids {
		if got[i]["uid"] != uid {
			t.Errorf("block[%d].uid = %v, want %v", i, got[i]["uid"], uid)
		}
	}
}

func TestNormalize_SkipsInvalidItems(t *testing.T) {
	input := []any{
		"not a map",
		nil,
		map[string]any{"_content_type_uid": "banner", "uid": "valid"},
	}
	got := normalize(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 valid block, got %d", len(got))
	}
}
