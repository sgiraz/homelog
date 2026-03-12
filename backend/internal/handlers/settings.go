package handlers

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
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

// UploadAvatar - POST /api/v1/settings/avatar
func (h *SettingsHandler) UploadAvatar(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	// Validate file size (max 5MB)
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 5MB)"})
		return
	}

	// Validate content type by reading first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
		return
	}
	contentType := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be an image (JPEG, PNG, or WebP)"})
		return
	}

	// Reset reader to beginning
	if seeker, ok := file.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// Decode image
	src, _, err := image.Decode(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode image"})
		return
	}

	// Center-crop to square and resize to 256x256
	bounds := src.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	cropSize := imgW
	if imgH < imgW {
		cropSize = imgH
	}
	offsetX := (imgW - cropSize) / 2
	offsetY := (imgH - cropSize) / 2

	// Create cropped sub-image
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	var cropped image.Image
	if si, ok := src.(subImager); ok {
		cropped = si.SubImage(image.Rect(
			bounds.Min.X+offsetX, bounds.Min.Y+offsetY,
			bounds.Min.X+offsetX+cropSize, bounds.Min.Y+offsetY+cropSize,
		))
	} else {
		cropped = src
	}

	// Resize to 256x256
	dst := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)

	// Ensure avatars directory exists (derive from DB_PATH for consistent paths across dev/prod)
	avatarDir := filepath.Join(dataDir(), "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create avatar directory"})
		return
	}

	// Generate filename and save
	filename := fmt.Sprintf("%d_%d.jpg", userID, time.Now().UnixNano())
	avatarPath := filepath.Join(avatarDir, filename)

	outFile, err := os.Create(avatarPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save avatar"})
		return
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, dst, &jpeg.Options{Quality: 85}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode avatar"})
		return
	}

	// Delete old avatar if exists
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}
	if user.AvatarPath != "" {
		oldPath := filepath.Join(dataDir(), user.AvatarPath)
		os.Remove(oldPath)
	}

	// Update user in DB
	relativePath := "avatars/" + filename
	if err := h.db.Model(&user).Update("avatar_path", relativePath).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	user.AvatarPath = relativePath

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// DeleteAvatar - DELETE /api/v1/settings/avatar
func (h *SettingsHandler) DeleteAvatar(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	// Delete file from disk
	if user.AvatarPath != "" {
		oldPath := filepath.Join(dataDir(), user.AvatarPath)
		os.Remove(oldPath)
	}

	// Clear in DB
	if err := h.db.Model(&user).Update("avatar_path", "").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	user.AvatarPath = ""

	c.JSON(http.StatusOK, gin.H{"user": user})
}
