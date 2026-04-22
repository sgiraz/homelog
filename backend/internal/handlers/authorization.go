package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// isPropertyAdmin checks if the user is an admin of the given property.
func isPropertyAdmin(db *gorm.DB, userID uint, propertyID uint) bool {
	var count int64
	db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND property_id = ? AND role = 'admin'", userID, propertyID).
		Count(&count)
	return count > 0
}

// isPropertyMember checks if the user is a member (any role) of the given property.
func isPropertyMember(db *gorm.DB, userID uint, propertyID uint) bool {
	var count int64
	db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND property_id = ?", userID, propertyID).
		Count(&count)
	return count > 0
}

// requirePropertyAdmin returns 403 if the user is not an admin of the given property.
// Returns true if authorized, false if the response was already sent.
func requirePropertyAdmin(c *gin.Context, db *gorm.DB, userID uint, propertyID uint) bool {
	if !isPropertyAdmin(db, userID, propertyID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo gli amministratori della proprietà possono eseguire questa operazione"})
		return false
	}
	return true
}

// requirePropertyMember returns 403 if the user is not a member (any role) of
// the given property. Use this for read endpoints that any member can access.
func requirePropertyMember(c *gin.Context, db *gorm.DB, userID uint, propertyID uint) bool {
	if !isPropertyMember(db, userID, propertyID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Non sei membro di questa proprietà"})
		return false
	}
	return true
}
