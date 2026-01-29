package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type PropertyHandler struct {
	db *gorm.DB
}

func NewPropertyHandler(db *gorm.DB) *PropertyHandler {
	return &PropertyHandler{db: db}
}

// CreatePropertyRequest represents the request body for creating a property
type CreatePropertyRequest struct {
	Name      string `json:"name" binding:"required"`
	Address   string `json:"address"`
	Type      string `json:"type"` // owned, rented
	IsCurrent bool   `json:"is_current"`
	Residents int    `json:"residents"`
}

// UpdatePropertyRequest represents the request body for updating a property
type UpdatePropertyRequest struct {
	Name      *string `json:"name"`
	Address   *string `json:"address"`
	Type      *string `json:"type"`
	IsCurrent *bool   `json:"is_current"`
	Residents *int    `json:"residents"`
}

// List - GET /api/v1/properties
// Returns properties where user is owner OR is a household member
func (h *PropertyHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Find property IDs where user is a household member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Get properties where user is owner OR is a member
	var properties []models.Property
	query := h.db.Where("user_id = ?", userID)
	if len(memberPropertyIDs) > 0 {
		query = h.db.Where("user_id = ? OR id IN ?", userID, memberPropertyIDs)
	}

	if err := query.Order("is_current DESC, created_at DESC").
		Find(&properties).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch properties"})
		return
	}

	log.Printf("User %d has access to %d properties (owner or member)", userID, len(properties))

	c.JSON(http.StatusOK, properties)
}

// Create - POST /api/v1/properties
func (h *PropertyHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreatePropertyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If this property is set as current, unset other properties
	if req.IsCurrent {
		h.db.Model(&models.Property{}).Where("user_id = ?", userID).Update("is_current", false)
	}

	residents := req.Residents
	if residents == 0 {
		residents = 1
	}

	property := models.Property{
		UserID:    userID,
		Name:      req.Name,
		Address:   req.Address,
		Type:      req.Type,
		StartDate: time.Now(),
		IsCurrent: req.IsCurrent,
		Residents: residents,
	}

	if err := h.db.Create(&property).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create property"})
		return
	}

	// Create default household settings for the property
	householdSettings := models.HouseholdSettings{
		PropertyID:       property.ID,
		SplitMode:        false,
		DefaultSplitType: "equal",
	}
	h.db.Create(&householdSettings)

	log.Printf("✅ Property created: ID=%d, Name=%s, UserID=%d", property.ID, property.Name, userID)

	c.JSON(http.StatusCreated, property)
}

// Get - GET /api/v1/properties/:id
func (h *PropertyHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	var property models.Property
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&property).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch property"})
		}
		return
	}

	c.JSON(http.StatusOK, property)
}

// Update - PUT /api/v1/properties/:id
func (h *PropertyHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	// Find existing property
	var property models.Property
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&property).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch property"})
		}
		return
	}

	var req UpdatePropertyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Residents != nil {
		updates["residents"] = *req.Residents
	}
	if req.IsCurrent != nil {
		// If setting as current, unset other properties
		if *req.IsCurrent {
			h.db.Model(&models.Property{}).Where("user_id = ? AND id != ?", userID, id).Update("is_current", false)
		}
		updates["is_current"] = *req.IsCurrent
	}

	if len(updates) > 0 {
		if err := h.db.Model(&property).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update property"})
			return
		}
	}

	// Reload property
	h.db.First(&property, id)

	c.JSON(http.StatusOK, property)
}

// Delete - DELETE /api/v1/properties/:id
func (h *PropertyHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	// Soft delete
	result := h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Property{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete property"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Property deleted successfully"})
}
