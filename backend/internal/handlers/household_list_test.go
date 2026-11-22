package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
)

func TestHouseholdList_OnlyTheUsersOwnWithRoleAndPrimaryHome(t *testing.T) {
	f := setupPropertyFixture(t)
	r := gin.New()
	r.GET("/households", middleware.AuthRequired(), NewHouseholdHandler(f.db).List)

	list := func(tok string) []HouseholdSummary {
		t.Helper()
		rec := doJSON(t, r, http.MethodGet, "/households", tok, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
		}
		var out []HouseholdSummary
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return out
	}

	owner := list(f.ownerTok)
	if len(owner) != 1 || owner[0].ID != f.home.HouseholdID || !owner[0].IsAdmin || owner[0].PrimaryHomeID != f.home.ID {
		t.Errorf("owner households = %+v, want the shared one, as admin, with its primary home", owner)
	}

	member := list(f.memTok)
	if len(member) != 1 || member[0].ID != f.home.HouseholdID || member[0].IsAdmin {
		t.Errorf("member households = %+v, want only the shared one, not as admin", member)
	}
}

func TestUtilityList_ScopedToTheRequestedHousehold(t *testing.T) {
	f := setupPropertyFixture(t)
	r := gin.New()
	r.GET("/utilities", middleware.AuthRequired(), NewUtilityHandler(f.db).List)

	count := func(tok, query string) (int, int) {
		t.Helper()
		rec := doJSON(t, r, http.MethodGet, "/utilities"+query, tok, nil)
		var out []map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, len(out)
	}
	if code, n := count(f.ownerTok, "?household_id="+itoa(f.home.HouseholdID)); code != http.StatusOK || n != 1 {
		t.Errorf("home household: status %d, %d services; want 200 and its one service", code, n)
	}
	if code, n := count(f.neighTok, "?household_id="+itoa(f.other.HouseholdID)); code != http.StatusOK || n != 0 {
		t.Errorf("neighbour's household: status %d, %d services; want 200 and none", code, n)
	}
	if code, _ := count(f.memTok, "?household_id="+itoa(f.other.HouseholdID)); code != http.StatusForbidden {
		t.Errorf("foreign household: status %d, want 403", code)
	}
}
