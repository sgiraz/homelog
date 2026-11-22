package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
)

// The expense list and stats follow the household asked for, the user's own
// without one; another household is refused.
func TestExpenseList_ScopedToTheRequestedHousehold(t *testing.T) {
	f := setupPropertyFixture(t)
	h := NewExpenseHandler(f.db)
	r := gin.New()
	r.Use(middleware.AuthRequired())
	r.GET("/expenses", h.List)
	r.GET("/expenses/stats", h.GetStats)

	descriptions := func(tok, query string) []string {
		t.Helper()
		rec := doJSON(t, r, http.MethodGet, "/expenses"+query, tok, nil)
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

	if got := descriptions(f.ownerTok, ""); len(got) != 1 || got[0] != "Tiles" {
		t.Errorf("own household: %v, want only Tiles", got)
	}
	if got := descriptions(f.ownerTok, "?household_id="+itoa(f.home.HouseholdID)); len(got) != 1 || got[0] != "Tiles" {
		t.Errorf("home household: %v, want only Tiles", got)
	}
	if got := descriptions(f.neighTok, "?household_id="+itoa(f.other.HouseholdID)); len(got) != 1 || got[0] != "Rent" {
		t.Errorf("neighbour's household: %v, want only Rent", got)
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
