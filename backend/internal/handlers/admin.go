package handlers

import (
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

// DeleteUser permanently deletes a user account and all associated data
func (h *AdminHandler) DeleteUser(c *gin.Context) {
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
		apierr.Fail(c, http.StatusBadRequest, "cannot_delete_own_account", "You cannot delete your own account from here")
		return
	}

	// Load target user (including soft-deleted)
	var targetUser models.User
	userFound := true
	if err := h.db.Unscoped().First(&targetUser, targetUserID).Error; err != nil {
		// User record may have been deleted in a previous partial cleanup.
		// Check if orphaned HouseholdMember records exist for this user_id.
		var orphanCount int64
		h.db.Unscoped().Model(&models.HouseholdMember{}).Where("user_id = ?", targetUserID).Count(&orphanCount)
		if orphanCount == 0 {
			apierr.Fail(c, http.StatusNotFound, "user_not_found", "User not found")
			return
		}
		userFound = false
		targetUser.ID = uint(targetUserID)
	}

	// Begin transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Internal error")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	targetID := uint(targetUserID)

	// Step A: Collect property IDs
	var propertyIDs []uint
	tx.Unscoped().Model(&models.Property{}).Where("user_id = ?", targetID).Pluck("id", &propertyIDs)

	// Step B: Collect utility IDs
	var utilityIDs []uint
	if len(propertyIDs) > 0 {
		tx.Unscoped().Model(&models.Utility{}).Where("property_id IN ?", propertyIDs).Pluck("id", &utilityIDs)
	}

	// Step C: Collect household member IDs (owned properties + user's own member records)
	var memberIDs []uint
	if len(propertyIDs) > 0 {
		tx.Unscoped().Model(&models.HouseholdMember{}).Where("property_id IN ?", propertyIDs).Pluck("id", &memberIDs)
	}
	// Also collect member records directly linked to this user (e.g., in other users' properties)
	var userMemberIDs []uint
	tx.Unscoped().Model(&models.HouseholdMember{}).Where("user_id = ?", targetID).Pluck("id", &userMemberIDs)
	for _, id := range userMemberIDs {
		found := false
		for _, existing := range memberIDs {
			if id == existing {
				found = true
				break
			}
		}
		if !found {
			memberIDs = append(memberIDs, id)
		}
	}

	// Step D: Cascade delete in FK-safe order

	// 1. ExpenseSplits (by member_id OR by expense owner)
	if len(memberIDs) > 0 {
		if err := tx.Unscoped().Where("member_id IN ?", memberIDs).Delete(&models.ExpenseSplit{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete expense splits")
			return
		}
	}
	// Also delete splits on expenses owned by this user
	var userExpenseIDs []uint
	tx.Unscoped().Model(&models.Expense{}).Where("user_id = ?", targetID).Pluck("id", &userExpenseIDs)
	if len(userExpenseIDs) > 0 {
		if err := tx.Unscoped().Where("expense_id IN ?", userExpenseIDs).Delete(&models.ExpenseSplit{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete expense splits")
			return
		}
	}

	// 2. Settlements
	if len(propertyIDs) > 0 {
		if err := tx.Unscoped().Where("property_id IN ?", propertyIDs).Delete(&models.Settlement{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete settlements")
			return
		}
	}

	// 3. Expenses
	if err := tx.Unscoped().Where("user_id = ?", targetID).Delete(&models.Expense{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete expenses")
		return
	}

	// 4. Bills
	if len(utilityIDs) > 0 {
		if err := tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.Bill{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete bills")
			return
		}
	}

	// 5. MeterReadings
	if len(utilityIDs) > 0 {
		if err := tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.MeterReading{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete readings")
			return
		}
	}

	// 6. UtilityRates
	if len(utilityIDs) > 0 {
		if err := tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.UtilityRate{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete rates")
			return
		}
	}

	// 7. Utilities
	if len(propertyIDs) > 0 {
		if err := tx.Unscoped().Where("property_id IN ?", propertyIDs).Delete(&models.Utility{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete utilities")
			return
		}
	}

	// 8. HouseholdSettings
	if len(propertyIDs) > 0 {
		if err := tx.Unscoped().Where("property_id IN ?", propertyIDs).Delete(&models.HouseholdSettings{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete household settings")
			return
		}
	}

	// 9. HouseholdMembers (owned properties + user's own member records in other properties)
	if len(memberIDs) > 0 {
		// Clean up stale member IDs from other users' default split settings
		for _, mid := range memberIDs {
			cleanupSplitDefaults(tx, mid)
		}
		if err := tx.Unscoped().Where("id IN ?", memberIDs).Delete(&models.HouseholdMember{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete members")
			return
		}
	}

	// 10. Properties
	if err := tx.Unscoped().Where("user_id = ?", targetID).Delete(&models.Property{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete properties")
		return
	}

	// 11. project_members join table (no GORM model)
	if err := tx.Exec("DELETE FROM project_members WHERE user_id = ?", targetID).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete project associations")
		return
	}

	// 12. Projects
	if err := tx.Unscoped().Where("user_id = ?", targetID).Delete(&models.Project{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete projects")
		return
	}

	// 13. Subcategories (for user's custom categories)
	var categoryIDs []uint
	tx.Unscoped().Model(&models.Category{}).Where("user_id = ?", targetID).Pluck("id", &categoryIDs)
	if len(categoryIDs) > 0 {
		if err := tx.Unscoped().Where("category_id IN ?", categoryIDs).Delete(&models.Subcategory{}).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete subcategories")
			return
		}
	}

	// 14. Categories (user-created only)
	if err := tx.Unscoped().Where("user_id = ?", targetID).Delete(&models.Category{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete categories")
		return
	}

	// 15. BillTemplates
	if err := tx.Unscoped().Where("user_id = ?", targetID).Delete(&models.BillTemplate{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete bill templates")
		return
	}

	// 16. ContractTemplates
	if err := tx.Unscoped().Where("user_id = ?", targetID).Delete(&models.ContractTemplate{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete contract templates")
		return
	}

	// 17. UserSettings
	if err := tx.Unscoped().Where("user_id = ?", targetID).Delete(&models.UserSettings{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete user settings")
		return
	}

	// 18. User
	if userFound {
		if err := tx.Unscoped().Delete(&models.User{}, targetID).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete user")
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to commit transaction")
		return
	}

	log.Printf("ADMIN: User %d deleted user %d (%s)", adminUserID, targetUser.ID, targetUser.Email)
	c.JSON(http.StatusOK, gin.H{"message": "Account e tutti i dati associati eliminati con successo"})
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

	if err := h.db.Model(&targetUser).Update("role", req.Role).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update role")
		return
	}

	log.Printf("ADMIN: User %d changed role of user %d (%s) to %s", adminUserID, targetUser.ID, targetUser.Email, req.Role)
	c.JSON(http.StatusOK, gin.H{"message": "Ruolo aggiornato con successo", "role": req.Role})
}
