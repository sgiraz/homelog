package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// --- Account Deletion ---

// DeleteAccountCheckResponse is the pre-flight check for account deletion
type DeleteAccountCheckResponse struct {
	CanDelete          bool                   `json:"can_delete"`
	BlockingProperties []BlockingPropertyInfo `json:"blocking_properties,omitempty"`
	DataLossProperties []string               `json:"data_loss_properties,omitempty"`
}

// BlockingPropertyInfo describes a property where the user must nominate a new admin before deleting
type BlockingPropertyInfo struct {
	PropertyID   uint           `json:"property_id"`
	PropertyName string         `json:"property_name"`
	Members      []MemberOption `json:"members"`
}

// MemberOption is a member eligible for admin promotion
type MemberOption struct {
	MemberID uint   `json:"member_id"`
	UserID   uint   `json:"user_id"`
	Name     string `json:"name"`
}

// computeDeleteCheck returns the deletion pre-flight status for a user
func (h *SettingsHandler) computeDeleteCheck(userID uint) DeleteAccountCheckResponse {
	resp := DeleteAccountCheckResponse{CanDelete: true}

	// Find properties where user is admin (via HouseholdMember role)
	var adminMembers []models.HouseholdMember
	h.db.Where("user_id = ? AND role = 'admin'", userID).Preload("Property").Find(&adminMembers)

	for _, am := range adminMembers {
		// Count other admins in this property
		var otherAdminCount int64
		h.db.Model(&models.HouseholdMember{}).
			Where("property_id = ? AND role = 'admin' AND user_id != ? AND is_virtual = false", am.PropertyID, userID).
			Count(&otherAdminCount)

		if otherAdminCount > 0 {
			continue // other admins exist, no problem
		}

		// No other admin — check if there are other (non-virtual) members
		var otherMembers []models.HouseholdMember
		h.db.Where("property_id = ? AND user_id != ? AND is_virtual = false", am.PropertyID, userID).
			Find(&otherMembers)

		if len(otherMembers) > 0 {
			// Blocking: must nominate a new admin
			resp.CanDelete = false
			members := make([]MemberOption, len(otherMembers))
			for i, m := range otherMembers {
				uid := uint(0)
				if m.UserID != nil {
					uid = *m.UserID
				}
				members[i] = MemberOption{
					MemberID: m.ID,
					UserID:   uid,
					Name:     m.Name,
				}
			}
			resp.BlockingProperties = append(resp.BlockingProperties, BlockingPropertyInfo{
				PropertyID:   am.PropertyID,
				PropertyName: am.Property.Name,
				Members:      members,
			})
		} else {
			// Sole member — data loss warning
			resp.DataLossProperties = append(resp.DataLossProperties, am.Property.Name)
		}
	}

	return resp
}

// DeleteAccountCheck - GET /api/v1/settings/account/delete-check
func (h *SettingsHandler) DeleteAccountCheck(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	c.JSON(http.StatusOK, h.computeDeleteCheck(userID))
}

// PromoteAdminRequest is the request body to promote a member to admin
type PromoteAdminRequest struct {
	PropertyID uint `json:"property_id" binding:"required"`
	MemberID   uint `json:"member_id" binding:"required"`
}

// PromoteAdmin - POST /api/v1/settings/account/promote-admin
func (h *SettingsHandler) PromoteAdmin(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req PromoteAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify current user is admin of the property
	var adminCount int64
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND property_id = ? AND role = 'admin'", userID, req.PropertyID).
		Count(&adminCount)
	if adminCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Non sei admin di questa proprietà"})
		return
	}

	// Verify target member exists in the property and is not virtual
	var target models.HouseholdMember
	if err := h.db.Where("id = ? AND property_id = ? AND is_virtual = false", req.MemberID, req.PropertyID).
		First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Membro non trovato"})
		return
	}

	if err := h.db.Model(&target).Update("role", "admin").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Errore promozione admin"})
		return
	}

	log.Printf("✅ Member ID=%d promoted to admin for property ID=%d by user ID=%d", req.MemberID, req.PropertyID, userID)
	c.JSON(http.StatusOK, gin.H{"message": "Admin nominato con successo"})
}

