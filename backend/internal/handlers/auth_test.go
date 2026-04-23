package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

func init() { gin.SetMode(gin.TestMode) }

// seedUser writes a user with a known password into the test DB.
func seedUser(t *testing.T, h *AuthHandler, email, password string) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         "Test",
		Role:         "user",
		IsActive:     true,
	}
	if err := h.db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func newAuthTestRouter(t *testing.T) (*gin.Engine, *AuthHandler) {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewAuthHandler(db)
	r := gin.New()
	r.POST("/auth/forgot-password", h.ForgotPassword)
	r.POST("/auth/login", h.Login)
	return r, h
}

func postJSON(t *testing.T, r http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// TestForgotPassword_DoesNotLeakToken is the regression test for the Phase A
// security fix: the reset token must never appear in the JSON response by
// default. A leaked token lets any caller who knows a registered email reset
// that account's password.
func TestForgotPassword_DoesNotLeakToken(t *testing.T) {
	r, h := newAuthTestRouter(t)
	seedUser(t, h, "alice@example.com", "hunter2ABC")

	rec := postJSON(t, r, "/auth/forgot-password", map[string]string{"email": "alice@example.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, leaked := resp["reset_token"]; leaked {
		t.Fatalf("reset_token present in response: %v", resp)
	}
}

// When DEV_EXPOSE_RESET_TOKEN=true the token is expected in the response to
// support local development without SMTP.
func TestForgotPassword_DevGateReturnsToken(t *testing.T) {
	r, h := newAuthTestRouter(t)
	seedUser(t, h, "bob@example.com", "hunter2ABC")
	t.Setenv("DEV_EXPOSE_RESET_TOKEN", "true")

	rec := postJSON(t, r, "/auth/forgot-password", map[string]string{"email": "bob@example.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if tok, ok := resp["reset_token"].(string); !ok || tok == "" {
		t.Fatalf("expected reset_token in dev mode, got %v", resp)
	}
}

// Unknown email must not produce a different response than a known one (email
// enumeration protection). Both cases must also omit the token.
func TestForgotPassword_UnknownEmailGenericResponse(t *testing.T) {
	r, _ := newAuthTestRouter(t)

	rec := postJSON(t, r, "/auth/forgot-password", map[string]string{"email": "nobody@example.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if _, ok := resp["reset_token"]; ok {
		t.Fatalf("reset_token must not be present for unknown email: %v", resp)
	}
	if _, ok := resp["message"]; !ok {
		t.Fatalf("expected generic message, got %v", resp)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	r, h := newAuthTestRouter(t)
	seedUser(t, h, "carol@example.com", "correct-horse")

	rec := postJSON(t, r, "/auth/login", map[string]string{
		"email":    "carol@example.com",
		"password": "wrong",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
