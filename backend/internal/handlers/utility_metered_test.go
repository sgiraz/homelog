package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// createUtilityFixture wires just POST /utilities against a property the user
// owns, which is all these cases need.
type utilityFixture struct {
	router *gin.Engine
	prop   models.Property
	token  string
}

func setupUtilityFixture(t *testing.T) *utilityFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)

	user := models.User{Email: "owner@example.com", PasswordHash: "x", Name: "Owner", Role: "admin", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	prop := models.Property{UserID: user.ID, Name: "Casa", IsCurrent: true}
	if err := db.Create(&prop).Error; err != nil {
		t.Fatalf("create property: %v", err)
	}
	member := models.HouseholdMember{PropertyID: prop.ID, UserID: &user.ID, Name: user.Name, Role: "admin"}
	if err := db.Create(&member).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(middleware.AuthRequired())
	g.POST("/utilities", NewUtilityHandler(db).Create)

	return &utilityFixture{router: r, prop: prop, token: testutil.SignToken(t, &user)}
}

// A mortgage is repaid on an amortisation schedule — there is no meter to read.
// The client must not be able to say otherwise, whatever it sends.
func TestCreateUtility_FixedCostTypeIsNeverMetered(t *testing.T) {
	for _, utilityType := range models.FixedCostTypes() {
		t.Run(utilityType, func(t *testing.T) {
			f := setupUtilityFixture(t)
			rec := doJSON(t, f.router, http.MethodPost, "/utilities", f.token, map[string]any{
				"property_id": f.prop.ID,
				"type":        utilityType,
				"provider":    "Acme",
				"is_metered":  true, // the client insists — the server must not listen
			})
			if rec.Code != http.StatusCreated {
				t.Fatalf("create %s: status %d, body %s", utilityType, rec.Code, rec.Body.String())
			}
			var got models.Utility
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.IsMetered {
				t.Errorf("%s stored as metered: a fixed-cost service has no meter, so it must never offer readings", utilityType)
			}
		})
	}
}

// The reverse must still work: a metered type may be turned off explicitly for
// a flat-rate contract, and stays metered when the client says nothing.
func TestCreateUtility_MeteredTypeHonoursExplicitOptOut(t *testing.T) {
	f := setupUtilityFixture(t)

	rec := doJSON(t, f.router, http.MethodPost, "/utilities", f.token, map[string]any{
		"property_id": f.prop.ID, "type": "water", "provider": "Acme", "is_metered": false,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("opt-out: status %d, body %s", rec.Code, rec.Body.String())
	}
	var optedOut models.Utility
	if err := json.Unmarshal(rec.Body.Bytes(), &optedOut); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if optedOut.IsMetered {
		t.Error("an explicit is_metered=false on a metered type must be honoured")
	}

	f2 := setupUtilityFixture(t)
	rec2 := doJSON(t, f2.router, http.MethodPost, "/utilities", f2.token, map[string]any{
		"property_id": f2.prop.ID, "type": "electricity", "provider": "Acme",
	})
	if rec2.Code != http.StatusCreated {
		t.Fatalf("default: status %d, body %s", rec2.Code, rec2.Body.String())
	}
	var byDefault models.Utility
	if err := json.Unmarshal(rec2.Body.Bytes(), &byDefault); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !byDefault.IsMetered {
		t.Error("electricity must default to metered when the client says nothing")
	}
}
