package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
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
	Name      *string `json:"name" binding:"omitempty,min=1"`
	Address   *string `json:"address"`
	Type      *string `json:"type" binding:"omitempty,oneof=owned rented"`
	IsCurrent *bool   `json:"is_current"`
	Residents *int    `json:"residents" binding:"omitempty,min=1"`
}

// List - GET /api/v1/properties
// Returns properties where user is owner OR is a household member
func (h *PropertyHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	visible, err := accessiblePropertyIDs(h.db, userID)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch properties")
		return
	}
	var properties []models.Property
	if err := h.db.Preload("Household").Where("id IN ?", visible).
		Order("is_current DESC, created_at DESC").
		Find(&properties).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch properties")
		return
	}

	ids := make([]uint, len(properties))
	for i, p := range properties {
		ids[i] = p.ID
	}
	shared, err := sharedPropertyIDs(h.db, ids)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to count members")
		return
	}
	for i := range properties {
		properties[i].IsShared = shared[properties[i].ID]
	}

	log.Printf("User %d has access to %d properties (owner or member)", userID, len(properties))

	c.JSON(http.StatusOK, properties)
}

// Create - POST /api/v1/properties
func (h *PropertyHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var req CreatePropertyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
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

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch user")
		return
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

	// Until a home can join an existing household, every new home starts its own.
	err := h.db.Transaction(func(tx *gorm.DB) error {
		household, _, err := createHousehold(tx, req.Name, user)
		if err != nil {
			return err
		}
		property.HouseholdID = household.ID
		return tx.Create(&property).Error
	})
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create property")
		return
	}

	log.Printf("✅ Property created: ID=%d, Name=%s, UserID=%d", property.ID, property.Name, userID)

	c.JSON(http.StatusCreated, property)
}

// Get - GET /api/v1/properties/:id
func (h *PropertyHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_property_id", "Invalid property id")
		return
	}

	var property models.Property
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&property).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "property_not_found", "Property not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch property")
		}
		return
	}

	c.JSON(http.StatusOK, property)
}

// Update - PUT /api/v1/properties/:id
func (h *PropertyHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_property_id", "Invalid property id")
		return
	}

	// Find existing property
	var property models.Property
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&property).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "property_not_found", "Property not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch property")
		}
		return
	}

	var req UpdatePropertyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	updates := make(map[string]any)

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "name must not be blank")
			return
		}
		updates["name"] = name
	}
	if req.Address != nil {
		updates["address"] = strings.TrimSpace(*req.Address)
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Residents != nil {
		updates["residents"] = *req.Residents
	}
	if req.IsCurrent != nil {
		updates["is_current"] = *req.IsCurrent
	}

	if len(updates) > 0 {
		err := h.db.Transaction(func(tx *gorm.DB) error {
			if req.IsCurrent != nil && *req.IsCurrent {
				if err := tx.Model(&models.Property{}).
					Where("user_id = ? AND id != ?", userID, property.ID).
					Update("is_current", false).Error; err != nil {
					return err
				}
			}
			return tx.Model(&property).Updates(updates).Error
		})
		if err != nil {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update property")
			return
		}
	}

	if err := h.db.First(&property, property.ID).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch property")
		return
	}

	c.JSON(http.StatusOK, property)
}

// Delete - DELETE /api/v1/properties/:id
func (h *PropertyHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_property_id", "Invalid property id")
		return
	}

	var property models.Property
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&property).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "property_not_found", "Property not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch property")
		}
		return
	}

	// A shared home holds other people's history, and for some of them it may
	// be their only home: only an owner left alone in it can delete it.
	shared, err := sharedPropertyIDs(h.db, []uint{property.ID})
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to count members")
		return
	}
	if shared[property.ID] {
		apierr.Fail(c, http.StatusConflict, "property_has_members", "A property shared with other members cannot be deleted")
		return
	}

	// The app has no screen without a home: the last one cannot go.
	visible, err := accessiblePropertyIDs(h.db, userID)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to count properties")
		return
	}
	var ownedOthers int64
	if err := h.db.Model(&models.Property{}).
		Where("user_id = ? AND id != ?", userID, property.ID).
		Count(&ownedOthers).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to count properties")
		return
	}
	if len(visible) <= 1 {
		apierr.Fail(c, http.StatusConflict, "last_property", "The only property cannot be deleted")
		return
	}

	// Deleting the primary home: the owner names its successor among their
	// other homes. With none left (only homes they joined), there is nothing
	// to promote.
	var successor *models.Property
	if property.IsCurrent && ownedOthers > 0 {
		raw := c.Query("new_principal_id")
		if raw == "" {
			apierr.Fail(c, http.StatusBadRequest, "new_principal_required", "new_principal_id is required when deleting the primary property")
			return
		}
		nextID, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || uint(nextID) == property.ID {
			apierr.Fail(c, http.StatusBadRequest, "new_principal_invalid", "new_principal_id must be another property you own")
			return
		}
		var next models.Property
		if err := h.db.Where("id = ? AND user_id = ?", nextID, userID).First(&next).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				apierr.Fail(c, http.StatusBadRequest, "new_principal_invalid", "new_principal_id must be another property you own")
			} else {
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch property")
			}
			return
		}
		successor = &next
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := cascadeDeleteProperty(tx, property.ID); err != nil {
			return err
		}
		if successor == nil {
			return nil
		}
		return tx.Model(successor).Update("is_current", true).Error
	})
	if err != nil {
		log.Printf("❌ Error deleting property %d: %v", property.ID, err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete property")
		return
	}

	log.Printf("Property deleted: ID=%d, UserID=%d", property.ID, userID)
	c.JSON(http.StatusOK, gin.H{"message": "Property deleted"})
}

// sharedPropertyIDs reports which of the given properties have a member linked
// to an account other than the owner's. Virtual members don't count: they
// have no account to lose the home from.
func sharedPropertyIDs(db *gorm.DB, propertyIDs []uint) (map[uint]bool, error) {
	shared := make(map[uint]bool)
	if len(propertyIDs) == 0 {
		return shared, nil
	}
	var ids []uint
	err := db.Model(&models.HouseholdMember{}).
		Joins("JOIN properties ON properties.household_id = household_members.household_id").
		Where("properties.id IN ? AND household_members.is_virtual = ?", propertyIDs, false).
		Where("household_members.user_id IS NOT NULL AND household_members.user_id <> properties.user_id").
		Distinct().Pluck("properties.id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		shared[id] = true
	}
	return shared, nil
}
