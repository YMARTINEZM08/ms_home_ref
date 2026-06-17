package application

import "ms_home/internal/domain"

// Shortcut values (shortcuts.constant.ts).
const (
	shopAssistantImage = "https://assetsgcpqa2.liverpool.com.mx/assets/personalShopper/chatNH.svg"
	shopAssistantLabel = "Asistente de compras"
	shopAssistantWeb   = "SHOPPING_ASSISTANT"
	shopAssistantApp   = "asistente-compras-shortcut"

	continueBuyingLabel = "Seguir comprando"
	continueBuyingWeb   = "/tienda/cart"
	continueBuyingApp   = "bag-shortcut"
)

// continueBuyingShortcut builds the continue-buying shortcut from the memoized ATG
// cart header (port of ShortcutsService.addContinueBuyingShortcut, non-decommission).
// Returns nil unless personalization is on, the user is logged in, and the cart has a
// last-added item image.
func continueBuyingShortcut(ri domain.RequestInfo) map[string]any {
	if !ri.Flag("personalization") || ri.State == nil {
		return nil
	}
	chd, ok := ri.State.CartHeader()
	if !ok {
		return nil
	}
	if loggedIn, _ := chd["isLoggedIn"].(bool); !loggedIn {
		return nil
	}
	image, _ := chd["lastCartAddedItem"].(string)
	if image == "" {
		return nil
	}
	value := continueBuyingWeb
	if ri.Source == domain.SourcePocket {
		value = continueBuyingApp
	}
	return map[string]any{
		"type":  "url",
		"label": continueBuyingLabel,
		"image": image,
		"value": value,
	}
}

// shoppingAssistantShortcut builds the shopping-assistant shortcut (self-contained;
// port of ShortcutsService.addShoppingAssistantShortcut). Returns nil when the
// shopping_assistant flag is off or the user is not logged in.
func shoppingAssistantShortcut(ri domain.RequestInfo) map[string]any {
	if !ri.Flag("shopping_assistant") || !ri.LoggedIn {
		return nil
	}
	value := shopAssistantWeb
	if ri.Source == domain.SourcePocket {
		value = shopAssistantApp
	}
	return map[string]any{
		"type":  "action",
		"label": shopAssistantLabel,
		"image": shopAssistantImage,
		"value": value,
	}
}
