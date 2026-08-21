package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// dataDir returns the base data directory, derived from DB_PATH for consistency
// across dev (WORKDIR /app/src) and prod (WORKDIR /app) environments.
func dataDir() string {
	dbPath := os.Getenv("DB_PATH")
	if dbPath != "" {
		return filepath.Dir(dbPath)
	}
	return "./data"
}

type SettingsHandler struct {
	db *gorm.DB
}

func NewSettingsHandler(db *gorm.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

// Allow-lists so unexpected client values never get persisted/echoed back.
const defaultColorTheme = "slate"

var validThemes = map[string]bool{"auto": true, "light": true, "dark": true}
var validColorThemes = map[string]bool{
	"slate": true, "paper": true, "forest": true, "ocean": true, "plum": true,
}

// colorThemeOrDefault coalesces an empty/unknown stored value to the default.
func colorThemeOrDefault(v string) string {
	if validColorThemes[v] {
		return v
	}
	return defaultColorTheme
}

// UserSettingsResponse represents the user settings response
type UserSettingsResponse struct {
	Theme                     string                  `json:"theme"`
	ColorTheme                string                  `json:"color_theme"`
	Currency                  string                  `json:"currency"`
	Language                  string                  `json:"language"`
	DateFormat                string                  `json:"date_format"`
	DefaultSplitWithMemberIDs string                  `json:"default_split_with_member_ids"` // JSON array as string e.g. "[2,3]"
	DefaultTemplates          string                  `json:"default_templates"`             // JSON object as string e.g. {"electricity": 1}
	EmailNotifications        bool                    `json:"email_notifications"`
	BillReminders             bool                    `json:"bill_reminders"`
	NotificationRetentionDays int                     `json:"notification_retention_days"`
	NotifyJoinRequests        bool                    `json:"notify_join_requests"`
	NotifySharedExpenses      bool                    `json:"notify_shared_expenses"`
	OnboardingCompleted       bool                    `json:"onboarding_completed"`
	HasProperty               bool                    `json:"has_property"`
	IsPropertyAdmin           bool                    `json:"is_property_admin"`
	PendingJoinRequest        *PendingJoinRequestInfo `json:"pending_join_request,omitempty"`
}

// PendingJoinRequestInfo contains summary info about a user's pending join request
type PendingJoinRequestInfo struct {
	PropertyName string    `json:"property_name"`
	RequestedAt  time.Time `json:"requested_at"`
}

// UpdateUserSettingsRequest represents the request for updating user settings
type UpdateUserSettingsRequest struct {
	Theme                     *string `json:"theme,omitempty"`
	ColorTheme                *string `json:"color_theme,omitempty"`
	Currency                  *string `json:"currency,omitempty"`
	Language                  *string `json:"language,omitempty"`
	DateFormat                *string `json:"date_format,omitempty"`
	DefaultSplitWithMemberIDs *string `json:"default_split_with_member_ids,omitempty"` // JSON array as string
	DefaultTemplates          *string `json:"default_templates,omitempty"`             // JSON object as string
	EmailNotifications        *bool   `json:"email_notifications,omitempty"`
	BillReminders             *bool   `json:"bill_reminders,omitempty"`
	NotificationRetentionDays *int    `json:"notification_retention_days,omitempty"`
	NotifyJoinRequests        *bool   `json:"notify_join_requests,omitempty"`
	NotifySharedExpenses      *bool   `json:"notify_shared_expenses,omitempty"`
	OnboardingCompleted       *bool   `json:"onboarding_completed,omitempty"`
}

// getPropertyStatus returns whether the user has a property membership, is a property admin, and any pending join request info.
func (h *SettingsHandler) getPropertyStatus(userID uint) (hasProperty bool, isPropertyAdmin bool, pendingReq *PendingJoinRequestInfo) {
	var memberCount int64
	h.db.Model(&models.HouseholdMember{}).Where("user_id = ?", userID).Count(&memberCount)
	hasProperty = memberCount > 0

	var adminCount int64
	h.db.Model(&models.HouseholdMember{}).Where("user_id = ? AND role = 'admin'", userID).Count(&adminCount)
	isPropertyAdmin = adminCount > 0

	var joinReqs []models.PropertyJoinRequest
	h.db.Where("user_id = ? AND status = 'pending'", userID).
		Preload("Property").Limit(1).Find(&joinReqs)
	if len(joinReqs) > 0 {
		pendingReq = &PendingJoinRequestInfo{
			PropertyName: joinReqs[0].Property.Name,
			RequestedAt:  joinReqs[0].CreatedAt,
		}
	}
	return
}

// Get - GET /api/v1/settings
func (h *SettingsHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	hasProperty, isPropertyAdmin, pendingReq := h.getPropertyStatus(userID)

	// Find or create user settings
	var settings models.UserSettings
	err := h.db.Where("user_id = ?", userID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return defaults
			c.JSON(http.StatusOK, UserSettingsResponse{
				Theme:                     "auto",
				ColorTheme:                defaultColorTheme,
				Currency:                  "EUR",
				Language:                  "it",
				DateFormat:                "DD/MM/YYYY",
				DefaultSplitWithMemberIDs: "",
				DefaultTemplates:          "",
				EmailNotifications:        true,
				BillReminders:             true,
				NotificationRetentionDays: 90,
				NotifyJoinRequests:        true,
				NotifySharedExpenses:      true,
				OnboardingCompleted:       false,
				HasProperty:               hasProperty,
				IsPropertyAdmin:           isPropertyAdmin,
				PendingJoinRequest:        pendingReq,
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

	retentionDays := settings.NotificationRetentionDays
	if retentionDays == 0 {
		retentionDays = 90
	}

	c.JSON(http.StatusOK, UserSettingsResponse{
		Theme:                     settings.Theme,
		ColorTheme:                colorThemeOrDefault(settings.ColorTheme),
		Currency:                  settings.Currency,
		Language:                  settings.Language,
		DateFormat:                dateFormat,
		DefaultSplitWithMemberIDs: settings.DefaultSplitWithMemberIDs,
		DefaultTemplates:          settings.DefaultTemplates,
		EmailNotifications:        settings.EmailNotifications,
		BillReminders:             settings.BillDueAlertDays > 0,
		NotificationRetentionDays: retentionDays,
		NotifyJoinRequests:        settings.NotifyJoinRequests,
		NotifySharedExpenses:      settings.NotifySharedExpenses,
		OnboardingCompleted:       settings.OnboardingCompleted,
		HasProperty:               hasProperty,
		IsPropertyAdmin:           isPropertyAdmin,
		PendingJoinRequest:        pendingReq,
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
				ColorTheme:                defaultColorTheme,
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

	// Update fields if provided (validated against allow-lists).
	if req.Theme != nil && validThemes[*req.Theme] {
		settings.Theme = *req.Theme
	}
	if req.ColorTheme != nil && validColorThemes[*req.ColorTheme] {
		settings.ColorTheme = *req.ColorTheme
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
	if req.NotificationRetentionDays != nil {
		settings.NotificationRetentionDays = *req.NotificationRetentionDays
	}
	if req.NotifyJoinRequests != nil {
		settings.NotifyJoinRequests = *req.NotifyJoinRequests
	}
	if req.NotifySharedExpenses != nil {
		settings.NotifySharedExpenses = *req.NotifySharedExpenses
	}
	if req.OnboardingCompleted != nil {
		settings.OnboardingCompleted = *req.OnboardingCompleted
	}

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

	updateRetentionDays := settings.NotificationRetentionDays
	if updateRetentionDays == 0 {
		updateRetentionDays = 90
	}

	c.JSON(http.StatusOK, UserSettingsResponse{
		Theme:                     settings.Theme,
		ColorTheme:                colorThemeOrDefault(settings.ColorTheme),
		Currency:                  settings.Currency,
		Language:                  settings.Language,
		DateFormat:                updateDateFormat,
		DefaultSplitWithMemberIDs: settings.DefaultSplitWithMemberIDs,
		DefaultTemplates:          settings.DefaultTemplates,
		EmailNotifications:        settings.EmailNotifications,
		BillReminders:             settings.BillDueAlertDays > 0,
		NotificationRetentionDays: updateRetentionDays,
		NotifyJoinRequests:        settings.NotifyJoinRequests,
		NotifySharedExpenses:      settings.NotifySharedExpenses,
		OnboardingCompleted:       settings.OnboardingCompleted,
	})
}
