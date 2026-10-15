package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/testutil"
)

// The invariants of the session: the refresh token never reaches the response
// body, a password change revokes the sessions issued before it, and a refresh
// validates the token type, its version and the state of the account.

func newSessionTestRouter(t *testing.T) (*gin.Engine, *AuthHandler) {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewAuthHandler(db)
	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.RefreshToken)
	r.POST("/auth/logout", h.Logout)
	return r, h
}

// refreshCookieFrom returns the refresh cookie the response set, or nil.
func refreshCookieFrom(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			return c
		}
	}
	return nil
}

// The long-lived credential travels in an HttpOnly cookie and never in the
// body, where any script in the page could read it.
func TestLogin_RefreshTokenOnlyInHttpOnlyCookie(t *testing.T) {
	r, h := newSessionTestRouter(t)
	seedUser(t, h, "alice@example.com", "hunter2ABC")

	rec := postJSON(t, r, "/auth/login", map[string]string{
		"email": "alice@example.com", "password": "hunter2ABC",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, leaked := body["refresh_token"]; leaked {
		t.Errorf("refresh_token present in login response: %v", body)
	}
	if tok, ok := body["token"].(string); !ok || tok == "" {
		t.Errorf("no access token in login response: %v", body)
	}

	cookie := refreshCookieFrom(rec)
	if cookie == nil {
		t.Fatal("no refresh cookie set")
	}
	if !cookie.HttpOnly {
		t.Error("refresh cookie is not HttpOnly")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("SameSite = %v, want Strict", cookie.SameSite)
	}
	if cookie.Path != refreshCookiePath {
		t.Errorf("Path = %q, want %q", cookie.Path, refreshCookiePath)
	}
}

// Refreshing with the cookie alone must work: the client never sees the token.
func TestRefresh_UsesCookie(t *testing.T) {
	r, h := newSessionTestRouter(t)
	seedUser(t, h, "alice@example.com", "hunter2ABC")

	login := postJSON(t, r, "/auth/login", map[string]string{
		"email": "alice@example.com", "password": "hunter2ABC",
	})
	cookie := refreshCookieFrom(login)
	if cookie == nil {
		t.Fatal("no refresh cookie after login")
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body %s", rec.Code, rec.Body.String())
	}
}

// Both kinds are signed with the same secret; only the type claim separates
// them.
func TestRefresh_RejectsAccessToken(t *testing.T) {
	r, h := newSessionTestRouter(t)
	user := seedUser(t, h, "alice@example.com", "hunter2ABC")
	access := testutil.SignToken(t, user)

	rec := postJSON(t, r, "/auth/refresh", map[string]string{"refresh_token": access})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (access token sent to /refresh)", rec.Code)
	}
}

// A refresh token issued before a password change must stop working.
// ChangePassword and ResetPassword bump the version; this bumps it directly.
func TestRefresh_RejectsRevokedTokenVersion(t *testing.T) {
	r, h := newSessionTestRouter(t)
	user := seedUser(t, h, "alice@example.com", "hunter2ABC")

	login := postJSON(t, r, "/auth/login", map[string]string{
		"email": "alice@example.com", "password": "hunter2ABC",
	})
	cookie := refreshCookieFrom(login)
	if cookie == nil {
		t.Fatal("no refresh cookie after login")
	}

	if err := h.db.Model(user).Update("token_version", user.TokenVersion+1).Error; err != nil {
		t.Fatalf("bump token version: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (revoked refresh token)", rec.Code)
	}
}

// Deactivating an account must end its sessions at the next refresh, not only
// at the next login.
func TestRefresh_RejectsInactiveAccount(t *testing.T) {
	r, h := newSessionTestRouter(t)
	user := seedUser(t, h, "alice@example.com", "hunter2ABC")

	login := postJSON(t, r, "/auth/login", map[string]string{
		"email": "alice@example.com", "password": "hunter2ABC",
	})
	cookie := refreshCookieFrom(login)
	if cookie == nil {
		t.Fatal("no refresh cookie after login")
	}

	if err := h.db.Model(user).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (inactive account)", rec.Code)
	}
}

