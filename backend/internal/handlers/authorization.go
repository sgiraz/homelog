package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// isHouseholdAdmin checks if the user is an admin of the given household.
func isHouseholdAdmin(db *gorm.DB, userID uint, householdID uint) bool {
	var count int64
	db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND household_id = ? AND role = 'admin'", userID, householdID).
		Count(&count)
	return count > 0
}

// isHouseholdMember checks if the user is a member (any role) of the given household.
func isHouseholdMember(db *gorm.DB, userID uint, householdID uint) bool {
	var count int64
	db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND household_id = ?", userID, householdID).
		Count(&count)
	return count > 0
}

// requireHouseholdAdmin returns 403 if the user is not an admin of the given
// household. Returns true if authorized, false if the response was already sent.
func requireHouseholdAdmin(c *gin.Context, db *gorm.DB, userID uint, householdID uint) bool {
	if !isHouseholdAdmin(db, userID, householdID) {
		apierr.Fail(c, http.StatusForbidden, "household_admin_only", "Only household admins can do this")
		return false
	}
	return true
}

// requireHouseholdMember returns 403 if the user is not a member (any role) of
// the given household.
func requireHouseholdMember(c *gin.Context, db *gorm.DB, userID uint, householdID uint) bool {
	if !isHouseholdMember(db, userID, householdID) {
		apierr.Fail(c, http.StatusForbidden, "not_household_member", "You are not a member of this household")
		return false
	}
	return true
}

// householdParam reads the :id path parameter of a /households/:id route.
// Returns false if the response was already sent.
func householdParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_household_id", "Invalid household id")
		return 0, false
	}
	return uint(id), true
}

// householdScope reads the optional ?household_id= filter of a list: the
// households to read, either that one (the user must be a member) or every
// household of the user. Returns false if the response was already sent.
func householdScope(c *gin.Context, db *gorm.DB, userID uint) ([]uint, bool) {
	raw := c.Query("household_id")
	if raw == "" {
		ids, err := userHouseholdIDs(db, userID)
		if err != nil {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch households")
			return nil, false
		}
		return ids, true
	}
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_household_id", "Invalid household id")
		return nil, false
	}
	if !requireHouseholdMember(c, db, userID, uint(id)) {
		return nil, false
	}
	return []uint{uint(id)}, true
}

// userHouseholdIDs lists the households the user is a member of.
func userHouseholdIDs(db *gorm.DB, userID uint) ([]uint, error) {
	ids := []uint{}
	err := db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Distinct().Pluck("household_id", &ids).Error
	return ids, err
}

// householdOfProperty returns the household a home belongs to.
func householdOfProperty(db *gorm.DB, propertyID uint) (uint, error) {
	var property models.Property
	if err := db.Select("id", "household_id").First(&property, propertyID).Error; err != nil {
		return 0, err
	}
	return property.HouseholdID, nil
}

// propertyMembers selects the household members with access to a home: the
// members of the household it belongs to.
func propertyMembers(db *gorm.DB, propertyID uint) *gorm.DB {
	return db.Model(&models.HouseholdMember{}).
		Where("household_id = (SELECT household_id FROM properties WHERE id = ?)", propertyID)
}

// isPropertyAdmin checks if the user is an admin of the household owning the home.
func isPropertyAdmin(db *gorm.DB, userID uint, propertyID uint) bool {
	var count int64
	propertyMembers(db, propertyID).
		Where("user_id = ? AND role = 'admin'", userID).
		Count(&count)
	return count > 0
}

// isPropertyMember checks if the user is a member (any role) of the household
// owning the home.
func isPropertyMember(db *gorm.DB, userID uint, propertyID uint) bool {
	var count int64
	propertyMembers(db, propertyID).
		Where("user_id = ?", userID).
		Count(&count)
	return count > 0
}

// requirePropertyAdmin returns 403 if the user is not an admin of the given property.
// Returns true if authorized, false if the response was already sent.
func requirePropertyAdmin(c *gin.Context, db *gorm.DB, userID uint, propertyID uint) bool {
	if !isPropertyAdmin(db, userID, propertyID) {
		apierr.Fail(c, http.StatusForbidden, "property_admin_only", "Only property admins can do this")
		return false
	}
	return true
}

// requirePropertyMember returns 403 if the user is not a member (any role) of
// the given property. Use this for read endpoints that any member can access.
func requirePropertyMember(c *gin.Context, db *gorm.DB, userID uint, propertyID uint) bool {
	if !isPropertyMember(db, userID, propertyID) {
		apierr.Fail(c, http.StatusForbidden, "not_property_member", "You are not a member of this property")
		return false
	}
	return true
}
