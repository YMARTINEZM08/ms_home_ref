package populate

import (
	"context"
	"errors"
	"testing"

	"ms_home/internal/domain"
)

func loggedFlags(flags ...string) context.Context {
	m := map[string]bool{}
	for _, f := range flags {
		m[f] = true
	}
	return domain.WithRequestInfo(context.Background(), domain.RequestInfo{FeatureFlags: m, LoggedIn: true, ProfileID: "u1"})
}

func guestFlags(flags ...string) context.Context {
	m := map[string]bool{}
	for _, f := range flags {
		m[f] = true
	}
	return domain.WithRequestInfo(context.Background(), domain.RequestInfo{FeatureFlags: m})
}

func TestContainerGuest(t *testing.T) {
	s := ContainerGuest{}
	b := Block{"_content_type_uid": "container_guest"}

	if out, _ := s.Populate(guestFlags(), b); out != nil {
		t.Error("no personalization → drop")
	}
	if out, _ := s.Populate(guestFlags("personalization"), b); out == nil {
		t.Error("guest + personalization → keep")
	}
	if out, _ := s.Populate(loggedFlags("personalization"), b); out != nil {
		t.Error("logged in → drop")
	}
}

func TestContainerShortcuts(t *testing.T) {
	s := ContainerShortcuts{}
	b := Block{
		"_content_type_uid": "container_shortcuts",
		"shortcut_items": []any{
			map[string]any{"primary": map[string]any{"title": "P", "is_active": true, "primary_id": "p1"}},
			map[string]any{"custom": map[string]any{"title": "C", "is_active": true}},
		},
	}

	if out, _ := s.Populate(guestFlags(), b); out != nil {
		t.Fatal("anonymous → drop")
	}
	out, _ := s.Populate(loggedFlags(), b)
	if out == nil {
		t.Fatal("logged in → keep")
	}
	items := out["shortcut_items"].([]any)
	if len(items) != 2 {
		t.Fatalf("want 2 mapped shortcuts, got %d", len(items))
	}
	if items[0].(map[string]any)["type"] != "primary" || items[1].(map[string]any)["type"] != "custom" {
		t.Errorf("types wrong: %v", items)
	}
}

// --- Salesforce strategies ---

type fakeSF struct {
	data  map[string]any
	err   error
	calls []string
}

func (f *fakeSF) GetActionFromUser(_ context.Context, action string) (map[string]any, error) {
	f.calls = append(f.calls, action)
	return f.data, f.err
}

func TestProductsCards(t *testing.T) {
	sf := &fakeSF{data: map[string]any{"offers": 3}}
	s := NewProductsCards(sf)
	b := Block{"_content_type_uid": "products_cards"}

	if out, _ := s.Populate(loggedFlags(), b); out != nil {
		t.Error("no salesforce flag → drop")
	}
	out, err := s.Populate(loggedFlags("salesforce"), b)
	if err != nil || out == nil {
		t.Fatalf("want populated, got out=%v err=%v", out, err)
	}
	if out["incredibleOffersData"] == nil || sf.calls[len(sf.calls)-1] != sfIncredibleOffers {
		t.Errorf("incredibleOffersData/action wrong: %v / %v", out["incredibleOffersData"], sf.calls)
	}
}

