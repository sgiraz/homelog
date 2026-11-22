package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// Registration wrote its own UserSettings literal and left both notification
// preferences off, while the settings screen promised they were on: no admin
// was ever told about a join request.
func TestRegister_StartsWithNotificationsOn(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	r := gin.New()
	r.POST("/auth/register", NewAuthHandler(db).Register)

	// The first account takes the admin path, every later one the other.
	for _, email := range []string{"first@example.com", "second@example.com"} {
		rec := postJSON(t, r, "/auth/register", map[string]any{
			"email": email, "password": "hunter2ABC", "name": "Test",
		})
		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("register %s: status %d, body %s", email, rec.Code, rec.Body.String())
		}

		var settings models.UserSettings
		if err := db.Joins("JOIN users ON users.id = user_settings.user_id").
			Where("users.email = ?", email).First(&settings).Error; err != nil {
			t.Fatalf("settings for %s: %v", email, err)
		}
		want := models.NewUserSettings(settings.UserID, settings.Language)
		if settings.NotifyJoinRequests != want.NotifyJoinRequests ||
			settings.NotifySharedExpenses != want.NotifySharedExpenses {
			t.Errorf("%s: notify_join_requests=%v notify_shared_expenses=%v, want %v/%v",
				email, settings.NotifyJoinRequests, settings.NotifySharedExpenses,
				want.NotifyJoinRequests, want.NotifySharedExpenses)
		}
	}
}

// Every UserSettings row starts from models.NewUserSettings. A hand-written
// literal with fields is how the defaults drifted apart in the first place.
func TestUserSettings_CreatedOnlyThroughTheConstructor(t *testing.T) {
	literal := regexp.MustCompile(`models\.UserSettings\{\s*\w+\s*:`)
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if literal.Match(src) {
			t.Errorf("%s builds a models.UserSettings literal; use models.NewUserSettings", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}
