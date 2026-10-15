package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/i18n"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// MinPasswordLength is the shortest password accepted on registration, change
// and reset. Keep in sync with the binding tags below and the client checks.
const MinPasswordLength = 8

// RegisterRequest represents registration input
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
	// Language is the browser locale the signup form was rendered in. Optional
	// and best-effort: anything unsupported falls back to models.DefaultLanguage.
	Language string `json:"language"`
}

// LoginRequest represents login input
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse represents authentication response. The refresh token is not
// in the body: it travels only in an HttpOnly cookie, out of reach of any
// script running in the page.
type TokenResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// refreshCookieName is the HttpOnly cookie that carries the refresh token.
const refreshCookieName = "homelog_refresh"

// refreshCookiePath scopes the cookie to the auth endpoints, so it does not
// ride along with every API request.
const refreshCookiePath = "/api/v1/auth"

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// cookieSecure decides the cookie's Secure flag: COOKIE_SECURE wins when set,
// otherwise it follows the request's scheme (directly or via a TLS-terminating
// proxy), so plain-HTTP LAN installs keep working.
func cookieSecure(c *gin.Context) bool {
	switch os.Getenv("COOKIE_SECURE") {
	case "true":
		return true
	case "false":
		return false
	}
	return c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func setRefreshCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   int(refreshTokenTTL / time.Second),
		HttpOnly: true,
		Secure:   cookieSecure(c),
		SameSite: http.SameSiteStrictMode,
	})
}

func clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecure(c),
		SameSite: http.SameSiteStrictMode,
	})
}

// registerFailedField returns the struct field of the first validation error,
// or "" for a failure that is not one (bad JSON).
func registerFailedField(err error) string {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) && len(verrs) > 0 {
		return verrs[0].Field()
	}
	return ""
}

// issueSession generates a fresh token pair, puts the refresh token in its
// cookie and returns the access token.
func (h *AuthHandler) issueSession(c *gin.Context, user *models.User) (string, error) {
	token, refreshToken, err := h.generateTokens(user)
	if err != nil {
		return "", err
	}
	setRefreshCookie(c, refreshToken)
	return token, nil
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Name the field that failed: one combined code told someone who typed
		// "admin@admin" that their password was too short.
		switch registerFailedField(err) {
		case "Email":
			apierr.Fail(c, http.StatusBadRequest, "invalid_email", err.Error())
		case "Password":
			apierr.Fail(c, http.StatusBadRequest, "password_too_short", err.Error())
		case "Name":
			apierr.Fail(c, http.StatusBadRequest, "name_required", err.Error())
		default:
			apierr.Fail(c, http.StatusBadRequest, "invalid_registration", err.Error())
		}
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		apierr.Fail(c, http.StatusConflict, "email_already_registered", "This email is already registered")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to hash password")
		return
	}

	// Determine role (first user is admin)
	var userCount int64
	h.db.Model(&models.User{}).Count(&userCount)
	role := "user"
	isFirstUser := userCount == 0
	if isFirstUser {
		role = "admin"
	}

	// A new account starts in the language the user is registering in, not in a
	// hardcoded default.
	language := models.NormalizeLanguage(req.Language)

	tx := h.db.Begin()

	// Create user
	user := models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         role,
		IsActive:     true,
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create user")
		return
	}

	log.Printf("✅ User created: ID=%d, Email=%s, Role=%s", user.ID, user.Email, user.Role)

	// Auto-create Property + HouseholdSettings + UserSettings for first user (admin)
	if isFirstUser {
		// Create default property
		now := time.Now()
		property := models.Property{
			UserID:    user.ID,
			Name:      i18n.T(language, "property.default_name"),
			Address:   "",
			Type:      "owned",
			StartDate: now,
			IsCurrent: true,
			Residents: 1,
		}

		if err := tx.Create(&property).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating default property: %v", err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create default property")
			return
		}

		log.Printf("✅ Default property created: ID=%d, Name=%s", property.ID, property.Name)

		// Create household settings for the property
		householdSettings := models.HouseholdSettings{
			PropertyID:       property.ID,
			SplitMode:        false,
			DefaultSplitType: "equal",
		}

		if err := tx.Create(&householdSettings).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating household settings: %v", err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create household settings")
			return
		}

		log.Printf("✅ Household settings created for property ID=%d", property.ID)

		// Create user settings
		userSettings := models.UserSettings{
			UserID:                    user.ID,
			Language:                  language,
			Currency:                  models.DefaultCurrency,
			Theme:                     "auto",
			DateFormat:                "DD/MM/YYYY",
			DefaultSplitWithMemberIDs: "",
			EmailNotifications:        true,
			BillDueAlertDays:          3,
		}

		if err := tx.Create(&userSettings).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating user settings: %v", err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create user settings")
			return
		}

		log.Printf("✅ User settings created for user ID=%d", user.ID)

		// Create HouseholdMember for the admin user (linked to their User account)
		adminMember := models.HouseholdMember{
			PropertyID: property.ID,
			UserID:     &user.ID,
			Name:       user.Name,
			Role:       "admin",
			IsVirtual:  false,
		}

		if err := tx.Create(&adminMember).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating household member: %v", err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create household member")
			return
		}

		log.Printf("✅ Household member created: ID=%d, Name=%s, UserID=%d", adminMember.ID, adminMember.Name, user.ID)

		// Seed default categories for first user
		if err := database.SeedDefaultCategories(tx); err != nil {
			tx.Rollback()
			log.Printf("ERROR seeding default categories: %v", err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to seed default categories")
			return
		}
		log.Printf("✅ Default categories seeded")
	} else {
		// Non-first user: create only user settings.
		// NO auto-join to any property — the onboarding wizard will guide the user
		// to either create a new property or request to join an existing one.
		userSettings := models.UserSettings{
			UserID:                    user.ID,
			Language:                  language,
			Currency:                  models.DefaultCurrency,
			Theme:                     "auto",
			DateFormat:                "DD/MM/YYYY",
			DefaultSplitWithMemberIDs: "",
			EmailNotifications:        true,
			BillDueAlertDays:          3,
		}

		if err := tx.Create(&userSettings).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating user settings: %v", err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create user settings")
			return
		}

		log.Printf("✅ User settings created for user ID=%d (no property yet — onboarding pending)", user.ID)
	}

	if err := tx.Commit().Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to complete registration")
		return
	}

	// Generate tokens
	token, err := h.issueSession(c, &user)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to generate tokens")
		return
	}

	c.JSON(http.StatusCreated, TokenResponse{Token: token, User: user})
}

