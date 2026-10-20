package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type DeviceCategoryHandler struct {
	db *gorm.DB
}

func NewDeviceCategoryHandler(db *gorm.DB) *DeviceCategoryHandler {
	return &DeviceCategoryHandler{db: db}
}

// List - GET /api/v1/device-categories
func (h *DeviceCategoryHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var categories []models.DeviceCategory

	if err := h.db.
		Where("user_id IS NULL OR user_id = ?", userID).
		Order("name ASC").
		Find(&categories).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch device categories")
		return
	}

	c.JSON(http.StatusOK, categories)
}

// Create - POST /api/v1/device-categories
func (h *DeviceCategoryHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var category models.DeviceCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid device category data")
		return
	}

	if category.Name == "" {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Category name is required")
		return
	}

	category.ID = 0
	category.UserID = &userID
	category.IsDefault = false

	if err := h.db.Create(&category).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create device category")
		return
	}

	c.JSON(http.StatusCreated, category)
}

// Update - PUT /api/v1/device-categories/:id
func (h *DeviceCategoryHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var category models.DeviceCategory
	if err := h.db.First(&category, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Device category not found")
			return
		}

		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch device category")
		return
	}

	if category.UserID == nil || *category.UserID != userID {
		apierr.Fail(c, http.StatusForbidden, "invalid_request", "You cannot modify this device category")
		return
	}

	var input models.DeviceCategory
	if err := c.ShouldBindJSON(&input); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid device category data")
		return
	}

	if input.Name == "" {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Category name is required")
		return
	}

	category.Name = input.Name
	category.Icon = input.Icon

	if err := h.db.Save(&category).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update device category")
		return
	}

	c.JSON(http.StatusOK, category)
}

// Delete - DELETE /api/v1/device-categories/:id
func (h *DeviceCategoryHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var category models.DeviceCategory
	if err := h.db.First(&category, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Device category not found")
			return
		}

		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch device category")
		return
	}

	if category.UserID == nil || *category.UserID != userID {
		apierr.Fail(c, http.StatusForbidden, "invalid_request", "You cannot delete this device category")
		return
	}

	var deviceCount int64
	if err := h.db.Model(&models.Device{}).
		Where("category_id = ?", category.ID).
		Count(&deviceCount).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to check device category usage")
		return
	}

	if deviceCount > 0 {
		apierr.Fail(c, http.StatusConflict, "category_in_use", "Device category is still in use")
		return
	}

	if err := h.db.Delete(&category).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete device category")
		return
	}

	c.Status(http.StatusNoContent)
}
