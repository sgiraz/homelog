package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
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
	ID          uint   `json:"id"`
	HouseholdID uint   `json:"household_id"`
	UserID      *uint  `json:"user_id,omitempty"`
	Name        string `json:"name"`
	Role        string `json:"role,omitempty"`
	IsVirtual   bool   `json:"is_virtual"`
	UserRole    string `json:"user_role,omitempty"`   // "admin" or "user" (from linked User account)
	AvatarPath  string `json:"avatar_path,omitempty"` // relative path e.g. "avatars/2_123.jpg"
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

// List - GET /api/v1/households/:id/members
func (h *MemberHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	householdID, ok := householdParam(c)
	if !ok {
		return
	}

	if !requireHouseholdMember(c, h.db, userID, householdID) {
		return
	}

	var members []models.HouseholdMember
	if err := h.db.Preload("User").Where("household_id = ?", householdID).Find(&members).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch members")
		return
	}

	// Convert to response
	response := make([]MemberResponse, len(members))
	for i, m := range members {
		resp := MemberResponse{
			ID:          m.ID,
			HouseholdID: m.HouseholdID,
			UserID:      m.UserID,
			Name:        m.Name,
			Role:        m.Role,
			IsVirtual:   m.IsVirtual,
		}
		if m.User != nil {
			resp.UserRole = m.User.Role
			resp.AvatarPath = m.User.AvatarPath
		}
		response[i] = resp
	}

	c.JSON(http.StatusOK, response)
}

// Create - POST /api/v1/households/:id/members
func (h *MemberHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	// Parse household ID from URL
	householdID, ok := householdParam(c)
	if !ok {
		return
	}

	if !requireHouseholdAdmin(c, h.db, userID, householdID) {
		return
	}

	var req CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid request body: "+err.Error())
		return
	}

	// Create virtual member (UserID = nil)
	member := models.HouseholdMember{
		HouseholdID: householdID,
		UserID:      nil, // Virtual member
		Name:        req.Name,
		Role:        req.Role,
		IsVirtual:   true,
	}

	if err := h.db.Create(&member).Error; err != nil {
		log.Printf("ERROR creating member: %v", err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create member")
		return
	}

	log.Printf("✅ Virtual member created: ID=%d, Name=%s, HouseholdID=%d", member.ID, member.Name, member.HouseholdID)

	c.JSON(http.StatusCreated, MemberResponse{
		ID:          member.ID,
		HouseholdID: member.HouseholdID,
		UserID:      member.UserID,
		Name:        member.Name,
		Role:        member.Role,
		IsVirtual:   member.IsVirtual,
	})
}

// Get - GET /api/v1/members/:id
func (h *MemberHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	memberIDStr := c.Param("id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_member_id", "Invalid member id")
		return
	}

	var member models.HouseholdMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "member_not_found", "Member not found")
			return
		}
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch member")
		return
	}

	if !requireHouseholdMember(c, h.db, userID, member.HouseholdID) {
		return
	}

	c.JSON(http.StatusOK, MemberResponse{
		ID:          member.ID,
		HouseholdID: member.HouseholdID,
		UserID:      member.UserID,
		Name:        member.Name,
		Role:        member.Role,
		IsVirtual:   member.IsVirtual,
	})
}

// Update - PUT /api/v1/members/:id
func (h *MemberHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	memberIDStr := c.Param("id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_member_id", "Invalid member id")
		return
	}

	var req UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "The request could not be processed")
		return
	}

	var member models.HouseholdMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "member_not_found", "Member not found")
			return
		}
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch member")
		return
	}

	if !requireHouseholdAdmin(c, h.db, userID, member.HouseholdID) {
		return
	}

	if req.Name != "" {
		member.Name = req.Name
	}
	if req.Role != "" {
		member.Role = req.Role
	}

	if err := h.db.Save(&member).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update member")
		return
	}

	c.JSON(http.StatusOK, MemberResponse{
		ID:          member.ID,
		HouseholdID: member.HouseholdID,
		UserID:      member.UserID,
		Name:        member.Name,
		Role:        member.Role,
		IsVirtual:   member.IsVirtual,
	})
}

// Delete - DELETE /api/v1/members/:id
func (h *MemberHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	memberIDStr := c.Param("id")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_member_id", "Invalid member id")
		return
	}

	var member models.HouseholdMember
	if err := h.db.First(&member, memberID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "member_not_found", "Member not found")
			return
		}
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch member")
		return
	}

	if !requireHouseholdAdmin(c, h.db, userID, member.HouseholdID) {
		return
	}

	// Don't allow deleting non-virtual members (linked to User accounts)
	if !member.IsVirtual {
		apierr.Fail(c, http.StatusForbidden, "member_linked_to_account", "A member linked to a user account cannot be deleted")
		return
	}

	// Check if member has any expenses or splits
	var expenseCount int64
	h.db.Model(&models.Expense{}).Where("paid_by_member_id = ?", memberID).Count(&expenseCount)
	if expenseCount > 0 {
		apierr.Fail(c, http.StatusConflict, "member_has_expenses", "This member has expenses and cannot be deleted")
		return
	}

	var splitCount int64
	h.db.Model(&models.ExpenseSplit{}).Where("member_id = ?", memberID).Count(&splitCount)
	if splitCount > 0 {
		apierr.Fail(c, http.StatusConflict, "member_has_splits", "This member has expense shares and cannot be deleted")
		return
	}

	if err := h.db.Delete(&member).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete member")
		return
	}

	if err := cleanupSplitDefaults(h.db, uint(memberID)); err != nil {
		log.Printf("❌ Error cleaning split defaults for member %d: %v", memberID, err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member deleted successfully"})
}

// cleanupSplitDefaults removes deleted member IDs from the lists that name
// who shares an expense by default: every user's default_split_with_member_ids
// and every service's split_member_ids. Both are JSON, so no foreign key keeps
// them in step with household_members.
func cleanupSplitDefaults(db *gorm.DB, deletedMemberIDs ...uint) error {
	if len(deletedMemberIDs) == 0 {
		return nil
	}
	gone := make(map[uint]bool, len(deletedMemberIDs))
	for _, id := range deletedMemberIDs {
		gone[id] = true
	}
	if err := pruneMemberLists(db, &models.UserSettings{}, "default_split_with_member_ids", gone); err != nil {
		return err
	}
	return pruneMemberLists(db, &models.Utility{}, "split_member_ids", gone)
}

// pruneMemberLists drops the gone IDs from a JSON member list column. A list
// that does not parse is left alone: nothing in it can be trusted to remove.
func pruneMemberLists(db *gorm.DB, model any, column string, gone map[uint]bool) error {
	var rows []struct {
		ID   uint
		List string
	}
	if err := db.Unscoped().Model(model).Select("id, " + column + " AS list").
		Where(column + " IS NOT NULL AND " + column + " != ''").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var ids []uint
		if err := json.Unmarshal([]byte(row.List), &ids); err != nil {
			continue
		}
		kept := make([]uint, 0, len(ids))
		for _, id := range ids {
			if !gone[id] {
				kept = append(kept, id)
			}
		}
		if len(kept) == len(ids) {
			continue
		}
		list, err := json.Marshal(kept)
		if err != nil {
			return err
		}
		if err := db.Unscoped().Model(model).Where("id = ?", row.ID).UpdateColumn(column, string(list)).Error; err != nil {
			return err
		}
	}
	return nil
}
