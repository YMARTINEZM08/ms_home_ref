package application

import (
	"testing"

	"ms_home/internal/domain"
)

func TestShoppingAssistantShortcut(t *testing.T) {
	t.Run("nil without flag or login", func(t *testing.T) {
		if shoppingAssistantShortcut(domain.RequestInfo{LoggedIn: true}) != nil {
			t.Error("no shopping_assistant flag → nil")
		}
		if shoppingAssistantShortcut(domain.RequestInfo{FeatureFlags: map[string]bool{"shopping_assistant": true}}) != nil {
			t.Error("not logged in → nil")
		}
	})

	t.Run("web value when logged in", func(t *testing.T) {
		sc := shoppingAssistantShortcut(domain.RequestInfo{
			FeatureFlags: map[string]bool{"shopping_assistant": true}, LoggedIn: true, Source: domain.SourceWeb,
		})
		if sc == nil || sc["type"] != "action" || sc["value"] != shopAssistantWeb {
			t.Errorf("unexpected shortcut: %v", sc)
		}
	})

	t.Run("pocket value", func(t *testing.T) {
		sc := shoppingAssistantShortcut(domain.RequestInfo{
			FeatureFlags: map[string]bool{"shopping_assistant": true}, LoggedIn: true, Source: domain.SourcePocket,
		})
		if sc["value"] != shopAssistantApp {
			t.Errorf("pocket value = %v", sc["value"])
		}
	})
}

func riWithCart(chd map[string]any) domain.RequestInfo {
	st := domain.NewRequestState()
	if chd != nil {
		st.SetCartHeader(chd)
	}
	return domain.RequestInfo{FeatureFlags: map[string]bool{"personalization": true}, Source: domain.SourceWeb, State: st}
}

func TestContinueBuyingShortcut(t *testing.T) {
	t.Run("built from logged-in cart with last item", func(t *testing.T) {
		sc := continueBuyingShortcut(riWithCart(map[string]any{"isLoggedIn": true, "lastCartAddedItem": "https://img/x.jpg"}))
		if sc == nil || sc["type"] != "url" || sc["image"] != "https://img/x.jpg" || sc["value"] != continueBuyingWeb {
			t.Errorf("unexpected shortcut: %v", sc)
		}
	})

	t.Run("nil when not logged in", func(t *testing.T) {
		if continueBuyingShortcut(riWithCart(map[string]any{"isLoggedIn": false, "lastCartAddedItem": "x"})) != nil {
			t.Error("guest cart → nil")
		}
	})

	t.Run("nil when no last item", func(t *testing.T) {
		if continueBuyingShortcut(riWithCart(map[string]any{"isLoggedIn": true})) != nil {
			t.Error("no last item → nil")
		}
	})

	t.Run("nil when cart not loaded", func(t *testing.T) {
		if continueBuyingShortcut(riWithCart(nil)) != nil {
			t.Error("no cart header → nil")
		}
	})
}
