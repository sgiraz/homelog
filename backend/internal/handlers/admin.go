package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

// AdminHandler handles admin-only operations
type AdminHandler struct {
	db *gorm.DB
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// DeleteUser deletes another user's account, under the same rules as deleting
// one's own (see deleteAccount).
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	adminUserID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	parsed, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_user_id", "Invalid user id")
		return
	}
	targetID := uint(parsed)
	if adminUserID == targetID {
		apierr.Fail(c, http.StatusBadRequest, "cannot_delete_own_account", "You cannot delete your own account from here")
		return
	}

	var target models.User
	if err := h.db.Unscoped().First(&target, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			apierr.Fail(c, http.StatusNotFound, "user_not_found", "User not found")
			return
		}
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch user")
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error { return deleteAccount(tx, targetID) })
	var blocked *accountBlockedError
	if errors.As(err, &blocked) {
		refuseAccountDelete(c, blocked.check)
		return
	}
	if err != nil {
		log.Printf("❌ Admin %d deleting user %d: %v", adminUserID, targetID, err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete the user")
		return
	}
	removeAvatar(target.AvatarPath)

	log.Printf("ADMIN: User %d deleted user %d (%s)", adminUserID, targetID, target.Email)
	c.JSON(http.StatusOK, gin.H{"message": "Account deleted"})
}

// ToggleAdmin changes a user's role between admin and user
func (h *AdminHandler) ToggleAdmin(c *gin.Context) {
	adminUserID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_user_id", "Invalid user id")
		return
	}

	if adminUserID == uint(targetUserID) {
		apierr.Fail(c, http.StatusBadRequest, "cannot_change_own_role", "You cannot change your own role")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin user"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_role", "Invalid role: use 'admin' or 'user'")
		return
	}

	var targetUser models.User
	if err := h.db.First(&targetUser, targetUserID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "user_not_found", "User not found")
		return
	}

	if err := h.db.Model(&targetUser).Updates(map[string]any{
		"role": req.Role,
		// Revoke their sessions: the old role travels in tokens already issued.
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update role")
		return
	}

	log.Printf("ADMIN: User %d changed role of user %d (%s) to %s", adminUserID, targetUser.ID, targetUser.Email, req.Role)
	c.JSON(http.StatusOK, gin.H{"message": "Role updated", "role": req.Role})
}
