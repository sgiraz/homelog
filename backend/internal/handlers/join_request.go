package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type JoinRequestHandler struct {
	db *gorm.DB
}

func NewJoinRequestHandler(db *gorm.DB) *JoinRequestHandler {
	return &JoinRequestHandler{db: db}
}

// CreateJoinRequestBody is the request body for creating a join request
type CreateJoinRequestBody struct {
	PropertyID uint `json:"property_id" binding:"required"`
}

// ResolveJoinRequestBody is the request body for resolving a join request
type ResolveJoinRequestBody struct {
	Status string `json:"status" binding:"required"` // "approved" or "rejected"
}

// JoinablePropertyResponse limits exposed fields for the joinable properties list
type JoinablePropertyResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Residents int    `json:"residents"`
}

// Create - POST /api/v1/join-requests
func (h *JoinRequestHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var req CreateJoinRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Validate: property exists
	var property models.Property
	if err := h.db.First(&property, req.PropertyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "property_not_found", "Property not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch property")
		}
		return
	}

	// Validate: user is not already a member of this property
	var memberCount int64
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND property_id = ?", userID, req.PropertyID).
		Count(&memberCount)
	if memberCount > 0 {
		apierr.Fail(c, http.StatusConflict, "already_property_member", "You are already a member of this property")
		return
	}

	// Validate: no pending request already exists for this user+property
	var existingCount int64
	h.db.Model(&models.PropertyJoinRequest{}).
		Where("user_id = ? AND property_id = ? AND status = 'pending'", userID, req.PropertyID).
		Count(&existingCount)
	if existingCount > 0 {
		apierr.Fail(c, http.StatusConflict, "join_request_pending", "A join request for this property is already pending")
		return
	}

	joinReq := models.PropertyJoinRequest{
		UserID:     userID,
		PropertyID: req.PropertyID,
		Status:     "pending",
	}

	if err := h.db.Create(&joinReq).Error; err != nil {
		log.Printf("❌ Failed to create join request: %v", err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create join request")
		return
	}

	// Preload relations for the response
	h.db.Preload("User").Preload("Property").First(&joinReq, joinReq.ID)

	log.Printf("✅ Join request created: ID=%d, UserID=%d, PropertyID=%d", joinReq.ID, userID, req.PropertyID)

	// Notify property admins
	var adminUserIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("property_id = ? AND role = 'admin' AND user_id IS NOT NULL", req.PropertyID).
		Pluck("user_id", &adminUserIDs)

	for _, adminUID := range adminUserIDs {
		if adminUID == userID {
			continue
		}
		relID := joinReq.ID
		propID := req.PropertyID
		createNotification(h.db, adminUID, "join_request",
			fmt.Sprintf("%s vuole unirsi a %s", joinReq.User.Name, property.Name),
			fmt.Sprintf("%s ha richiesto di unirsi alla proprietà %s.", joinReq.User.Name, property.Name),
			&relID, &propID)
	}

	c.JSON(http.StatusCreated, joinReq)
}

// List - GET /api/v1/join-requests
func (h *JoinRequestHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	// Find property IDs where user is admin
	var adminPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND role = 'admin'", userID).
		Pluck("property_id", &adminPropertyIDs)

	var results []models.PropertyJoinRequest

	// If admin of any property: fetch pending requests for those properties
	if len(adminPropertyIDs) > 0 {
		h.db.Where("property_id IN ? AND status = 'pending'", adminPropertyIDs).
			Preload("User").
			Preload("Property").
			Find(&results)
	}

	// Always also fetch the user's own requests (all statuses)
	var ownRequests []models.PropertyJoinRequest
	h.db.Where("user_id = ?", userID).
		Preload("User").
		Preload("Property").
		Find(&ownRequests)

	// Merge and deduplicate by ID
	seen := make(map[uint]bool)
	for _, r := range results {
		seen[r.ID] = true
	}
	for _, r := range ownRequests {
		if !seen[r.ID] {
			seen[r.ID] = true
			results = append(results, r)
		}
	}

	c.JSON(http.StatusOK, results)
}

