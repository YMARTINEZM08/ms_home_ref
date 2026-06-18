package application

import "testing"

func TestProjectMe(t *testing.T) {
	chd := map[string]any{
		"isLoggedIn":          true,
		"cartCount":           float64(3),
		"firstName":           "Ana",
		"lastName":            "García",
		"gender":              "F",
		"login":               "ana@example.com",
		"email":               "notify@example.com", // notificationEmail; must NOT become me.email
		"profileId":           "p1",
		"favoriteStore":       map[string]any{"id": "42", "storeName": "Centro", "city": "CDMX"},
		"isBlockedDueToFraud": true, // not @Expose → dropped
	}
	me := projectMe(chd, nil)

	if me["email"] != "ana@example.com" {
		t.Errorf("email should be the login value, got %v", me["email"])
	}
	if _, ok := me["isBlockedDueToFraud"]; ok {
		t.Error("non-@Expose field should be dropped")
	}
	if me["firstName"] != "Ana" || me["profileId"] != "p1" {
		t.Errorf("expected projected fields, got %v", me)
	}
	fav, ok := me["favoriteStore"].(map[string]any)
	if !ok || fav["id"] != "42" || fav["storeName"] != "Centro" {
		t.Fatalf("favoriteStore projection wrong: %v", me["favoriteStore"])
	}
	if _, ok := fav["city"]; ok {
		t.Error("favoriteStore should only expose storeName + id")
	}
}

func TestProjectMeDefaultsAndGuest(t *testing.T) {
	// Guest cart header: getUserInfo-style defaults must still appear.
	me := projectMe(map[string]any{"isLoggedIn": false}, nil)
	if me["email"] != "" || me["firstName"] != "" || me["profileId"] != "" {
		t.Errorf("string fields should default to \"\": %v", me)
	}
	if me["isGuest"] != true || me["isLoggedIn"] != false {
		t.Errorf("isGuest should be !isLoggedIn: %v", me)
	}
	if _, ok := me["favoriteStore"]; ok {
		t.Error("favoriteStore omitted when absent")
	}
	if _, ok := me["gender"]; ok {
		t.Error("conditional field gender omitted when absent")
	}
}

func TestProjectMeMergesClaims(t *testing.T) {
	chd := map[string]any{"firstName": "Ana", "login": "ana@x.com"}
	claims := map[string]any{
		"lastPasswordReset": "2026-01-01", // token-only @Expose field
		"dateOfBirth":       float64(938736000000),
		"email":             "claims@x.com", // claims win over cart-header login
		"unrelated":         "drop-me",      // not @Expose
	}
	me := projectMe(chd, claims)

	if me["lastPasswordReset"] != "2026-01-01" || me["dateOfBirth"] != float64(938736000000) {
		t.Errorf("token-claim fields missing: %v", me)
	}
	if me["email"] != "claims@x.com" {
		t.Errorf("claims should win over cart-header login, got %v", me["email"])
	}
	if me["firstName"] != "Ana" {
		t.Errorf("cart-header field lost: %v", me["firstName"])
	}
	if _, ok := me["unrelated"]; ok {
		t.Error("non-@Expose claim should be dropped")
	}
}
