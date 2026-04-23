package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// authzFixture sets up two independent households so we can verify that a
// user from household A cannot read data from household B.
type authzFixture struct {
	router   *gin.Engine
	alice    *models.User // admin of propA
	bob      *models.User // admin of propB
	propA    models.Property
	propB    models.Property
	memberA  models.HouseholdMember
	memberB  models.HouseholdMember
	aliceTok string
	bobTok   string
}

func setupAuthzFixture(t *testing.T) *authzFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)

	db := testutil.NewDB(t)

	// Two users, two properties, each user admin of their own.
	now := time.Now()
	alice := &models.User{Email: "alice@example.com", PasswordHash: "x", Name: "Alice", Role: "admin", IsActive: true}
	bob := &models.User{Email: "bob@example.com", PasswordHash: "x", Name: "Bob", Role: "admin", IsActive: true}
	if err := db.Create(alice).Error; err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := db.Create(bob).Error; err != nil {
		t.Fatalf("create bob: %v", err)
	}

	propA := models.Property{UserID: alice.ID, Name: "Casa Alice", Type: "owned", StartDate: now, IsCurrent: true, Residents: 1}
	propB := models.Property{UserID: bob.ID, Name: "Casa Bob", Type: "owned", StartDate: now, IsCurrent: true, Residents: 1}
	if err := db.Create(&propA).Error; err != nil {
		t.Fatalf("create propA: %v", err)
	}
	if err := db.Create(&propB).Error; err != nil {
		t.Fatalf("create propB: %v", err)
	}

	memberA := models.HouseholdMember{PropertyID: propA.ID, UserID: &alice.ID, Name: alice.Name, Role: "admin", IsVirtual: false}
	memberB := models.HouseholdMember{PropertyID: propB.ID, UserID: &bob.ID, Name: bob.Name, Role: "admin", IsVirtual: false}
	if err := db.Create(&memberA).Error; err != nil {
		t.Fatalf("create memberA: %v", err)
	}
	if err := db.Create(&memberB).Error; err != nil {
		t.Fatalf("create memberB: %v", err)
	}

	// Household settings row for propB so we cover the lookup-hit path too.
	if err := db.Create(&models.HouseholdSettings{PropertyID: propB.ID}).Error; err != nil {
		t.Fatalf("create household settings: %v", err)
	}

	// Wire the routes exactly as main.go does, under the real AuthRequired
	// middleware so we exercise the same context plumbing that prod uses.
	memberHandler := NewMemberHandler(db)
	settingsHandler := NewSettingsHandler(db)

	r := gin.New()
	protected := r.Group("")
	protected.Use(middleware.AuthRequired())
	protected.GET("/properties/:id/members", memberHandler.List)
	protected.GET("/members/:id", memberHandler.Get)
	protected.GET("/properties/:id/settings", settingsHandler.GetHouseholdSettings)

	return &authzFixture{
		router:   r,
		alice:    alice,
		bob:      bob,
		propA:    propA,
		propB:    propB,
		memberA:  memberA,
		memberB:  memberB,
		aliceTok: testutil.SignToken(t, alice),
		bobTok:   testutil.SignToken(t, bob),
	}
}

func doGET(t *testing.T, r http.Handler, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestMembersList_NonMember_403(t *testing.T) {
	f := setupAuthzFixture(t)

	rec := doGET(t, f.router, "/properties/"+itoa(f.propB.ID)+"/members", f.aliceTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (alice reading Bob's members)", rec.Code)
	}
}

func TestMembersList_Member_200(t *testing.T) {
	f := setupAuthzFixture(t)

	rec := doGET(t, f.router, "/properties/"+itoa(f.propA.ID)+"/members", f.aliceTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("expected 1 member, got %d", len(body))
	}
}

func TestMemberGet_NonMember_403(t *testing.T) {
	f := setupAuthzFixture(t)

	rec := doGET(t, f.router, "/members/"+itoa(f.memberB.ID), f.aliceTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (alice reading Bob's member)", rec.Code)
	}
}

func TestHouseholdSettings_NonMember_403(t *testing.T) {
	f := setupAuthzFixture(t)

	rec := doGET(t, f.router, "/properties/"+itoa(f.propB.ID)+"/settings", f.aliceTok)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (alice reading Bob's household settings)", rec.Code)
	}
}

func TestHouseholdSettings_Member_200(t *testing.T) {
	f := setupAuthzFixture(t)

	rec := doGET(t, f.router, "/properties/"+itoa(f.propB.ID)+"/settings", f.bobTok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// itoa is tiny and local — avoids pulling strconv into a test file that
// otherwise needs no other string formatting.
func itoa(u uint) string {
	if u == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for u > 0 {
		i--
		buf[i] = byte('0' + u%10)
		u /= 10
	}
	return string(buf[i:])
}
