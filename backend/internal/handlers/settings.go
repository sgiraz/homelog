package handlers

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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
	Theme                     string                  `json:"theme"`
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

	var joinReq models.PropertyJoinRequest
	if err := h.db.Where("user_id = ? AND status = 'pending'", userID).
		Preload("Property").First(&joinReq).Error; err == nil {
		pendingReq = &PendingJoinRequestInfo{
			PropertyName: joinReq.Property.Name,
			RequestedAt:  joinReq.CreatedAt,
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

	updateRetentionDays := settings.NotificationRetentionDays
	if updateRetentionDays == 0 {
		updateRetentionDays = 90
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
		NotificationRetentionDays: updateRetentionDays,
		NotifyJoinRequests:        settings.NotifyJoinRequests,
		NotifySharedExpenses:      settings.NotifySharedExpenses,
		OnboardingCompleted:       settings.OnboardingCompleted,
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
				tx.Model(&member).Updates(map[string]interface{}{
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
