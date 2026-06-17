package application

// exposeFields is the full @Expose allowlist of CartHeaderDetailsDto (atg-user.ts).
// `me` projects the merge of the cart header and the JWT claims through this set
// (claims win, mirroring `{...cartHeaderDetails, ...decodeAccessToken}`).
var exposeFields = []string{
	"isLoggedIn", "cartCount", "lastPasswordReset", "favoriteStore",
	"firstName_full", "firstName", "lastName", "maternalName", "gender", "email",
	"enrollOnAccountCreation", "dateOfBirth", "enableLoyaltyCoupons", "enableLoyaltyForUser",
	"gtmUserId", "profileId", "displayLoyaltyWelcomeModal", "enableForAlloweduser",
	"userIsAllowedForLoyalty", "isSignUp", "requireProgressiveProfiling", "phoneHash",
	"prn", "isGuest",
}

// cartHeaderFields are the @Expose fields sourced (directly) from the cart header,
// copied verbatim. email and favoriteStore are remapped/sub-projected separately.
var cartHeaderFields = []string{
	"isLoggedIn", "cartCount", "firstName_full", "firstName", "lastName",
	"maternalName", "gender", "gtmUserId", "profileId",
	"enableLoyaltyCoupons", "enableLoyaltyForUser", "enrollOnAccountCreation",
	"displayLoyaltyWelcomeModal",
}

// projectMe builds `me` from the ATG cart header overlaid with JWT claims, mirroring
// UserService.getUserInformation → CartHeaderDetailsDto (excludeExtraneousValues).
// Remaps from MiddlewareService.getUserInfo: email = cart header `login`;
// favoriteStore → its @Expose subset {storeName, id}.
func projectMe(chd map[string]any, claims map[string]any) map[string]any {
	base := cartHeaderBase(chd)

	me := make(map[string]any, len(exposeFields))
	for _, f := range exposeFields {
		if v, ok := claims[f]; ok { // claims win
			me[f] = v
		} else if v, ok := base[f]; ok {
			me[f] = v
		}
	}
	return me
}

// cartHeaderBase projects the cart-header-sourced @Expose fields (with remaps).
func cartHeaderBase(chd map[string]any) map[string]any {
	base := make(map[string]any, len(cartHeaderFields)+2)
	for _, f := range cartHeaderFields {
		if v, ok := chd[f]; ok {
			base[f] = v
		}
	}
	if login, ok := chd["login"]; ok {
		base["email"] = login
	}
	if fav, ok := chd["favoriteStore"].(map[string]any); ok {
		store := map[string]any{}
		if v, ok := fav["storeName"]; ok {
			store["storeName"] = v
		}
		if v, ok := fav["id"]; ok {
			store["id"] = v
		}
		base["favoriteStore"] = store
	}
	return base
}