// DeleteAccountRequest is the request body for self-account deletion
type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

// DeleteAccount - DELETE /api/v1/settings/account
func (h *SettingsHandler) DeleteAccount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify password
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utente non trovato"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password non corretta"})
		return
	}

	// Re-run delete check
	check := h.computeDeleteCheck(userID)
	if !check.CanDelete {
		c.JSON(http.StatusConflict, gin.H{
			"error":               "Devi nominare un admin per ogni proprietà prima di eliminare l'account",
			"blocking_properties": check.BlockingProperties,
		})
		return
	}

	// Begin transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Errore interno"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Collect user's household member IDs (across all properties)
	var userMemberIDs []uint
	tx.Model(&models.HouseholdMember{}).Where("user_id = ?", userID).Pluck("id", &userMemberIDs)

	// Collect user's household members with property info
	var userMembers []models.HouseholdMember
	tx.Where("user_id = ?", userID).Find(&userMembers)

	// For each property the user belongs to, decide: full cascade or just leave
	for _, member := range userMembers {
		// Count other non-virtual members in this property
		var otherMemberCount int64
		tx.Model(&models.HouseholdMember{}).
			Where("property_id = ? AND id != ? AND is_virtual = false", member.PropertyID, member.ID).
			Count(&otherMemberCount)

		if otherMemberCount == 0 {
			// Sole member: cascade delete entire property and all data
			if err := h.cascadeDeleteProperty(tx, member.PropertyID); err != nil {
				tx.Rollback()
				log.Printf("❌ Error cascade deleting property %d: %v", member.PropertyID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Errore eliminazione proprietà"})
				return
			}
			// Delete the member record itself
			tx.Unscoped().Delete(&models.HouseholdMember{}, member.ID)
		} else {
			// Other members exist: check if this member has associated expenses/splits
			var splitCount int64
			tx.Model(&models.ExpenseSplit{}).Where("member_id = ?", member.ID).Count(&splitCount)

			var paidExpenseCount int64
			tx.Model(&models.Expense{}).Where("paid_by_member_id = ?", member.ID).Count(&paidExpenseCount)

			if splitCount > 0 || paidExpenseCount > 0 {
				// Convert to virtual member to preserve expense history
				tx.Model(&member).Updates(map[string]any{
					"is_virtual": true,
					"user_id":    nil,
				})
			} else {
				// No expense references, safe to delete
				tx.Unscoped().Delete(&models.HouseholdMember{}, member.ID)
			}

			// Decrement residents
			tx.Model(&models.Property{}).Where("id = ?", member.PropertyID).
				UpdateColumn("residents", gorm.Expr("CASE WHEN residents > 1 THEN residents - 1 ELSE 1 END"))

			// Clean up split defaults referencing this member
			cleanupSplitDefaults(tx, member.ID)
		}
	}

	// Handle user's expenses:
	// - Shared expenses (with splits involving other members): keep expense, remove user's splits only
	// - Solo expenses (no other splits): delete entirely
	var userExpenseIDs []uint
	tx.Model(&models.Expense{}).Where("user_id = ?", userID).Pluck("id", &userExpenseIDs)

	for _, expID := range userExpenseIDs {
		// Check if this expense has splits from OTHER members (not user's members)
		var otherSplitCount int64
		if len(userMemberIDs) > 0 {
			tx.Model(&models.ExpenseSplit{}).
				Where("expense_id = ? AND member_id NOT IN ?", expID, userMemberIDs).
				Count(&otherSplitCount)
		}

		if otherSplitCount > 0 {
			// Shared expense: keep it, only remove user's splits
			if len(userMemberIDs) > 0 {
				tx.Unscoped().Where("expense_id = ? AND member_id IN ?", expID, userMemberIDs).
					Delete(&models.ExpenseSplit{})
			}
			// Leave the expense record as-is (user_id stays, PaidBy member is now virtual)
		} else {
			// Solo expense: delete splits and expense
			tx.Unscoped().Where("expense_id = ?", expID).Delete(&models.ExpenseSplit{})
			tx.Unscoped().Delete(&models.Expense{}, expID)
		}
	}

	// Delete remaining splits where user's members are involved but expense belongs to someone else
	if len(userMemberIDs) > 0 {
		tx.Unscoped().Where("member_id IN ?", userMemberIDs).Delete(&models.ExpenseSplit{})
	}

	// Delete PropertyJoinRequests
	tx.Unscoped().Where("user_id = ?", userID).Delete(&models.PropertyJoinRequest{})

	// Delete project_members join table
	tx.Exec("DELETE FROM project_members WHERE user_id = ?", userID)

	// Delete owned projects
	tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Project{})

	// Delete custom categories and subcategories
	var categoryIDs []uint
	tx.Model(&models.Category{}).Where("user_id = ?", userID).Pluck("id", &categoryIDs)
	if len(categoryIDs) > 0 {
		tx.Unscoped().Where("category_id IN ?", categoryIDs).Delete(&models.Subcategory{})
	}
	tx.Unscoped().Where("user_id = ?", userID).Delete(&models.Category{})

	// Delete BillTemplates
	tx.Unscoped().Where("user_id = ?", userID).Delete(&models.BillTemplate{})

	// Delete ContractTemplates
	tx.Unscoped().Where("user_id = ?", userID).Delete(&models.ContractTemplate{})

	// Delete ExpenseTemplates
	tx.Unscoped().Where("user_id = ?", userID).Delete(&models.ExpenseTemplate{})

	// Delete UserSettings
	tx.Unscoped().Where("user_id = ?", userID).Delete(&models.UserSettings{})

	// Delete avatar file
	if user.AvatarPath != "" {
		os.Remove(filepath.Join(dataDir(), user.AvatarPath))
	}

	// Delete User
	if err := tx.Unscoped().Delete(&models.User{}, userID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Errore eliminazione utente"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Errore commit transazione"})
		return
	}

	log.Printf("✅ User ID=%d (%s) self-deleted their account", userID, user.Email)
	c.JSON(http.StatusOK, gin.H{"message": "Account eliminato con successo"})
}

