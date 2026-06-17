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
	me := projectMe(chd)

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

func TestProjectMeOmitsAbsent(t *testing.T) {
	me := projectMe(map[string]any{"isLoggedIn": false})
	if _, ok := me["email"]; ok {
		t.Error("email omitted when login absent")
	}
	if _, ok := me["favoriteStore"]; ok {
		t.Error("favoriteStore omitted when absent")
	}
}
