package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// HouseholdSettingsResponse is the response for household settings
type HouseholdSettingsResponse struct {
	SplitMode bool `json:"split_mode"`
}

// UpdateHouseholdSettingsRequest is the request for updating household settings
type UpdateHouseholdSettingsRequest struct {
	SplitMode bool `json:"split_mode"`
}

// GetHouseholdSettings - GET /api/v1/properties/:id/settings
func (h *SettingsHandler) GetHouseholdSettings(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse property ID
	propertyIDStr := c.Param("id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	if !requirePropertyMember(c, h.db, userID, uint(propertyID)) {
		return
	}

	var settings models.HouseholdSettings
	err = h.db.Where("property_id = ?", propertyID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default settings for this property
			settings = models.HouseholdSettings{
				PropertyID: uint(propertyID),
				SplitMode:  false,
			}
			if createErr := h.db.Create(&settings).Error; createErr != nil {
				// If creation fails, just return defaults without persisting
				c.JSON(http.StatusOK, HouseholdSettingsResponse{
					SplitMode: false,
				})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
			return
		}
	}

	c.JSON(http.StatusOK, HouseholdSettingsResponse{
		SplitMode: settings.SplitMode,
	})
}

// UpdateHouseholdSettings - PUT /api/v1/properties/:id/settings
func (h *SettingsHandler) UpdateHouseholdSettings(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse property ID
	propertyIDStr := c.Param("id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	if !requirePropertyAdmin(c, h.db, userID, uint(propertyID)) {
		return
	}

	// Parse request body
	var req UpdateHouseholdSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find or create household settings
	var settings models.HouseholdSettings
	err = h.db.Where("property_id = ?", propertyID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new settings
			settings = models.HouseholdSettings{
				PropertyID: uint(propertyID),
				SplitMode:  req.SplitMode,
			}
			if err := h.db.Create(&settings).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create settings"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
			return
		}
	} else {
		// Update existing settings
		settings.SplitMode = req.SplitMode
		if err := h.db.Save(&settings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
			return
		}
	}

	c.JSON(http.StatusOK, HouseholdSettingsResponse{
		SplitMode: settings.SplitMode,
	})
}
