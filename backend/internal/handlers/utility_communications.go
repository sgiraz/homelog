package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// GetCommunications returns all communications for a utility.
func (h *UtilityHandler) GetCommunications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_utility_id", "Invalid service id")
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", utilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "utility_not_found", "Service not found")
		return
	}

	var comms []models.ServiceCommunication
	h.db.Where("utility_id = ?", utilityID).Order("created_at DESC").Find(&comms)

	c.JSON(http.StatusOK, comms)
}

// AddCommunication creates a new communication for a utility.
func (h *UtilityHandler) AddCommunication(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_utility_id", "Invalid service id")
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", utilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "utility_not_found", "Service not found")
		return
	}

	var input struct {
		BillID         *uint      `json:"bill_id"`
		Type           string     `json:"type" binding:"required,oneof=price_change contract_modification info privacy"`
		Title          string     `json:"title" binding:"required"`
		Content        string     `json:"content"`
		ActionDeadline *time.Time `json:"action_deadline"`
		IsImportant    bool       `json:"is_important"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	comm := models.ServiceCommunication{
		UtilityID:      uint(utilityID),
		BillID:         input.BillID,
		Type:           input.Type,
		Title:          input.Title,
		Content:        input.Content,
		ActionDeadline: input.ActionDeadline,
		IsImportant:    input.IsImportant,
	}

	if err := h.db.Create(&comm).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create communication")
		return
	}

	c.JSON(http.StatusCreated, comm)
}

// MarkCommunicationRead marks a communication as read.
func (h *UtilityHandler) MarkCommunicationRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_utility_id", "Invalid service id")
		return
	}

	commID, err := strconv.ParseUint(c.Param("commId"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_communication_id", "Invalid communication id")
		return
	}

	var comm models.ServiceCommunication
	if err := h.db.First(&comm, commID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "communication_not_found", "Communication not found")
		return
	}

	if comm.UtilityID != uint(utilityID) {
		apierr.Fail(c, http.StatusBadRequest, "communication_not_in_utility", "This communication belongs to another service")
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", comm.UtilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		apierr.Fail(c, http.StatusForbidden, "not_authorized", "You are not authorized to do this")
		return
	}

	if err := h.db.Model(&models.ServiceCommunication{}).
		Where("id = ?", comm.ID).
		Update("is_read", true).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to mark as read")
		return
	}
	comm.IsRead = true

	c.JSON(http.StatusOK, comm)
}

// DeleteCommunication deletes a communication.
func (h *UtilityHandler) DeleteCommunication(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_utility_id", "Invalid service id")
		return
	}

	commID, err := strconv.ParseUint(c.Param("commId"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_communication_id", "Invalid communication id")
		return
	}

	var comm models.ServiceCommunication
	if err := h.db.First(&comm, commID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "communication_not_found", "Communication not found")
		return
	}

	if comm.UtilityID != uint(utilityID) {
		apierr.Fail(c, http.StatusBadRequest, "communication_not_in_utility", "This communication belongs to another service")
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", comm.UtilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		apierr.Fail(c, http.StatusForbidden, "not_authorized", "You are not authorized to do this")
		return
	}

	h.db.Unscoped().Delete(&comm)
	c.JSON(http.StatusOK, gin.H{"message": "Communication deleted"})
}

// DeleteReadCommunications bulk-deletes all read communications for the user.
func (h *UtilityHandler) DeleteReadCommunications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utilityIDs []uint
	h.db.Model(&models.Utility{}).
		Where("property_id IN ?", memberPropertyIDs).
		Pluck("id", &utilityIDs)

	result := h.db.Unscoped().
		Where("utility_id IN ? AND is_read = ?", utilityIDs, true).
		Delete(&models.ServiceCommunication{})

	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

// GetAllCommunications returns all communications across the user's utilities.
// Auto-cleans communications older than the user's retention setting.
func (h *UtilityHandler) GetAllCommunications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utilityIDs []uint
	h.db.Model(&models.Utility{}).
		Where("property_id IN ?", memberPropertyIDs).
		Pluck("id", &utilityIDs)

	var settings models.UserSettings
	if err := h.db.Where("user_id = ?", userID).First(&settings).Error; err == nil {
		retentionDays := settings.NotificationRetentionDays
		if retentionDays == 0 {
			retentionDays = 90
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		h.db.Unscoped().
			Where("utility_id IN ? AND created_at < ?", utilityIDs, cutoff).
			Delete(&models.ServiceCommunication{})
	}

	query := h.db.Where("utility_id IN ?", utilityIDs).
		Preload("Utility").
		Order("created_at DESC")

	if c.Query("unread_only") == "true" {
		query = query.Where("is_read = ?", false)
	}

	if c.Query("read_only") == "true" {
		query = query.Where("is_read = ?", true)
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	query = query.Limit(limit)

	var comms []models.ServiceCommunication
	query.Find(&comms)

	c.JSON(http.StatusOK, comms)
}

// GetUnreadCount returns the count of unread communications.
func (h *UtilityHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utilityIDs []uint
	h.db.Model(&models.Utility{}).
		Where("property_id IN ?", memberPropertyIDs).
		Pluck("id", &utilityIDs)

	var count int64
	h.db.Model(&models.ServiceCommunication{}).
		Where("utility_id IN ? AND is_read = ?", utilityIDs, false).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"count": count})
}
