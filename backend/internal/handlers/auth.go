package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type AuthHandler struct {
	db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// RegisterRequest represents registration input
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
}

// LoginRequest represents login input
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse represents authentication response
type TokenResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	User         models.User `json:"user"`
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"hint":  "Password must be at least 6 characters",
		})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
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

	// Start transaction
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	log.Printf("✅ User created: ID=%d, Email=%s, Role=%s", user.ID, user.Email, user.Role)

	// Auto-create Property + HouseholdSettings + UserSettings for first user (admin)
	if isFirstUser {
		// Create default property
		now := time.Now()
		property := models.Property{
			UserID:    user.ID,
			Name:      "Casa Principale",
			Address:   "",
			Type:      "owned",
			StartDate: now,
			IsCurrent: true,
			Residents: 1,
		}

		if err := tx.Create(&property).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating default property: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default property"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create household settings"})
			return
		}

		log.Printf("✅ Household settings created for property ID=%d", property.ID)

		// Create user settings
		userSettings := models.UserSettings{
			UserID:                    user.ID,
			Language:                  "it",
			Currency:                  "EUR",
			Theme:                     "auto",
			DateFormat:                "DD/MM/YYYY",
			DefaultSplitWithMemberIDs: "",
			EmailNotifications:        true,
			BillDueAlertDays:          3,
		}

		if err := tx.Create(&userSettings).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating user settings: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user settings"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create household member"})
			return
		}

		log.Printf("✅ Household member created: ID=%d, Name=%s, UserID=%d", adminMember.ID, adminMember.Name, user.ID)

		// Seed default categories for first user
		if err := database.SeedDefaultCategories(tx); err != nil {
			tx.Rollback()
			log.Printf("ERROR seeding default categories: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to seed default categories"})
			return
		}
		log.Printf("✅ Default categories seeded")
	} else {
		// Non-first user: join the existing property
		// Find the first property in the system (created by admin)
		var existingProperty models.Property
		if err := tx.First(&existingProperty).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR finding existing property: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No property found to join"})
			return
		}

		log.Printf("New user joining property: ID=%d, Name=%s", existingProperty.ID, existingProperty.Name)

		// Create user settings
		userSettings := models.UserSettings{
			UserID:                    user.ID,
			Language:                  "it",
			Currency:                  "EUR",
			Theme:                     "auto",
			DateFormat:                "DD/MM/YYYY",
			DefaultSplitWithMemberIDs: "",
			EmailNotifications:        true,
			BillDueAlertDays:          3,
		}

		if err := tx.Create(&userSettings).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating user settings: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user settings"})
			return
		}

		log.Printf("User settings created for user ID=%d", user.ID)

		// Create HouseholdMember for the new user (linked to their User account)
		newMember := models.HouseholdMember{
			PropertyID: existingProperty.ID,
			UserID:     &user.ID,
			Name:       user.Name,
			Role:       "member",
			IsVirtual:  false,
		}

		if err := tx.Create(&newMember).Error; err != nil {
			tx.Rollback()
			log.Printf("ERROR creating household member: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create household member"})
			return
		}

		log.Printf("Household member created: ID=%d, Name=%s, UserID=%d, PropertyID=%d",
			newMember.ID, newMember.Name, user.ID, existingProperty.ID)

		// Update property residents count
		tx.Model(&existingProperty).Update("residents", gorm.Expr("residents + 1"))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete registration"})
		return
	}

	// Generate tokens
	token, refreshToken, err := h.generateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusCreated, TokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	})
}

// Login authenticates a user
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is inactive"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate tokens
	token, refreshToken, err := h.generateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	})
}

// RefreshToken generates new access token from refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse refresh token
	jwtSecret := os.Getenv("JWT_SECRET")

	token, err := jwt.ParseWithClaims(req.RefreshToken, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	claims, ok := token.Claims.(*middleware.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	// Get user
	var user models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate new tokens
	newToken, newRefreshToken, err := h.generateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		User:         user,
	})
}

// generateTokens creates access and refresh tokens
func (h *AuthHandler) generateTokens(user *models.User) (string, string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")

	// Access token (expires in 15 minutes)
	accessClaims := middleware.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	// Refresh token (expires in 7 days)
	refreshClaims := middleware.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

// ChangePasswordRequest represents the change-password input
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword updates the authenticated user's password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password attuale non corretta"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := h.db.Model(&user).Update("password_hash", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password aggiornata con successo"})
}

// ForgotPasswordRequest represents the forgot-password input
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword generates a reset token and returns it directly
// (no email configured — token is shown in the response for self-hosted use)
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Return success even if email not found to avoid enumeration
		c.JSON(http.StatusOK, gin.H{"message": "Se l'email esiste, riceverai le istruzioni."})
		return
	}

	// Generate a cryptographically secure 32-byte token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	token := hex.EncodeToString(tokenBytes)
	expires := time.Now().Add(1 * time.Hour)

	if err := h.db.Model(&user).Updates(map[string]interface{}{
		"password_reset_token":   token,
		"password_reset_expires": expires,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reset token"})
		return
	}

	log.Printf("Password reset token generated for %s", user.Email)

	// Since email is not configured, return the token directly
	c.JSON(http.StatusOK, gin.H{
		"message":     "Token generato. Poiché l'email non è configurata, usa il token qui sotto.",
		"reset_token": token,
		"expires_in":  "1 ora",
	})
}

// ResetPasswordRequest represents the reset-password input
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPassword validates the token and updates the password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("password_reset_token = ?", req.Token).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token non valido o scaduto"})
		return
	}

	if user.PasswordResetExpires == nil || time.Now().After(*user.PasswordResetExpires) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token scaduto. Richiedi un nuovo reset."})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := h.db.Model(&user).Updates(map[string]interface{}{
		"password_hash":          string(hashed),
		"password_reset_token":   "",
		"password_reset_expires": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	log.Printf("Password reset successfully for %s", user.Email)
	c.JSON(http.StatusOK, gin.H{"message": "Password reimpostata con successo. Puoi ora accedere."})
}
