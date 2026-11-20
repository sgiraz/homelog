package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// --- Account Deletion ---

// DeleteAccountCheckResponse is the pre-flight check for account deletion
type DeleteAccountCheckResponse struct {
	CanDelete bool `json:"can_delete"`
	// BlockingHouseholds are shared households that need another admin first.
	BlockingHouseholds []BlockingHouseholdInfo `json:"blocking_households,omitempty"`
	// OpenBalanceHouseholds are shared households where the user still owes or is owed.
	OpenBalanceHouseholds []string `json:"open_balance_households,omitempty"`
	// DataLossProperties are homes nobody else lives in: they go with the account.
	DataLossProperties []string `json:"data_loss_properties,omitempty"`
}

// BlockingHouseholdInfo describes a household where the user must nominate a new admin before deleting
type BlockingHouseholdInfo struct {
	HouseholdID   uint           `json:"household_id"`
	HouseholdName string         `json:"household_name"`
	Members       []MemberOption `json:"members"`
}

// MemberOption is a member eligible for admin promotion
type MemberOption struct {
	MemberID uint   `json:"member_id"`
	UserID   uint   `json:"user_id"`
	Name     string `json:"name"`
}

// DeleteAccountCheck - GET /api/v1/settings/account/delete-check
func (h *SettingsHandler) DeleteAccountCheck(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	check, err := accountDeleteCheck(h.db, userID)
	if err != nil {
		log.Printf("❌ Account delete check for user %d: %v", userID, err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to check the account")
		return
	}
	c.JSON(http.StatusOK, check)
}

// PromoteAdminRequest is the request body to promote a member to admin
type PromoteAdminRequest struct {
	HouseholdID uint `json:"household_id" binding:"required"`
	MemberID    uint `json:"member_id" binding:"required"`
}

// PromoteAdmin - POST /api/v1/settings/account/promote-admin
func (h *SettingsHandler) PromoteAdmin(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var req PromoteAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if !requireHouseholdAdmin(c, h.db, userID, req.HouseholdID) {
		return
	}

	// The new admin must be a registered member of the same household
	var target models.HouseholdMember
	if err := h.db.Where("id = ? AND household_id = ? AND is_virtual = false", req.MemberID, req.HouseholdID).
		First(&target).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "member_not_found", "Member not found")
		return
	}

	if err := h.db.Model(&target).Update("role", "admin").Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to promote admin")
		return
	}

	log.Printf("✅ Member ID=%d promoted to admin for household ID=%d by user ID=%d", req.MemberID, req.HouseholdID, userID)
	c.JSON(http.StatusOK, gin.H{"message": "Admin appointed"})
}

// DeleteAccountRequest is the request body for self-account deletion
type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

// DeleteAccount - DELETE /api/v1/settings/account
func (h *SettingsHandler) DeleteAccount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Verify password
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "user_not_found", "User not found")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		apierr.Fail(c, http.StatusUnauthorized, "password_wrong", "The password is not correct")
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error { return deleteAccount(tx, userID) })
	var blocked *accountBlockedError
	if errors.As(err, &blocked) {
		refuseAccountDelete(c, blocked.check)
		return
	}
	if err != nil {
		log.Printf("❌ Deleting account %d: %v", userID, err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete the account")
		return
	}
	removeAvatar(user.AvatarPath)

	log.Printf("✅ User ID=%d (%s) self-deleted their account", userID, user.Email)
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}

// removeAvatar deletes an avatar file once the account it belonged to is gone.
func removeAvatar(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(filepath.Join(database.DataDir(), path)); err != nil && !os.IsNotExist(err) {
		log.Printf("⚠️  Could not remove avatar %s: %v", path, err)
	}
}