func TestRefresh_WithoutTokenIs401(t *testing.T) {
	r, _ := newSessionTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// Only the server can expire the cookie — it is HttpOnly.
func TestLogout_ExpiresRefreshCookie(t *testing.T) {
	r, _ := newSessionTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	cookie := refreshCookieFrom(rec)
	if cookie == nil {
		t.Fatal("logout did not set the refresh cookie")
	}
	if cookie.MaxAge >= 0 || cookie.Value != "" {
		t.Errorf("cookie not expired: value=%q MaxAge=%d", cookie.Value, cookie.MaxAge)
	}
}

// Every session of the user is revoked, then the caller gets a fresh one so
// the tab that made the change stays signed in.
func TestChangePassword_RevokesOtherSessionsAndReissues(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewAuthHandler(db)
	user := seedUser(t, h, "alice@example.com", "hunter2ABC")

	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.RefreshToken)
	// The real route sits behind AuthRequired; inject the user id instead.
	r.PUT("/settings/password", func(c *gin.Context) {
		c.Set("user_id", user.ID)
	}, h.ChangePassword)

	login := postJSON(t, r, "/auth/login", map[string]string{
		"email": "alice@example.com", "password": "hunter2ABC",
	})
	oldCookie := refreshCookieFrom(login)
	if oldCookie == nil {
		t.Fatal("no refresh cookie after login")
	}

	buf, _ := json.Marshal(map[string]string{
		"current_password": "hunter2ABC", "new_password": "brand-new-pass",
	})
	req := httptest.NewRequest(http.MethodPut, "/settings/password", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("change password: status %d, body %s", rec.Code, rec.Body.String())
	}

	// The caller keeps a working session.
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if tok, ok := body["token"].(string); !ok || tok == "" {
		t.Errorf("no replacement access token returned: %v", body)
	}
	if refreshCookieFrom(rec) == nil {
		t.Error("no replacement refresh cookie returned")
	}

	// The session that existed before the change does not.
	stale := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	stale.AddCookie(oldCookie)
	staleRec := httptest.NewRecorder()
	r.ServeHTTP(staleRec, stale)
	if staleRec.Code != http.StatusUnauthorized {
		t.Fatalf("stale refresh: status = %d, want 401", staleRec.Code)
	}
}

// The password minimum is a server rule, not a client-side suggestion.
func TestRegister_RejectsShortPassword(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewAuthHandler(db)
	r := gin.New()
	r.POST("/auth/register", h.Register)

	short := strings.Repeat("a", MinPasswordLength-1)
	rec := postJSON(t, r, "/auth/register", map[string]any{
		"email": "short@example.com", "password": short, "name": "Test",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a %d-character password", rec.Code, len(short))
	}
}

// Registration names the field that failed: one combined code told anyone who
// typed an address like "admin@admin" that their password was too short.
func TestRegister_ReportsTheFailingField(t *testing.T) {
	cases := []struct {
		name     string
		body     map[string]any
		wantCode string
	}{
		{
			name:     "no domain suffix",
			body:     map[string]any{"email": "admin@admin", "password": "long-enough", "name": "Test"},
			wantCode: "invalid_email",
		},
		{
			name:     "short password",
			body:     map[string]any{"email": "a@example.com", "password": "short", "name": "Test"},
			wantCode: "password_too_short",
		},
		{
			name:     "missing name",
			body:     map[string]any{"email": "b@example.com", "password": "long-enough"},
			wantCode: "name_required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
			db := testutil.NewDB(t)
			h := NewAuthHandler(db)
			r := gin.New()
			r.POST("/auth/register", h.Register)

			rec := postJSON(t, r, "/auth/register", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400, body %s", rec.Code, rec.Body.String())
			}
			var resp struct {
				ErrorCode string `json:"error_code"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.ErrorCode != tc.wantCode {
				t.Errorf("error_code = %q, want %q", resp.ErrorCode, tc.wantCode)
			}
		})
	}
}

// A reset is the recovery path for a compromised account, so it must end the
// sessions that account already had. ResetPassword had no test at all.
func TestResetPassword_RevokesSessions(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	h := NewAuthHandler(db)
	user := seedUser(t, h, "alice@example.com", "hunter2ABC")

	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.RefreshToken)
	r.POST("/auth/reset-password", h.ResetPassword)

	login := postJSON(t, r, "/auth/login", map[string]string{
		"email": "alice@example.com", "password": "hunter2ABC",
	})
	cookie := refreshCookieFrom(login)
	if cookie == nil {
		t.Fatal("no refresh cookie after login")
	}

	expires := time.Now().Add(time.Hour)
	if err := db.Model(user).Updates(map[string]any{
		"password_reset_token":   "reset-token",
		"password_reset_expires": expires,
	}).Error; err != nil {
		t.Fatalf("seed reset token: %v", err)
	}

	rec := postJSON(t, r, "/auth/reset-password", map[string]string{
		"token": "reset-token", "new_password": "brand-new-pass",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("reset: status %d, body %s", rec.Code, rec.Body.String())
	}

	stale := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	stale.AddCookie(cookie)
	staleRec := httptest.NewRecorder()
	r.ServeHTTP(staleRec, stale)
	if staleRec.Code != http.StatusUnauthorized {
		t.Fatalf("session from before the reset: status %d, want 401", staleRec.Code)
	}
}

// The Secure flag decides whether the refresh cookie may travel over plain
// HTTP, so each way of deciding it is worth pinning down.
func TestCookieSecure(t *testing.T) {
	cases := []struct {
		name    string
		env     string
		forward string
		want    bool
	}{
		{name: "plain HTTP LAN install", want: false},
		{name: "behind a TLS proxy", forward: "https", want: true},
		{name: "proxy header is case-insensitive", forward: "HTTPS", want: true},
		{name: "forced on", env: "true", want: true},
		{name: "forced off beats the proxy header", env: "false", forward: "https", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("COOKIE_SECURE", tc.env)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
			if tc.forward != "" {
				req.Header.Set("X-Forwarded-Proto", tc.forward)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req

			if got := cookieSecure(c); got != tc.want {
				t.Errorf("cookieSecure = %v, want %v", got, tc.want)
			}
		})
	}
}
