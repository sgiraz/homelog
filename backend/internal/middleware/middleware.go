package middleware

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// CORS middleware restricts cross-origin requests to an explicit allow-list
// derived from the CORS_ALLOWED_ORIGINS env var (comma-separated). If the var
// is unset, localhost dev origins are allowed by default. Same-origin requests
// (no Origin header) are passed through untouched. Using "*" is rejected here
// because Access-Control-Allow-Credentials=true requires a specific origin.
func CORS() gin.HandlerFunc {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowed []string
	if raw == "" {
		allowed = []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:8080",
			"http://127.0.0.1:8080",
		}
	} else {
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				allowed = append(allowed, o)
			}
		}
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allowedSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowedSet[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			}
			// If the origin is not in the allow-list, no CORS headers are emitted
			// and the browser will block the response. Same-origin (no Origin
			// header) requests — e.g. the embedded frontend in prod — still work.
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthRequired middleware validates JWT tokens
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		jwtSecret := os.Getenv("JWT_SECRET")

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// AdminRequired middleware ensures user is admin
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// rateClient tracks request count inside a time window for a single key.
type rateClient struct {
	count      int
	lastAccess time.Time
}

// newRateLimiter returns a Gin middleware enforcing `limit` requests per
// `window` per remote IP. The shared map is protected by a mutex so the
// limiter is safe under the concurrent request handling Gin performs.
func newRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string]*rateClient)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists {
			clients[ip] = &rateClient{count: 1, lastAccess: now}
		} else if now.Sub(cl.lastAccess) > window {
			cl.count = 1
			cl.lastAccess = now
		} else {
			cl.count++
		}
		exceeded := exists && cl.count > limit && now.Sub(cl.lastAccess) <= window

		// Opportunistic cleanup while we still hold the lock.
		if len(clients) > 10000 {
			for k, v := range clients {
				if now.Sub(v.lastAccess) > window*10 {
					delete(clients, k)
				}
			}
		}
		mu.Unlock()

		if exceeded {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimiter is the global limiter applied to every request (100 req/min/IP).
func RateLimiter() gin.HandlerFunc {
	return newRateLimiter(100, time.Minute)
}

// AuthRateLimiter is a stricter limiter for /auth/* endpoints to slow down
// credential stuffing, email enumeration and password-reset abuse (10 req/min/IP).
func AuthRateLimiter() gin.HandlerFunc {
	return newRateLimiter(10, time.Minute)
}

// Logger middleware logs requests
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return ""
		// Custom log format if needed
	})
}

// GetUserID helper function to extract user ID from context
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}
