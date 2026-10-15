package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-do-not-use-in-prod"

// signClaims forges claim shapes a real issuer would never produce.
func signClaims(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func claimsFor(tokenType string, ttl time.Duration) JWTClaims {
	c := JWTClaims{
		UserID:    7,
		Email:     "alice@example.com",
		Role:      "user",
		TokenType: tokenType,
	}
	if ttl != 0 {
		c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ttl))
	}
	return c
}

// The invariant the "typ" claim exists for: a 7-day refresh token must not
// open a protected route. Without it the two kinds are interchangeable and the
// 15-minute access lifetime protects nothing.
func TestAuthRequired_RejectsRefreshTokenAsBearer(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	refresh := signClaims(t, jwt.SigningMethodHS256, []byte(testSecret),
		claimsFor(TokenTypeRefresh, 7*24*time.Hour))

	r := newPingRouter(AuthRequired())
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+refresh)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (refresh token used as bearer)", rec.Code)
	}
}

func TestAuthRequired_AcceptsAccessToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	access := signClaims(t, jwt.SigningMethodHS256, []byte(testSecret),
		claimsFor(TokenTypeAccess, 15*time.Minute))

	r := newPingRouter(AuthRequired())
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

// The mirror of the above: /auth/refresh must not accept an access token.
func TestParseToken_RejectsAccessTokenWhereRefreshExpected(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	access := signClaims(t, jwt.SigningMethodHS256, []byte(testSecret),
		claimsFor(TokenTypeAccess, 15*time.Minute))

	if _, err := ParseToken(access, TokenTypeRefresh); err != ErrWrongTokenType {
		t.Fatalf("err = %v, want ErrWrongTokenType", err)
	}
}

// No "typ" means a token issued before this change, or forged by hand.
func TestParseToken_RejectsUntypedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	untyped := signClaims(t, jwt.SigningMethodHS256, []byte(testSecret),
		claimsFor("", 15*time.Minute))

	if _, err := ParseToken(untyped, TokenTypeAccess); err == nil {
		t.Fatal("untyped token accepted as access token")
	}
}

// Without WithExpirationRequired, a token omitting "exp" is valid forever.
func TestParseToken_RequiresExpiry(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	noExpiry := signClaims(t, jwt.SigningMethodHS256, []byte(testSecret),
		claimsFor(TokenTypeAccess, 0))

	if _, err := ParseToken(noExpiry, TokenTypeAccess); err == nil {
		t.Fatal("token without exp accepted")
	}
}

func TestParseToken_RejectsExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	expired := signClaims(t, jwt.SigningMethodHS256, []byte(testSecret),
		claimsFor(TokenTypeAccess, -time.Minute))

	if _, err := ParseToken(expired, TokenTypeAccess); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestParseToken_RejectsWrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	foreign := signClaims(t, jwt.SigningMethodHS256, []byte("some-other-secret"),
		claimsFor(TokenTypeAccess, 15*time.Minute))

	if _, err := ParseToken(foreign, TokenTypeAccess); err == nil {
		t.Fatal("token signed with another secret accepted")
	}
}

// jwt/v5 refuses alg=none on its own; WithValidMethods makes that explicit
// rather than incidental.
func TestParseToken_RejectsUnsignedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	unsigned := signClaims(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType,
		claimsFor(TokenTypeAccess, 15*time.Minute))

	if _, err := ParseToken(unsigned, TokenTypeAccess); err == nil {
		t.Fatal("alg=none token accepted")
	}
}

// The refresh handler compares this against the user's current version.
func TestParseToken_CarriesTokenVersion(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	claims := claimsFor(TokenTypeRefresh, time.Hour)
	claims.TokenVersion = 4
	tok := signClaims(t, jwt.SigningMethodHS256, []byte(testSecret), claims)

	got, err := ParseToken(tok, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.TokenVersion != 4 {
		t.Fatalf("TokenVersion = %d, want 4", got.TokenVersion)
	}
}
