package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// A role change has to reach the sessions the user already holds: the old
// role travels inside tokens that are still valid otherwise.
func TestToggleAdmin_RevokesSessions(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewAdminHandler(db)
	authH := NewAuthHandler(db)

	admin := seedUser(t, authH, "admin@example.com", "hunter2ABC")
	if err := db.Model(admin).Update("role", "admin").Error; err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	target := seedUser(t, authH, "bob@example.com", "hunter2ABC")

	r := gin.New()
	r.PUT("/admin/users/:id/role", func(c *gin.Context) {
		c.Set("user_id", admin.ID)
	}, h.ToggleAdmin)

	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req := httptest.NewRequest(http.MethodPut, "/admin/users/"+itoa(target.ID)+"/role", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}

	var updated models.User
	if err := db.First(&updated, target.ID).Error; err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if updated.Role != "admin" {
		t.Errorf("role = %q, want admin", updated.Role)
	}
	if updated.TokenVersion != target.TokenVersion+1 {
		t.Errorf("TokenVersion = %d, want %d — the role change did not revoke the sessions",
			updated.TokenVersion, target.TokenVersion+1)
	}
}

// Changing your own role is refused, so it must not bump anything either.
func TestToggleAdmin_RefusesSelf(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewAdminHandler(db)
	admin := seedUser(t, NewAuthHandler(db), "admin@example.com", "hunter2ABC")

	r := gin.New()
	r.PUT("/admin/users/:id/role", func(c *gin.Context) {
		c.Set("user_id", admin.ID)
	}, h.ToggleAdmin)

	body, _ := json.Marshal(map[string]string{"role": "user"})
	req := httptest.NewRequest(http.MethodPut, "/admin/users/"+itoa(admin.ID)+"/role", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var unchanged models.User
	if err := db.First(&unchanged, admin.ID).Error; err != nil {
		t.Fatalf("reload admin: %v", err)
	}
	if unchanged.TokenVersion != admin.TokenVersion {
		t.Errorf("TokenVersion changed on a refused request: %d", unchanged.TokenVersion)
	}
}