func TestContainerGreeting(t *testing.T) {
	t.Run("non-birthday only_logged kept when logged", func(t *testing.T) {
		s := NewContainerGreeting(&fakeSF{})
		b := Block{"_content_type_uid": "container_greeting", "only_logged": true}
		if out, _ := s.Populate(loggedFlags("personalization"), b); out == nil {
			t.Error("should keep")
		}
	})

	t.Run("birthday without campaign drops", func(t *testing.T) {
		sf := &fakeSF{data: map[string]any{"campaignResponses": []any{
			map[string]any{"payload": map[string]any{"birthday": false}},
		}}}
		s := NewContainerGreeting(sf)
		b := Block{"_content_type_uid": "container_greeting", "is_birthday": true}
		if out, _ := s.Populate(loggedFlags("personalization", "salesforce"), b); out != nil {
			t.Error("no birthday campaign → drop")
		}
	})

	t.Run("birthday with campaign kept and salesforce attached", func(t *testing.T) {
		sf := &fakeSF{data: map[string]any{"campaignResponses": []any{
			map[string]any{"payload": map[string]any{"birthday": true}},
		}}}
		s := NewContainerGreeting(sf)
		b := Block{"_content_type_uid": "container_greeting", "is_birthday": true}
		out, _ := s.Populate(loggedFlags("personalization", "salesforce"), b)
		if out == nil || out["salesforce"] == nil {
			t.Errorf("should keep + attach salesforce: %v", out)
		}
	})

	t.Run("birthday without salesforce flag drops", func(t *testing.T) {
		s := NewContainerGreeting(&fakeSF{})
		b := Block{"_content_type_uid": "container_greeting", "is_birthday": true}
		if out, _ := s.Populate(loggedFlags("personalization"), b); out != nil {
			t.Error("birthday needs salesforce flag")
		}
	})
}

func TestProductListSalesforce(t *testing.T) {
	sf := &fakeSF{data: map[string]any{"campaignResponses": []any{
		map[string]any{"payload": map[string]any{
			"title": "Para ti",
			"products": []any{
				map[string]any{"id": "S1", "attributes": map[string]any{"name": map[string]any{"value": "A"}}},
				map[string]any{"id": "S2", "attributes": map[string]any{"name": map[string]any{"value": "B"}}},
			},
		}},
	}}}
	s := NewProductListSalesforce(sf)
	b := Block{
		"_content_type_uid": "products_list", "source_of_data": "salesforce",
		"salesforce_carousel": "CMSx", "min_of_products": "2", "max_of_products": "10",
	}
	out, err := s.Populate(loggedFlags("salesforce"), b)
	if err != nil || out == nil {
		t.Fatalf("want populated, got out=%v err=%v", out, err)
	}
	if products, _ := out["products"].([]any); len(products) != 2 {
		t.Fatalf("want 2 products, got %d", len(products))
	}
	if out["products_list_title"] != "Para ti" {
		t.Errorf("title not applied: %v", out["products_list_title"])
	}
}

func TestProductListSalesforceBelowMinDrops(t *testing.T) {
	sf := &fakeSF{data: map[string]any{"campaignResponses": []any{
		map[string]any{"payload": map[string]any{"products": []any{map[string]any{"id": "S1"}}}},
	}}}
	s := NewProductListSalesforce(sf)
	b := Block{
		"_content_type_uid": "products_list", "source_of_data": "salesforce",
		"salesforce_carousel": "CMSx", "min_of_products": "2", "max_of_products": "10",
	}
	if out, _ := s.Populate(loggedFlags("salesforce"), b); out != nil {
		t.Error("below minimum → drop")
	}
}

func TestSalesforceErrorDrops(t *testing.T) {
	sf := &fakeSF{err: errors.New("boom")}
	s := NewProductsCards(sf)
	if _, err := s.Populate(loggedFlags("salesforce"), Block{"_content_type_uid": "products_cards"}); err == nil {
		t.Error("want error")
	}
}

func TestDedupeGreetings(t *testing.T) {
	blocks := []any{
		map[string]any{"_content_type_uid": "container_greeting"},
		map[string]any{"_content_type_uid": "container"},
		map[string]any{"_content_type_uid": "container_greeting", "is_birthday": true},
		map[string]any{"_content_type_uid": "container_greeting"},
	}
	out := dedupeGreetings(blocks)
	greetings := 0
	for _, b := range out {
		if b.(map[string]any)["_content_type_uid"] == "container_greeting" {
			greetings++
		}
	}
	if greetings != 1 || len(out) != 2 {
		t.Fatalf("want 1 greeting + container (len 2), got %d greetings, len %d", greetings, len(out))
	}
}
