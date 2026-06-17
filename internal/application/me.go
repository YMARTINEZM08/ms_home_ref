package application

// meFields are the cart-header-sourced @Expose fields of CartHeaderDetailsDto
// (atg-user.ts). Token-claim @Expose fields (lastPasswordReset, dateOfBirth, …)
// are gateway-forwarded and currently omitted — see docs/todos.md (auth boundary).
var meFields = []string{
	"isLoggedIn", "cartCount", "firstName_full", "firstName", "lastName",
	"maternalName", "gender", "gtmUserId", "profileId",
	"enableLoyaltyCoupons", "enableLoyaltyForUser", "enrollOnAccountCreation",
	"displayLoyaltyWelcomeModal",
}

// projectMe builds the `me` object from the ATG cart header, mirroring
// UserService.getUserInformation → CartHeaderDetailsDto (excludeExtraneousValues).
// Notable remaps from MiddlewareService.getUserInfo: email = cart header `login`;
// favoriteStore is projected to its @Expose subset {storeName, id}.
func projectMe(chd map[string]any) map[string]any {
	me := make(map[string]any, len(meFields)+2)
	for _, k := range meFields {
		if v, ok := chd[k]; ok {
			me[k] = v
		}
	}
	if login, ok := chd["login"]; ok {
		me["email"] = login
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
	return me
}
