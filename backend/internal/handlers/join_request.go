package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/i18n"
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
	HouseholdID uint `json:"household_id" binding:"required"`
}

// ResolveJoinRequestBody is the request body for resolving a join request
type ResolveJoinRequestBody struct {
	Status string `json:"status" binding:"required"` // "approved" or "rejected"
}

// JoinableHouseholdResponse limits exposed fields for the joinable households list
type JoinableHouseholdResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"` // of its primary home
	MemberCount int64  `json:"member_count"`
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

	// Validate: household exists
	var household models.Household
	if err := h.db.First(&household, req.HouseholdID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "household_not_found", "Household not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch household")
		}
		return
	}

	// Validate: user is not in a household already (an account belongs to one)
	current, err := userHouseholdID(h.db, userID)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch household")
		return
	}
	if current != 0 {
		apierr.Fail(c, http.StatusConflict, "already_in_household", "You already belong to a household")
		return
	}

	// Validate: no pending request already exists for this user+household
	var existingCount int64
	h.db.Model(&models.HouseholdJoinRequest{}).
		Where("user_id = ? AND household_id = ? AND status = 'pending'", userID, req.HouseholdID).
		Count(&existingCount)
	if existingCount > 0 {
		apierr.Fail(c, http.StatusConflict, "join_request_pending", "A join request for this household is already pending")
		return
	}

	joinReq := models.HouseholdJoinRequest{
		UserID:      userID,
		HouseholdID: req.HouseholdID,
		Status:      "pending",
	}

	if err := h.db.Create(&joinReq).Error; err != nil {
		log.Printf("❌ Failed to create join request: %v", err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create join request")
		return
	}

	// Preload relations for the response
	h.db.Preload("User").Preload("Household").First(&joinReq, joinReq.ID)

	log.Printf("✅ Join request created: ID=%d, UserID=%d, HouseholdID=%d", joinReq.ID, userID, req.HouseholdID)

	// Notify household admins
	var adminUserIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("household_id = ? AND role = 'admin' AND user_id IS NOT NULL", req.HouseholdID).
		Pluck("user_id", &adminUserIDs)

	for _, adminUID := range adminUserIDs {
		if adminUID == userID {
			continue
		}
		relID := joinReq.ID
		// Written in the recipient's language, not the requester's.
		lang := i18n.UserLanguage(h.db, adminUID)
		createNotification(h.db, adminUID, "join_request",
			i18n.T(lang, "notification.join_request.title",
				"name", joinReq.User.Name, "household", household.Name),
			i18n.T(lang, "notification.join_request.body",
				"name", joinReq.User.Name, "household", household.Name),
			&relID, nil)
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

	// Find household IDs where user is admin
	var adminHouseholdIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ? AND role = 'admin'", userID).
		Pluck("household_id", &adminHouseholdIDs)

	results := []models.HouseholdJoinRequest{}

	// If admin of any household: fetch pending requests for those households
	if len(adminHouseholdIDs) > 0 {
		h.db.Where("household_id IN ? AND status = 'pending'", adminHouseholdIDs).
			Preload("User").
			Preload("Household").
			Find(&results)
	}

	// Always also fetch the user's own requests (all statuses)
	var ownRequests []models.HouseholdJoinRequest
	h.db.Where("user_id = ?", userID).
		Preload("User").
		Preload("Household").
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
	var joinReq models.HouseholdJoinRequest
	if err := h.db.First(&joinReq, id).Error; err != nil {
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

	// Validate: current user is admin of the request's household
	if !requireHouseholdAdmin(c, h.db, userID, joinReq.HouseholdID) {
		return
	}

	now := time.Now()

	tx := h.db.Begin()
	if tx.Error != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to start transaction")
		return
	}

	if req.Status == "approved" {
		// Check if user is already in a household (e.g., added manually while
		// request was pending): this one is fine, another one is not
		current, err := userHouseholdID(tx, joinReq.UserID)
		if err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch household")
			return
		}
		if current != 0 && current != joinReq.HouseholdID {
			tx.Rollback()
			apierr.Fail(c, http.StatusConflict, "requester_in_household", "The requester already belongs to another household")
			return
		}
		if current == 0 {
			// Fetch the requesting user's name for the household member record
			var requestingUser models.User
			if err := tx.First(&requestingUser, joinReq.UserID).Error; err != nil {
				tx.Rollback()
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch requesting user")
				return
			}

			// Create HouseholdMember for the requesting user
			member := models.HouseholdMember{
				HouseholdID: joinReq.HouseholdID,
				UserID:      &joinReq.UserID,
				Name:        requestingUser.Name,
				Role:        "member",
				IsVirtual:   false,
			}
			if err := tx.Create(&member).Error; err != nil {
				tx.Rollback()
				log.Printf("❌ Failed to create household member on join request approval: %v", err)
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to add member to household")
				return
			}

			// Increment residents count of the household's homes
			if err := tx.Model(&models.Property{}).Where("household_id = ?", joinReq.HouseholdID).
				UpdateColumn("residents", gorm.Expr("residents + ?", 1)).Error; err != nil {
				tx.Rollback()
				apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update residents")
				return
			}

			log.Printf("✅ Join request approved: UserID=%d joined HouseholdID=%d as MemberID=%d",
				joinReq.UserID, joinReq.HouseholdID, member.ID)
		} else {
			log.Printf("ℹ️ Join request approved but UserID=%d is already a member of HouseholdID=%d",
				joinReq.UserID, joinReq.HouseholdID)
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

	h.db.Preload("User").Preload("Household").First(&joinReq, joinReq.ID)

	log.Printf("✅ Join request ID=%d resolved as '%s' by UserID=%d", joinReq.ID, req.Status, userID)

	c.JSON(http.StatusOK, joinReq)
}

// ListJoinable - GET /api/v1/households/joinable
func (h *JoinRequestHandler) ListJoinable(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	// Someone in a household has none to join: an account belongs to one.
	current, err := userHouseholdID(h.db, userID)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch households")
		return
	}
	if current != 0 {
		c.JSON(http.StatusOK, []JoinableHouseholdResponse{})
		return
	}

	pending := h.db.Model(&models.HouseholdJoinRequest{}).Select("household_id").
		Where("user_id = ? AND status = 'pending'", userID)

	var households []models.Household
	if err := h.db.Where("id NOT IN (?)", pending).
		Order("name ASC").Find(&households).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch households")
		return
	}

	// Map to limited response struct (no sensitive data)
	response := make([]JoinableHouseholdResponse, len(households))
	for i, hh := range households {
		var home models.Property
		if err := h.db.Where("household_id = ?", hh.ID).
			Order("is_current DESC, created_at ASC").Limit(1).Find(&home).Error; err != nil {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch households")
			return
		}
		var members int64
		if err := h.db.Model(&models.HouseholdMember{}).Where("household_id = ?", hh.ID).
			Count(&members).Error; err != nil {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch households")
			return
		}
		response[i] = JoinableHouseholdResponse{
			ID:          hh.ID,
			Name:        hh.Name,
			Address:     home.Address,
			MemberCount: members,
		}
	}

	c.JSON(http.StatusOK, response)
}