// cascadeDeleteProperty deletes all data associated with a property (utilities, bills, readings, etc.)
func (h *SettingsHandler) cascadeDeleteProperty(tx *gorm.DB, propertyID uint) error {
	// Collect utility IDs
	var utilityIDs []uint
	tx.Model(&models.Utility{}).Where("property_id = ?", propertyID).Pluck("id", &utilityIDs)

	if len(utilityIDs) > 0 {
		// PriceChanges
		tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.PriceChange{})
		// ServiceCommunications
		tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.ServiceCommunication{})
		// Bills
		tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.Bill{})
		// MeterReadings
		tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.MeterReading{})
		// UtilityRates
		tx.Unscoped().Where("utility_id IN ?", utilityIDs).Delete(&models.UtilityRate{})
		// Utilities
		tx.Unscoped().Where("property_id = ?", propertyID).Delete(&models.Utility{})
	}

	// Delete expenses associated with this property (and their splits)
	var propertyExpenseIDs []uint
	tx.Model(&models.Expense{}).Where("property_id = ?", propertyID).Pluck("id", &propertyExpenseIDs)
	if len(propertyExpenseIDs) > 0 {
		tx.Unscoped().Where("expense_id IN ?", propertyExpenseIDs).Delete(&models.ExpenseSplit{})
		tx.Unscoped().Where("property_id = ?", propertyID).Delete(&models.Expense{})
	}

	// Settlements
	tx.Unscoped().Where("property_id = ?", propertyID).Delete(&models.Settlement{})
	// HouseholdSettings
	tx.Unscoped().Where("property_id = ?", propertyID).Delete(&models.HouseholdSettings{})
	// Join requests
	tx.Unscoped().Where("property_id = ?", propertyID).Delete(&models.PropertyJoinRequest{})
	// Household members (remaining)
	tx.Unscoped().Where("property_id = ?", propertyID).Delete(&models.HouseholdMember{})
	// Property
	if err := tx.Unscoped().Delete(&models.Property{}, propertyID).Error; err != nil {
		return err
	}

	return nil
}