// Login authenticates a user
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		apierr.Fail(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Check if user is active
	if !user.IsActive {
		apierr.Fail(c, http.StatusForbidden, "account_inactive", "This account is inactive")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		apierr.Fail(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Generate tokens
	token, err := h.issueSession(c, &user)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to generate tokens")
		return
	}

	c.JSON(http.StatusOK, TokenResponse{Token: token, User: user})
}

// RefreshToken issues a new access token. The refresh token arrives in its
// HttpOnly cookie, or in the body for non-browser clients.
//
// It is rejected unless it is a refresh token, its user still exists and is
// active, and its version matches the user's current TokenVersion.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	raw, _ := c.Cookie(refreshCookieName)
	if raw == "" {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = c.ShouldBindJSON(&req)
		raw = req.RefreshToken
	}
	if raw == "" {
		apierr.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "Missing refresh token")
		return
	}

	claims, err := middleware.ParseToken(raw, middleware.TokenTypeRefresh)
	if err != nil {
		clearRefreshCookie(c)
		apierr.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "Invalid refresh token")
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		clearRefreshCookie(c)
		apierr.Fail(c, http.StatusUnauthorized, "user_not_found", "User not found")
		return
	}
	if !user.IsActive {
		clearRefreshCookie(c)
		apierr.Fail(c, http.StatusForbidden, "account_inactive", "This account is inactive")
		return
	}
	if claims.TokenVersion != user.TokenVersion {
		clearRefreshCookie(c)
		apierr.Fail(c, http.StatusUnauthorized, "invalid_refresh_token", "Refresh token has been revoked")
		return
	}

	// Role and email come from the DB, not the old token, so a demotion takes
	// effect on the next refresh.
	newToken, err := h.issueSession(c, &user)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to generate tokens")
		return
	}

	c.JSON(http.StatusOK, TokenResponse{Token: newToken, User: user})
}