// Resolve - PATCH /api/v1/join-requests/:id
func (h *JoinRequestHandler) Resolve(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_join_request_id", "Invalid join request id")
		return
	}

	var req ResolveJoinRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.Status != "approved" && req.Status != "rejected" {
		apierr.Fail(c, http.StatusBadRequest, "invalid_join_request_status", "Status must be 'approved' or 'rejected'")
		return
	}

	// Find the join request
	var joinReq models.PropertyJoinRequest
	if err := h.db.Preload("Property").First(&joinReq, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "join_request_not_found", "Join request not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch join request")
		}
		return
	}

	// Validate: request is still pending
	if joinReq.Status != "pending" {
		apierr.Fail(c, http.StatusConflict, "join_request_resolved", "This join request has already been resolved")
		return
	}

	// Validate: current user is admin of the request's property
	var adminMemberCount int64
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND property_id = ? AND role = 'admin'", userID, joinReq.PropertyID).
		Count(&adminMemberCount)
	if adminMemberCount == 0 {
		apierr.Fail(c, http.StatusForbidden, "not_property_admin", "You are not an admin of this property")
		return
	}

	now := time.Now()

	tx := h.db.Begin()
	if tx.Error != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to start transaction")
		return
	}

	if req.Status == "approved" {
		// Check if user is already a member (e.g., added manually while request was pending)
		var existingCount int64
		tx.Model(&models.HouseholdMember{}).
			Where("user_id = ? AND property_id = ?", joinReq.UserID, joinReq.PropertyID).
			Count(&existingCount)

		if existingCount == 0 {
			// Fetch the requesting user's name for the household member record
			var requestingUser models.User
			if err := tx.First(&requestingUser, joinReq.UserID).Error; err != nil {
				tx.Rollback()
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch requesting user")
				return
			}

			// Create HouseholdMember for the requesting user
			member := models.HouseholdMember{
				PropertyID: joinReq.PropertyID,
				UserID:     &joinReq.UserID,
				Name:       requestingUser.Name,
				Role:       "member",
				IsVirtual:  false,
			}
			if err := tx.Create(&member).Error; err != nil {
				tx.Rollback()
				log.Printf("❌ Failed to create household member on join request approval: %v", err)
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to add member to property")
				return
			}

			// Increment property residents count
			tx.Model(&models.Property{}).Where("id = ?", joinReq.PropertyID).
				UpdateColumn("residents", gorm.Expr("residents + ?", 1))

			log.Printf("✅ Join request approved: UserID=%d joined PropertyID=%d as MemberID=%d",
				joinReq.UserID, joinReq.PropertyID, member.ID)
		} else {
			log.Printf("ℹ️ Join request approved but UserID=%d is already a member of PropertyID=%d",
				joinReq.UserID, joinReq.PropertyID)
		}
	}

	// Update request status
	resolvedByID := userID
	if err := tx.Model(&joinReq).Updates(map[string]any{
		"status":      req.Status,
		"resolved_by": resolvedByID,
		"resolved_at": now,
	}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update join request")
		return
	}

	if err := tx.Commit().Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to complete operation")
		return
	}

	h.db.Preload("User").Preload("Property").First(&joinReq, joinReq.ID)

	log.Printf("✅ Join request ID=%d resolved as '%s' by UserID=%d", joinReq.ID, req.Status, userID)

	c.JSON(http.StatusOK, joinReq)
}

// ListJoinable - GET /api/v1/properties/joinable
func (h *JoinRequestHandler) ListJoinable(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	// Find property IDs where user is already a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Find property IDs where user has a pending request
	var pendingPropertyIDs []uint
	h.db.Model(&models.PropertyJoinRequest{}).
		Where("user_id = ? AND status = 'pending'", userID).
		Pluck("property_id", &pendingPropertyIDs)

	// Build exclusion list
	excludedIDs := make([]uint, 0, len(memberPropertyIDs)+len(pendingPropertyIDs))
	excludedIDs = append(excludedIDs, memberPropertyIDs...)
	excludedIDs = append(excludedIDs, pendingPropertyIDs...)

	var properties []models.Property
	query := h.db.Model(&models.Property{})
	if len(excludedIDs) > 0 {
		query = query.Where("id NOT IN ?", excludedIDs)
	}
	if err := query.Order("name ASC").Find(&properties).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch properties")
		return
	}

	// Map to limited response struct (no sensitive data)
	response := make([]JoinablePropertyResponse, len(properties))
	for i, p := range properties {
		response[i] = JoinablePropertyResponse{
			ID:        p.ID,
			Name:      p.Name,
			Address:   p.Address,
			Residents: p.Residents,
		}
	}

	c.JSON(http.StatusOK, response)
}
