package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/i18n"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// An account with no user_settings row was reported, and then created, as
// Italian: three fallbacks carried their own "it" / "EUR" literals, so the
// whole UI and the onboarding switched to a language nobody had chosen.

func TestGetSettings_NoRowUsesTheDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewSettingsHandler(db)
	user := seedUser(t, NewAuthHandler(db), "alice@example.com", "hunter2ABC")

	// No UserSettings row is created for this user on purpose.
	r := gin.New()
	r.GET("/settings", func(c *gin.Context) { c.Set("user_id", user.ID) }, h.Get)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}

	var got struct {
		Language string `json:"language"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Language != models.DefaultLanguage {
		t.Errorf("language = %q, want %q", got.Language, models.DefaultLanguage)
	}
	if got.Currency != models.DefaultCurrency {
		t.Errorf("currency = %q, want %q", got.Currency, models.DefaultCurrency)
	}
}

func TestSeedDefaultData_UsesTheDefaults(t *testing.T) {
	db := testutil.NewDB(t)
	user := models.User{Email: "bob@example.com", PasswordHash: "x", Name: "Bob", Role: "user", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := database.SeedDefaultData(db, user.ID); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var settings models.UserSettings
	if err := db.Where("user_id = ?", user.ID).First(&settings).Error; err != nil {
		t.Fatalf("settings not created: %v", err)
	}
	if settings.Language != models.DefaultLanguage {
		t.Errorf("language = %q, want %q", settings.Language, models.DefaultLanguage)
	}
	if settings.Currency != models.DefaultCurrency {
		t.Errorf("currency = %q, want %q", settings.Currency, models.DefaultCurrency)
	}
}

// The join-request notification used to be hardcoded Italian for everyone.
// Like every notification, it belongs in the recipient's language, not the
// requester's.
func TestJoinRequest_NotifiesTheAdminInTheirOwnLanguage(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	authH := NewAuthHandler(db)
	h := NewJoinRequestHandler(db)

	admin := seedUser(t, authH, "admin@example.com", "hunter2ABC")
	joiner := seedUser(t, authH, "joiner@example.com", "hunter2ABC")
	if err := db.Model(joiner).Update("name", "Bob").Error; err != nil {
		t.Fatalf("name joiner: %v", err)
	}

	// The admin reads the app in German; the requester's language is irrelevant.
	const adminLang = "de"
	if err := db.Create(&models.UserSettings{
		UserID: admin.ID, Language: adminLang, Currency: models.DefaultCurrency,
		NotifyJoinRequests: true,
	}).Error; err != nil {
		t.Fatalf("admin settings: %v", err)
	}
	if err := db.Create(&models.UserSettings{
		UserID: joiner.ID, Language: "it", Currency: models.DefaultCurrency,
	}).Error; err != nil {
		t.Fatalf("joiner settings: %v", err)
	}

	prop := models.Property{UserID: admin.ID, Name: "Hauptwohnsitz", Type: "owned", StartDate: time.Now(), IsCurrent: true, Residents: 1}
	if err := db.Create(&prop).Error; err != nil {
		t.Fatalf("property: %v", err)
	}
	if err := db.Create(&models.HouseholdMember{
		PropertyID: prop.ID, UserID: &admin.ID, Name: "Admin", Role: "admin",
	}).Error; err != nil {
		t.Fatalf("member: %v", err)
	}

	r := gin.New()
	r.POST("/join-requests", func(c *gin.Context) { c.Set("user_id", joiner.ID) }, h.Create)
	rec := postJSON(t, r, "/join-requests", map[string]any{"property_id": prop.ID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}

	var n models.Notification
	if err := db.Where("user_id = ? AND type = ?", admin.ID, "join_request").First(&n).Error; err != nil {
		t.Fatalf("no notification for the admin: %v", err)
	}

	wantTitle := i18n.T(adminLang, "notification.join_request.title", "name", "Bob", "property", prop.Name)
	wantBody := i18n.T(adminLang, "notification.join_request.body", "name", "Bob", "property", prop.Name)
	if n.Title != wantTitle {
		t.Errorf("title = %q, want %q", n.Title, wantTitle)
	}
	if n.Content != wantBody {
		t.Errorf("content = %q, want %q", n.Content, wantBody)
	}
}

// The demo guard answered Italian prose; every refusal in this codebase
// returns a code the client translates.
func TestDemoGuard_ReturnsACode(t *testing.T) {
	t.Setenv("DEMO_MODE", "true")

	r := gin.New()
	r.POST("/auth/register", middleware.DemoGuard(), func(c *gin.Context) {
		t.Error("the guard let the request through")
	})

	rec := postJSON(t, r, "/auth/register", map[string]string{})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ErrorCode != "demo_mode_forbidden" {
		t.Errorf("error_code = %q, want demo_mode_forbidden", body.ErrorCode)
	}
}
