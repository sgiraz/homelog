package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type NotificationHandler struct {
	db *gorm.DB
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

// createNotification creates a notification respecting user preferences
func createNotification(db *gorm.DB, userID uint, notifType, title, content string, relatedID *uint, propertyID *uint) {
	var settings models.UserSettings
	db.Where("user_id = ?", userID).First(&settings)

	switch notifType {
	case "join_request":
		if !settings.NotifyJoinRequests {
			return
		}
	case "expense_shared":
		if !settings.NotifySharedExpenses {
			return
		}
	}

	notif := models.Notification{
		UserID:     userID,
		Type:       notifType,
		Title:      title,
		Content:    content,
		RelatedID:  relatedID,
		PropertyID: propertyID,
	}
	if err := db.Create(&notif).Error; err != nil {
		log.Printf("❌ Failed to create notification for user %d: %v", userID, err)
	}
}

// List - GET /api/v1/notifications
func (h *NotificationHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	limit := 100
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}

	var notifications []models.Notification
	h.db.Where("user_id = ?", userID).
		Preload("Property").
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications)

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadCount - GET /api/v1/notifications/unread-count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var count int64
	h.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkRead - PATCH /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	result := h.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

// MarkAllRead - POST /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	h.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true)

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

// Delete - DELETE /api/v1/notifications/:id
func (h *NotificationHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	result := h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Notification{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted"})
}

// DeleteAllRead - DELETE /api/v1/notifications/read
func (h *NotificationHandler) DeleteAllRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	h.db.Where("user_id = ? AND is_read = ?", userID, true).Delete(&models.Notification{})

	c.JSON(http.StatusOK, gin.H{"message": "Read notifications deleted"})
}
