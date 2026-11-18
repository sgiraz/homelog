package handlers

import (
	"net/http"
	"testing"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// A deleted member drops out of every list that splits expenses by default,
// so no future bill or expense is split with someone who is gone.
func TestDeleteMember_LeavesNoSplitDefaults(t *testing.T) {
	f := setupMoneyFixture(t)
	f.router.DELETE("/members/:id", middleware.AuthRequired(), NewMemberHandler(f.db).Delete)

	carol := itoa(f.mCarol.ID)
	bob := itoa(f.mBob.ID)
	settings := models.UserSettings{UserID: f.alice.ID, DefaultSplitWithMemberIDs: "[" + bob + "," + carol + "]"}
	custom := models.Utility{UserID: f.alice.ID, PropertyID: f.prop.ID, Type: "water", Provider: "Aqua", IsActive: true,
		SplitOverride: "custom", SplitMemberIDs: "[" + carol + "]"}
	garbled := models.Utility{UserID: f.alice.ID, PropertyID: f.prop.ID, Type: "gas", Provider: "Gas Co", IsActive: true,
		SplitOverride: "custom", SplitMemberIDs: "[" + carol + ","}
	mustCreate(t, f.db, &settings, &custom, &garbled)

	rec := doJSON(t, f.router, http.MethodDelete, "/members/"+carol, f.aliceTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete member: status %d, body %s", rec.Code, rec.Body.String())
	}

	f.db.First(&settings, settings.ID)
	f.db.First(&custom, custom.ID)
	f.db.First(&garbled, garbled.ID)
	if settings.DefaultSplitWithMemberIDs != "["+bob+"]" {
		t.Errorf("alice's default split = %s, want [%s]", settings.DefaultSplitWithMemberIDs, bob)
	}
	if custom.SplitMemberIDs != "[]" {
		t.Errorf("service split = %s, want []", custom.SplitMemberIDs)
	}
	// A list that does not parse is left for whoever fixes it, not rewritten.
	if garbled.SplitMemberIDs != "["+carol+"," {
		t.Errorf("unparseable service split rewritten to %s", garbled.SplitMemberIDs)
	}
}

func TestListProperties_NoHomes(t *testing.T) {
	f := setupPropertyFixture(t)
	nobody := &models.User{Email: "new@example.com", PasswordHash: "x", Name: "New", Role: "user", IsActive: true}
	mustCreate(t, f.db, nobody)

	rec := doJSON(t, f.router, http.MethodGet, "/properties", testutil.SignToken(t, nobody), nil)
	if rec.Code != http.StatusOK || rec.Body.String() != "[]" {
		t.Errorf("status %d, body %s; want 200 []", rec.Code, rec.Body.String())
	}
}