// Logout clears the refresh-token cookie. The access token is short-lived and
// is simply dropped by the client.
func (h *AuthHandler) Logout(c *gin.Context) {
	clearRefreshCookie(c)
	c.Status(http.StatusNoContent)
}

// generateTokens creates access and refresh tokens
func (h *AuthHandler) generateTokens(user *models.User) (string, string, error) {
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	now := time.Now()

	sign := func(tokenType string, ttl time.Duration) (string, error) {
		claims := middleware.JWTClaims{
			UserID:       user.ID,
			Email:        user.Email,
			Role:         user.Role,
			TokenType:    tokenType,
			TokenVersion: user.TokenVersion,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
				IssuedAt:  jwt.NewNumericDate(now),
			},
		}
		return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	}

	accessToken, err := sign(middleware.TokenTypeAccess, accessTokenTTL)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := sign(middleware.TokenTypeRefresh, refreshTokenTTL)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// ChangePasswordRequest represents the change-password input
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword updates the authenticated user's password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "user_not_found", "User not found")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		apierr.Fail(c, http.StatusUnauthorized, "current_password_wrong", "The current password is not correct")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to hash password")
		return
	}

	// Revoke every other session of this user: the point of changing a password
	// you believe is compromised.
	if err := h.db.Model(&user).Updates(map[string]any{
		"password_hash": string(hashed),
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update password")
		return
	}

	// Keep the session that made the change alive with a fresh token pair.
	if err := h.db.First(&user, user.ID).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to reload user")
		return
	}
	token, err := h.issueSession(c, &user)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to generate tokens")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated", "token": token})
}

// ForgotPasswordRequest represents the forgot-password input
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword generates a reset token. The token is never returned in the
// response body (previous behavior leaked it to anyone who knew the email).
// Self-hosted admins without SMTP configured can read the token from server
// logs. As a dev-only convenience, setting DEV_EXPOSE_RESET_TOKEN=true will
// include the token in the JSON response — DO NOT enable in production.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Always respond with the same generic message to avoid user enumeration,
	// whether the email exists or not.
	genericResponse := gin.H{"message": "If the email exists, you will receive instructions"}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, genericResponse)
		return
	}

	// Generate a cryptographically secure 32-byte token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to generate token")
		return
	}
	token := hex.EncodeToString(tokenBytes)
	expires := time.Now().Add(1 * time.Hour)

	if err := h.db.Model(&user).Updates(map[string]any{
		"password_reset_token":   token,
		"password_reset_expires": expires,
	}).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to save reset token")
		return
	}

	// Emit the token to the server log so a self-hosted admin can relay it
	// out-of-band when SMTP is not configured. The banner lines make it easy
	// to grep. Once SMTP is wired up this log line should be removed.
	log.Printf("========================================================================")
	log.Printf("🔑 PASSWORD RESET TOKEN for %s (valid 1h)", user.Email)
	log.Printf("   %s", token)
	log.Printf("========================================================================")

	resp := gin.H{"message": genericResponse["message"]}
	if os.Getenv("DEV_EXPOSE_RESET_TOKEN") == "true" {
		resp["reset_token"] = token
		resp["expires_in"] = "1h"
		resp["warning"] = "DEV_EXPOSE_RESET_TOKEN is enabled — do not use in production"
	}
	c.JSON(http.StatusOK, resp)
}

// ResetPasswordRequest represents the reset-password input
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ResetPassword validates the token and updates the password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var user models.User
	if err := h.db.Where("password_reset_token = ?", req.Token).First(&user).Error; err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_or_expired_token", "Invalid or expired token")
		return
	}

	if user.PasswordResetExpires == nil || time.Now().After(*user.PasswordResetExpires) {
		apierr.Fail(c, http.StatusBadRequest, "reset_token_expired", "This link has expired. Request a new password reset.")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to hash password")
		return
	}

	if err := h.db.Model(&user).Updates(map[string]any{
		"password_hash":          string(hashed),
		"password_reset_token":   "",
		"password_reset_expires": nil,
		// A reset recovers a compromised account: end the sessions it had.
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to reset password")
		return
	}

	log.Printf("Password reset successfully for %s", user.Email)
	c.JSON(http.StatusOK, gin.H{"message": "Password reset. You can now sign in."})
}
