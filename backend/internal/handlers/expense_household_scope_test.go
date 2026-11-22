package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
)

// The expense list and stats follow the household the client is looking at;
// without one they span every household of the user, as before.
func TestExpenseList_ScopedToTheRequestedHousehold(t *testing.T) {
	f := setupPropertyFixture(t)
	h := NewExpenseHandler(f.db)
	r := gin.New()
	r.Use(middleware.AuthRequired())
	r.GET("/expenses", h.List)
	r.GET("/expenses/stats", h.GetStats)

	descriptions := func(query string) []string {
		t.Helper()
		rec := doJSON(t, r, http.MethodGet, "/expenses"+query, f.ownerTok, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list%s: status %d, body %s", query, rec.Code, rec.Body.String())
		}
		var out struct {
			Expenses []struct {
				Description string `json:"description"`
			} `json:"expenses"`
			Total int64 `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if int(out.Total) != len(out.Expenses) {
			t.Errorf("list%s: total %d but %d expenses", query, out.Total, len(out.Expenses))
		}
		var d []string
		for _, e := range out.Expenses {
			d = append(d, e.Description)
		}
		return d
	}

	if got := descriptions(""); len(got) != 2 {
		t.Errorf("all households: %v, want both expenses", got)
	}
	if got := descriptions("?household_id=" + itoa(f.other.HouseholdID)); len(got) != 1 || got[0] != "Rent" {
		t.Errorf("other household: %v, want only Rent", got)
	}
	if got := descriptions("?household_id=" + itoa(f.home.HouseholdID)); len(got) != 1 || got[0] != "Tiles" {
		t.Errorf("home household: %v, want only Tiles", got)
	}

	// Not a member of the other household: refused, not silently empty.
	for _, path := range []string{"/expenses", "/expenses/stats"} {
		rec := doJSON(t, r, http.MethodGet, path+"?household_id="+itoa(f.other.HouseholdID), f.memTok, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s for a foreign household: status %d, want 403", path, rec.Code)
		}
		if rec := doJSON(t, r, http.MethodGet, path+"?household_id=abc", f.ownerTok, nil); rec.Code != http.StatusBadRequest {
			t.Errorf("%s with a bad household id: status %d, want 400", path, rec.Code)
		}
	}
}
