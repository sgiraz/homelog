package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func newPingRouter(mw ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	for _, m := range mw {
		r.Use(m)
	}
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	// OPTIONS is handled by the middleware itself (CORS preflight).
	r.OPTIONS("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return r
}

// newRateLimiter is package-local and parameterised so we can exercise the
// limiter without hitting the 100/minute production threshold.
func TestRateLimiter_BlocksAfterLimit(t *testing.T) {
	r := newPingRouter(newRateLimiter(3, time.Minute))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: status = %d, want 429", rec.Code)
	}
}

// TestRateLimiter_ConcurrentSafe is the regression test for the race on the
// previously unguarded clients map. Run with `go test -race`.
func TestRateLimiter_ConcurrentSafe(t *testing.T) {
	r := newPingRouter(newRateLimiter(10000, time.Minute))

	const goroutines = 40
	const perGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				req := httptest.NewRequest(http.MethodGet, "/ping", nil)
				// Vary the client IP so we exercise both insert + update paths
				// in the shared map from many goroutines.
				if id%2 == 0 {
					req.RemoteAddr = "10.0.0.1:1234"
				} else {
					req.RemoteAddr = "10.0.0.2:1234"
				}
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, req)
			}
		}(g)
	}
	wg.Wait()
}

// CORS allow-list: a known origin must be reflected in the ACA-Origin header;
// an unknown origin must NOT receive any CORS headers (browser will block it).
func TestCORS_AllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://ok.example.com")
	r := newPingRouter(CORS())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://ok.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ok.example.com" {
		t.Fatalf("ACA-Origin = %q, want the allowed origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("ACA-Credentials = %q, want true", got)
	}
}

func TestCORS_RejectsUnknownOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://ok.example.com")
	r := newPingRouter(CORS())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("ACA-Origin = %q, want empty for non-allowed origin", got)
	}
}

// Same-origin requests (no Origin header) still pass through cleanly — this
// is the normal case in production where the embedded frontend is same-origin.
func TestCORS_SameOrigin_NoHeaders(t *testing.T) {
	r := newPingRouter(CORS())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("ACA-Origin = %q, want empty", got)
	}
}
