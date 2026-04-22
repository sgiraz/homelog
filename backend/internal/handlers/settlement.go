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

// SettlementHandler handles settlement operations
type SettlementHandler struct {
	db *gorm.DB
}

// NewSettlementHandler creates a new settlement handler
func NewSettlementHandler(db *gorm.DB) *SettlementHandler {
	return &SettlementHandler{db: db}
}

// CreateSettlementRequest represents the request body for creating a settlement
type CreateSettlementRequest struct {
	PropertyID    uint    `json:"property_id" binding:"required"`
	FromMemberID  uint    `json:"from_member_id" binding:"required"`
	ToMemberID    uint    `json:"to_member_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	Date          string  `json:"date" binding:"required"`
	PaymentMethod string  `json:"payment_method"`
	Note          string  `json:"note"`
}

// Create registers a payment between members
// POST /api/v1/settlements
func (h *SettlementHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR binding settlement JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate: from_member != to_member
	if req.FromMemberID == req.ToMemberID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "FromMemberID and ToMemberID cannot be the same"})
		return
	}

	// Find the current user's member for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", req.PropertyID, userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be a member of this property"})
		return
	}

	// Validate: current user's member is involved in the settlement
	if currentMember.ID != req.FromMemberID && currentMember.ID != req.ToMemberID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be involved in the settlement"})
		return
	}

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	// Verify property exists
	var property models.Property
	if err := h.db.First(&property, req.PropertyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify property"})
		}
		return
	}

	// Start transaction
	tx := h.db.Begin()

	// Create Settlement record
	settlement := models.Settlement{
		PropertyID:    req.PropertyID,
		FromMemberID:  req.FromMemberID,
		ToMemberID:    req.ToMemberID,
		Amount:        req.Amount,
		Date:          date,
		PaymentMethod: req.PaymentMethod,
		Note:          req.Note,
	}

	if err := tx.Create(&settlement).Error; err != nil {
		tx.Rollback()
		log.Printf("ERROR creating settlement: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create settlement"})
		return
	}

	log.Printf("Settlement created - ID: %d, From: %d, To: %d, Amount: %.2f",
		settlement.ID, req.FromMemberID, req.ToMemberID, req.Amount)

	// Mark unsettled ExpenseSplits as settled
	// SQLite doesn't support UPDATE with JOIN, so we use a two-step approach:
	// 1. Find the IDs of expense_splits that need to be updated
	// 2. Update those specific IDs

	// Step 1: Find expense_split IDs to update
	var splitIDs []uint
	err = tx.Model(&models.ExpenseSplit{}).
		Select("expense_splits.id").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expense_splits.is_settled = ?", false).
		Where("expenses.property_id = ?", req.PropertyID).
		Where("(expenses.paid_by_member_id = ? AND expense_splits.member_id = ?) OR (expenses.paid_by_member_id = ? AND expense_splits.member_id = ?)",
			req.ToMemberID, req.FromMemberID, req.FromMemberID, req.ToMemberID).
		Pluck("expense_splits.id", &splitIDs).Error

	if err != nil {
		tx.Rollback()
		log.Printf("ERROR finding expense splits to settle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find expense splits"})
		return
	}

	log.Printf("Found %d expense splits to settle: %v", len(splitIDs), splitIDs)

	// Step 2: Update those splits by ID
	now := time.Now()
	var result *gorm.DB
	if len(splitIDs) > 0 {
		result = tx.Model(&models.ExpenseSplit{}).
			Where("id IN ?", splitIDs).
			Updates(map[string]any{
				"is_settled":    true,
				"settled_at":    now,
				"settlement_id": settlement.ID,
			})

		if result.Error != nil {
			tx.Rollback()
			log.Printf("ERROR updating expense splits: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense splits"})
			return
		}

		log.Printf("✅ SETTLEMENT: Marked %d splits as settled", result.RowsAffected)
	} else {
		log.Printf("ℹ️ No unsettled splits found to mark as settled")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Printf("ERROR committing transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete settlement"})
		return
	}

	// Reload with relations
	h.db.Preload("FromMember").Preload("ToMember").First(&settlement, settlement.ID)

	c.JSON(http.StatusCreated, settlement)
}

// List returns settlements for a property
// GET /api/v1/settlements?property_id=1
func (h *SettlementHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get property_id from query
	propertyIDStr := c.Query("property_id")
	if propertyIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property_id is required"})
		return
	}

	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property_id"})
		return
	}

	// Find the current user's member for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", propertyID, userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"settlements": []models.Settlement{},
			"total":       0,
		})
		return
	}

	var settlements []models.Settlement
	query := h.db.
		Preload("FromMember").
		Preload("ToMember").
		Where("property_id = ?", propertyID).
		Where("from_member_id = ? OR to_member_id = ?", currentMember.ID, currentMember.ID).
		Order("date DESC")

	// Optional date filters
	if fromDate := c.Query("from_date"); fromDate != "" {
		query = query.Where("date >= ?", fromDate)
	}
	if toDate := c.Query("to_date"); toDate != "" {
		query = query.Where("date <= ?", toDate)
	}

	if err := query.Find(&settlements).Error; err != nil {
		log.Printf("ERROR fetching settlements: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settlements"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"settlements": settlements,
		"total":       len(settlements),
	})
}

// Get returns a single settlement by ID
// GET /api/v1/settlements/:id
func (h *SettlementHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settlement ID"})
		return
	}

	var settlement models.Settlement
	if err := h.db.
		Preload("FromMember").
		Preload("ToMember").
		Preload("ExpenseSplits").
		First(&settlement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Settlement not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settlement"})
		}
		return
	}

	// Find the current user's member for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", settlement.PropertyID, userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be a member of this property"})
		return
	}

	// Verify user is involved
	if settlement.FromMemberID != currentMember.ID && settlement.ToMemberID != currentMember.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be involved in the settlement"})
		return
	}

	c.JSON(http.StatusOK, settlement)
}

// Delete removes a settlement (soft delete)
// DELETE /api/v1/settlements/:id
func (h *SettlementHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settlement ID"})
		return
	}

	// Find the settlement
	var settlement models.Settlement
	if err := h.db.First(&settlement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Settlement not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find settlement"})
		}
		return
	}

	// Find the current user's member for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", settlement.PropertyID, userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be a member of this property"})
		return
	}

	// Verify user is involved
	if settlement.FromMemberID != currentMember.ID && settlement.ToMemberID != currentMember.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be involved in the settlement"})
		return
	}

	// Start transaction
	tx := h.db.Begin()

	// Unsettle the expense splits linked to this settlement
	if err := tx.Model(&models.ExpenseSplit{}).
		Where("settlement_id = ?", settlement.ID).
		Updates(map[string]any{
			"is_settled":    false,
			"settled_at":    nil,
			"settlement_id": nil,
		}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsettle expense splits"})
		return
	}

	// Delete settlement
	if err := tx.Delete(&settlement).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete settlement"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settlement deleted successfully"})
}
