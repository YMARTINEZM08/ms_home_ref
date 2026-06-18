package application

// exposeFields is the full @Expose allowlist of CartHeaderDetailsDto (atg-user.ts),
// used to overlay JWT claims onto `me` (claims win, mirroring
// `{...cartHeaderDetails, ...decodeAccessToken}` projected with excludeExtraneousValues).
var exposeFields = []string{
	"isLoggedIn", "cartCount", "lastPasswordReset", "favoriteStore",
	"firstName_full", "firstName", "lastName", "maternalName", "gender", "email",
	"enrollOnAccountCreation", "dateOfBirth", "enableLoyaltyCoupons", "enableLoyaltyForUser",
	"gtmUserId", "profileId", "displayLoyaltyWelcomeModal", "enableForAlloweduser",
	"userIsAllowedForLoyalty", "isSignUp", "requireProgressiveProfiling", "phoneHash",
	"prn", "isGuest",
}

// meStringDefaults are cart-header fields MiddlewareService.getUserInfo emits with a
// `?? ”` default — so `me` always carries them (as "" when the cart header is a guest).
var meStringDefaults = []string{
	"firstName_full", "firstName", "lastName", "maternalName", "gtmUserId", "profileId",
}

// meConditional are cart-header @Expose fields getUserInfo assigns without a default,
// so they appear only when present in the cart header.
var meConditional = []string{
	"gender", "enableLoyaltyCoupons", "enableLoyaltyForUser", "enrollOnAccountCreation",
	"displayLoyaltyWelcomeModal", "lastPasswordReset", "dateOfBirth",
	"enableForAlloweduser", "userIsAllowedForLoyalty",
}

// projectMe builds `me` from the ATG cart header overlaid with JWT claims, mirroring
// UserService.getUserInformation → CartHeaderDetailsDto (excludeExtraneousValues).
// getUserInfo remaps: email = cart header `login`; isGuest = !isLoggedIn; names/ids
// default to ""; favoriteStore → its @Expose subset {storeName, id}.
func projectMe(chd map[string]any, claims map[string]any) map[string]any {
	me := make(map[string]any, len(exposeFields))

	for _, f := range meStringDefaults {
		me[f] = strOrEmpty(chd[f])
	}
	me["email"] = strOrEmpty(chd["login"])

	loggedIn := boolOrFalse(chd["isLoggedIn"])
	if _, ok := chd["isLoggedIn"]; ok {
		me["isLoggedIn"] = loggedIn
	}
	me["isGuest"] = !loggedIn
	if v, ok := chd["cartCount"]; ok {
		me["cartCount"] = v
	}

	for _, f := range meConditional {
		if v, ok := chd[f]; ok {
			me[f] = v
		}
	}
	if fav, ok := chd["favoriteStore"].(map[string]any); ok {
		store := map[string]any{}
		if v, ok := fav["storeName"]; ok {
			store["storeName"] = v
		}
		if v, ok := fav["id"]; ok {
			store["id"] = v
		}
		me["favoriteStore"] = store
	}

	// Overlay JWT claims for any @Expose field present (claims win).
	for _, f := range exposeFields {
		if v, ok := claims[f]; ok {
			me[f] = v
		}
	}
	return me
}

func strOrEmpty(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func boolOrFalse(v any) bool {
	b, _ := v.(bool)
	return b
}
