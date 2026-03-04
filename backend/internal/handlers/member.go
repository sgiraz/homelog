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

type MemberHandler struct {
	db *gorm.DB
}

func NewMemberHandler(db *gorm.DB) *MemberHandler {
	return &MemberHandler{db: db}
}

// MemberResponse represents a household member response
type MemberResponse struct {
	ID         uint   `json:"id"`
	PropertyID uint   `json:"property_id"`
	UserID     *uint  `json:"user_id,omitempty"`
	Name       string `json:"name"`
	Role       string `json:"role,omitempty"`
	IsVirtual  bool   `json:"is_virtual"`
	UserRole   string `json:"user_role,omitempty"` // "admin" or "user" (from linked User account)
}

// CreateMemberRequest represents the request for creating a member
type CreateMemberRequest struct {
	Name string `json:"name" binding:"required"`
	Role string `json:"role,omitempty"`
}

// UpdateMemberRequest represents the request for updating a member
type UpdateMemberRequest struct {
	Name string `json:"name,omitempty"`
	Role string `json:"role,omitempty"`
}

// List - GET /api/v1/properties/:id/members
func (h *MemberHandler) List(c *gin.Context) {
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

	var members []models.HouseholdMember
	if err := h.db.Preload("User").Where("property_id = ?", propertyID).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch members"})
		return
	}

	// Convert to response
	response := make([]MemberResponse, len(members))
	for i, m := range members {
		resp := MemberResponse{
			ID:         m.ID,
			PropertyID: m.PropertyID,
			UserID:     m.UserID,
			Name:       m.Name,
			Role:       m.Role,
			IsVirtual:  m.IsVirtual,
		}
		if m.User != nil {
			resp.UserRole = m.User.Role
		}
		response[i] = resp
	}

	c.JSON(http.StatusOK, response)
}

// Create - POST /api/v1/properties/:id/members
func (h *MemberHandler) Create(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse property ID from URL
	propertyIDStr := c.Param("id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	var req CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Create virtual member (UserID = nil)
	member := models.HouseholdMember{
		PropertyID: uint(propertyID),
		UserID:     nil, // Virtual member
		Name:       req.Name,
		Role:       req.Role,
		IsVirtual:  true,
	}

	if err := h.db.Create(&member).Error; err != nil {
		log.Printf("ERROR creating member: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create member"})
		return
	}

	log.Printf("✅ Virtual member created: ID=%d, Name=%s, PropertyID=%d", member.ID, member.Name, member.PropertyID)

	c.JSON(http.StatusCreated, MemberResponse{
		ID:         member.ID,
		PropertyID: member.PropertyID,
		UserID:     member.UserID,
		Name:       member.Name,
		Role:       member.Role,
		IsVirtual:  member.IsVirtual,
	})
}

// Get - GET /api/v1/members/:id
func (h *MemberHandler) Get(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse member ID
	memberIDStr := c.Param("id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	var member models.HouseholdMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Member not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch member"})
		return
	}

	c.JSON(http.StatusOK, MemberResponse{
		ID:         member.ID,
		PropertyID: member.PropertyID,
		UserID:     member.UserID,
		Name:       member.Name,
		Role:       member.Role,
		IsVirtual:  member.IsVirtual,
	})
}

// Update - PUT /api/v1/members/:id
func (h *MemberHandler) Update(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse member ID
	memberIDStr := c.Param("id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var member models.HouseholdMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Member not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch member"})
		return
	}

	// Update fields
	if req.Name != "" {
		member.Name = req.Name
	}
	if req.Role != "" {
		member.Role = req.Role
	}

	if err := h.db.Save(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member"})
		return
	}

	c.JSON(http.StatusOK, MemberResponse{
		ID:         member.ID,
		PropertyID: member.PropertyID,
		UserID:     member.UserID,
		Name:       member.Name,
		Role:       member.Role,
		IsVirtual:  member.IsVirtual,
	})
}

// Delete - DELETE /api/v1/members/:id
func (h *MemberHandler) Delete(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse member ID
	memberIDStr := c.Param("id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}

	var member models.HouseholdMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Member not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch member"})
		return
	}

	// Don't allow deleting non-virtual members (linked to User accounts)
	if !member.IsVirtual {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete a member linked to a user account"})
		return
	}

	// Check if member has any expenses or splits
	var expenseCount int64
	h.db.Model(&models.Expense{}).Where("paid_by_member_id = ?", memberID).Count(&expenseCount)
	if expenseCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete member with associated expenses"})
		return
	}

	var splitCount int64
	h.db.Model(&models.ExpenseSplit{}).Where("member_id = ?", memberID).Count(&splitCount)
	if splitCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete member with associated expense splits"})
		return
	}

	if err := h.db.Delete(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member deleted successfully"})
}
