// Package testutil provides shared setup helpers for the Go test suite.
// Tests run against an in-memory SQLite database so they are hermetic and
// fast enough to be part of CI on the Raspberry Pi target.
package testutil

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// TestJWTSecret is the secret used in tests. The handler & middleware code
// reads JWT_SECRET from the environment; tests set it via t.Setenv.
const TestJWTSecret = "test-secret-do-not-use-in-prod"

// NewDB returns a fresh in-memory SQLite DB with all migrations applied.
// Each call returns an isolated database so tests never share state.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	db.Exec("PRAGMA foreign_keys=ON;")
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}
	return db
}

// SignToken produces a valid access token for the given user, signed with
// TestJWTSecret — so tests must set JWT_SECRET to that value beforehand.
func SignToken(t *testing.T, user *models.User) string {
	t.Helper()
	claims := middleware.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(TestJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tok
}
