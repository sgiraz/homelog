package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type SettingsHandler struct {
	db *gorm.DB
}

func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

// UserSettingsResponse represents the user settings response
type UserSettingsResponse struct {
	Theme                     string `json:"theme"`
	Currency                  string `json:"currency"`
	Language                  string `json:"language"`
	DateFormat                string `json:"date_format"`
	DefaultSplitWithMemberIDs string `json:"default_split_with_member_ids"` // JSON array as string e.g. "[2,3]"
	DefaultTemplates          string `json:"default_templates"`             // JSON object as string e.g. {"electricity": 1}
	EmailNotifications        bool   `json:"email_notifications"`
	BillReminders             bool   `json:"bill_reminders"`
}

// UpdateUserSettingsRequest represents the request for updating user settings
type UpdateUserSettingsRequest struct {
	Theme                     *string `json:"theme,omitempty"`
	Currency                  *string `json:"currency,omitempty"`
	Language                  *string `json:"language,omitempty"`
	DateFormat                *string `json:"date_format,omitempty"`
	DefaultSplitWithMemberIDs *string `json:"default_split_with_member_ids,omitempty"` // JSON array as string
	DefaultTemplates          *string `json:"default_templates,omitempty"`             // JSON object as string
	EmailNotifications        *bool   `json:"email_notifications,omitempty"`
	BillReminders             *bool   `json:"bill_reminders,omitempty"`
}

// Get - GET /api/v1/settings
func (h *SettingsHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Find or create user settings
	var settings models.UserSettings
	err := h.db.Where("user_id = ?", userID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return defaults
			c.JSON(http.StatusOK, UserSettingsResponse{
				Theme:                     "auto",
				Currency:                  "EUR",
				Language:                  "it",
				DateFormat:                "DD/MM/YYYY",
				DefaultSplitWithMemberIDs: "",
				DefaultTemplates:          "",
				EmailNotifications:        true,
				BillReminders:             true,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}

	dateFormat := settings.DateFormat
	if dateFormat == "" {
		dateFormat = "DD/MM/YYYY"
	}

	c.JSON(http.StatusOK, UserSettingsResponse{
		Theme:                     settings.Theme,
		Currency:                  settings.Currency,
		Language:                  settings.Language,
		DateFormat:                dateFormat,
		DefaultSplitWithMemberIDs: settings.DefaultSplitWithMemberIDs,
		DefaultTemplates:          settings.DefaultTemplates,
		EmailNotifications:        settings.EmailNotifications,
		BillReminders:             settings.BillDueAlertDays > 0,
	})
}

// Update - PUT /api/v1/settings
func (h *SettingsHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req UpdateUserSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find or create user settings
	var settings models.UserSettings
	err := h.db.Where("user_id = ?", userID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new settings
			settings = models.UserSettings{
				UserID:                    userID,
				Theme:                     "auto",
				Currency:                  "EUR",
				Language:                  "it",
				DefaultSplitWithMemberIDs: "",
				EmailNotifications:        true,
				BillDueAlertDays:          3,
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
			return
		}
	}

	// Update fields if provided
	if req.Theme != nil {
		settings.Theme = *req.Theme
	}
	if req.Currency != nil {
		settings.Currency = *req.Currency
	}
	if req.Language != nil {
		settings.Language = *req.Language
	}
	if req.DateFormat != nil {
		settings.DateFormat = *req.DateFormat
	}
	if req.DefaultSplitWithMemberIDs != nil {
		settings.DefaultSplitWithMemberIDs = *req.DefaultSplitWithMemberIDs
	}
	if req.DefaultTemplates != nil {
		settings.DefaultTemplates = *req.DefaultTemplates
	}
	if req.EmailNotifications != nil {
		settings.EmailNotifications = *req.EmailNotifications
	}
	if req.BillReminders != nil {
		if *req.BillReminders {
			settings.BillDueAlertDays = 3
		} else {
			settings.BillDueAlertDays = 0
		}
	}

	// Save settings
	if settings.ID == 0 {
		if err := h.db.Create(&settings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create settings"})
			return
		}
	} else {
		if err := h.db.Save(&settings).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
			return
		}
	}

	updateDateFormat := settings.DateFormat
	if updateDateFormat == "" {
		updateDateFormat = "DD/MM/YYYY"
	}

	c.JSON(http.StatusOK, UserSettingsResponse{
		Theme:                     settings.Theme,
		Currency:                  settings.Currency,
		Language:                  settings.Language,
		DateFormat:                updateDateFormat,
		DefaultSplitWithMemberIDs: settings.DefaultSplitWithMemberIDs,
		DefaultTemplates:          settings.DefaultTemplates,
		EmailNotifications:        settings.EmailNotifications,
		BillReminders:             settings.BillDueAlertDays > 0,
	})
}

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
	_, exists := middleware.GetUserID(c)
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

	// Get household settings for this property (no ownership check - any authenticated user can access)
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
	_, exists := middleware.GetUserID(c)
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

	// Parse request body
	var req UpdateHouseholdSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find or create household settings (no ownership check - any authenticated user can update)
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
