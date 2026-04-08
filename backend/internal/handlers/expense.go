package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// ExpenseHandler handles expense-related requests
type ExpenseHandler struct {
	db *gorm.DB
}

// NewExpenseHandler creates a new expense handler
func NewExpenseHandler(db *gorm.DB) *ExpenseHandler {
	return &ExpenseHandler{db: db}
}

// CreateExpenseRequest represents the request body for creating an expense
type CreateExpenseRequest struct {
	Amount             float64 `json:"amount" binding:"required,gt=0"`
	Description        string  `json:"description"`
	CategoryID         uint    `json:"category_id" binding:"required"`
	PropertyID         *uint   `json:"property_id"`
	SubcategoryID      *uint   `json:"subcategory_id"`
	ProjectID          *uint   `json:"project_id"`
	Date               string  `json:"date" binding:"required"` // Format: YYYY-MM-DD
	AttachmentURL      string  `json:"attachment_url"`
	PaidByMemberID     uint    `json:"paid_by_member_id"`
	IsSplit            bool    `json:"is_split"`
	SplitWithMemberIDs []uint  `json:"split_with_member_ids"`
}

// UpdateExpenseRequest represents the request body for updating an expense
type UpdateExpenseRequest struct {
	Amount        *float64 `json:"amount"`
	Description   *string  `json:"description"`
	CategoryID    *uint    `json:"category_id"`
	PropertyID    *uint    `json:"property_id"`
	SubcategoryID *uint    `json:"subcategory_id"`
	ProjectID     *uint    `json:"project_id"`
	Date          *string  `json:"date"`
	AttachmentURL *string  `json:"attachment_url"`
}

// MonthlyStats represents monthly expense statistics
type MonthlyStats struct {
	Month  string  `json:"month"`
	Year   int     `json:"year"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// TrendPoint represents a single data point in the trend chart (day/month/quarter)
type TrendPoint struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// CategoryStats represents expense statistics by category
type CategoryStats struct {
	CategoryID   uint    `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Amount       float64 `json:"amount"`
	Count        int     `json:"count"`
	Percentage   float64 `json:"percentage"`
}

// List returns all expenses for properties where the user is a member
// GET /api/v1/expenses?category_id=1&property_id=1&from=2024-01-01&to=2024-12-31&limit=50&offset=0
func (h *ExpenseHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Find member IDs for the current user across all their properties
	var currentMemberIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("id", &currentMemberIDs)

	// Find split expense IDs where the current user has a split record (as recipient)
	var splitExpenseIDs []uint
	if len(currentMemberIDs) > 0 {
		h.db.Model(&models.ExpenseSplit{}).
			Where("member_id IN ?", currentMemberIDs).
			Pluck("expense_id", &splitExpenseIDs)
	}

	// Visibility rules:
	// - Own unsplit expenses (created by this user, not shared)
	// - Split expenses where this user is the creator (payer)
	// - Split expenses where this user has a split record (recipient)
	var expenses []models.Expense
	visibilityClause := h.db.Where(
		"(user_id = ? AND is_split = false) OR (is_split = true AND (user_id = ? OR (paid_by_member_id IN ? AND is_split = true) OR id IN ?))",
		userID, userID, currentMemberIDs, splitExpenseIDs,
	)
	query := visibilityClause.
		Preload("Category").
		Preload("Property").
		Preload("Subcategory").
		Preload("Project").
		Preload("PaidBy").
		Preload("Splits").
		Preload("Splits.Member")

	// Apply optional filters
	if categoryID := c.Query("category_id"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	if propertyID := c.Query("property_id"); propertyID != "" {
		query = query.Where("property_id = ?", propertyID)
	}

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// Parse date filters once — reused for both query and countQuery
	var parsedFrom, parsedTo *time.Time
	if from := c.Query("from"); from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' date format. Use YYYY-MM-DD"})
			return
		}
		parsedFrom = &t
		query = query.Where("date >= ?", t)
	}

	if to := c.Query("to"); to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' date format. Use YYYY-MM-DD"})
			return
		}
		endOfDay := t.AddDate(0, 0, 1)
		parsedTo = &endOfDay
		query = query.Where("date < ?", endOfDay)
	}

	if search := c.Query("search"); search != "" {
		query = query.Where("description LIKE ?", "%"+search+"%")
	}

	// Filter: only unsettled split expenses
	unsettledOnly := c.Query("unsettled_only") == "true"
	if unsettledOnly {
		query = query.Where("is_split = ? AND EXISTS (SELECT 1 FROM expense_splits WHERE expense_splits.expense_id = expenses.id AND expense_splits.is_settled = ?)", true, false)
	}

	// Pagination
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get total count before pagination (same visibility + filters)
	var total int64
	countQuery := h.db.Model(&models.Expense{}).Where(
		"(user_id = ? AND is_split = false) OR (is_split = true AND (user_id = ? OR (paid_by_member_id IN ? AND is_split = true) OR id IN ?))",
		userID, userID, currentMemberIDs, splitExpenseIDs,
	)
	if categoryID := c.Query("category_id"); categoryID != "" {
		countQuery = countQuery.Where("category_id = ?", categoryID)
	}
	if propertyID := c.Query("property_id"); propertyID != "" {
		countQuery = countQuery.Where("property_id = ?", propertyID)
	}
	if projectID := c.Query("project_id"); projectID != "" {
		countQuery = countQuery.Where("project_id = ?", projectID)
	}
	if parsedFrom != nil {
		countQuery = countQuery.Where("date >= ?", *parsedFrom)
	}
	if parsedTo != nil {
		countQuery = countQuery.Where("date < ?", *parsedTo)
	}
	if search := c.Query("search"); search != "" {
		countQuery = countQuery.Where("description LIKE ?", "%"+search+"%")
	}
	if unsettledOnly {
		countQuery = countQuery.Where("is_split = ? AND id IN (SELECT expense_id FROM expense_splits WHERE is_settled = ?)", true, false)
	}
	countQuery.Count(&total)

	// Ordering — driven by client sort param
	orderClause := "date DESC, id DESC"
	switch c.Query("sort") {
	case "date_asc":
		orderClause = "date ASC, id ASC"
	case "amount_desc":
		orderClause = "amount DESC, date DESC"
	case "amount_asc":
		orderClause = "amount ASC, date DESC"
	case "description_asc":
		orderClause = "description ASC"
	}

	if err := query.Order(orderClause).Limit(limit).Offset(offset).Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expenses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"expenses": expenses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// Create creates a new expense
// POST /api/v1/expenses
func (h *ExpenseHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ EXPENSE CREATE: Validation error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// LOG REQUEST DATA
	log.Printf("📝 EXPENSE CREATE REQUEST:")
	log.Printf("   UserID: %d", userID)
	log.Printf("   Amount: %.2f", req.Amount)
	log.Printf("   Description: %s", req.Description)
	log.Printf("   CategoryID: %d", req.CategoryID)
	log.Printf("   PropertyID: %v", req.PropertyID)
	log.Printf("   PaidByMemberID: %d", req.PaidByMemberID)
	log.Printf("   IsSplit: %t", req.IsSplit)
	log.Printf("   SplitWithMemberIDs: %v", req.SplitWithMemberIDs)

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	// PaidByMemberID must be provided (frontend should resolve current user's member ID)
	paidByMemberID := req.PaidByMemberID
	if paidByMemberID == 0 {
		// Try to find the current user's member ID for this property
		if req.PropertyID != nil {
			var member models.HouseholdMember
			if err := h.db.Where("property_id = ? AND user_id = ?", *req.PropertyID, userID).First(&member).Error; err == nil {
				paidByMemberID = member.ID
				log.Printf("   ℹ️ Auto-resolved PaidByMemberID: %d", paidByMemberID)
			} else {
				log.Printf("❌ EXPENSE CREATE: Could not find member for user %d in property %d", userID, *req.PropertyID)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Could not find your member profile for this property"})
				return
			}
		} else {
			log.Printf("❌ EXPENSE CREATE: PaidByMemberID required when PropertyID is provided")
			c.JSON(http.StatusBadRequest, gin.H{"error": "paid_by_member_id is required"})
			return
		}
	}

	// Verify category belongs to user
	var category models.Category
	if err := h.db.Where("id = ? AND (user_id = ? OR user_id IS NULL)", req.CategoryID, userID).First(&category).Error; err != nil {
		log.Printf("❌ EXPENSE CREATE: Invalid category %d for user %d", req.CategoryID, userID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	// Verify user is a member of the property (if provided)
	if req.PropertyID != nil {
		var member models.HouseholdMember
		if err := h.db.Where("property_id = ? AND user_id = ?", *req.PropertyID, userID).First(&member).Error; err != nil {
			log.Printf("❌ EXPENSE CREATE: User %d is not a member of property %d", userID, *req.PropertyID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of this property"})
			return
		}
	}

	// Verify project is accessible to user (owner or shared member)
	if req.ProjectID != nil {
		var project models.Project
		if err := h.db.Where(
			"id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))",
			*req.ProjectID, userID, userID,
		).First(&project).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project"})
			return
		}
	}

	// Start transaction
	tx := h.db.Begin()

	expense := models.Expense{
		UserID:         userID,
		Amount:         req.Amount,
		Description:    req.Description,
		CategoryID:     req.CategoryID,
		PropertyID:     req.PropertyID,
		SubcategoryID:  req.SubcategoryID,
		ProjectID:      req.ProjectID,
		Date:           date,
		AttachmentURL:  req.AttachmentURL,
		PaidByMemberID: paidByMemberID,
		IsSplit:        req.IsSplit,
	}

	if err := tx.Create(&expense).Error; err != nil {
		tx.Rollback()
		log.Printf("❌ EXPENSE CREATE: Failed to save expense: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create expense", "details": err.Error()})
		return
	}

	log.Printf("✅ EXPENSE CREATED: ID=%d, Amount=%.2f, PropertyID=%v", expense.ID, expense.Amount, expense.PropertyID)

	// If split is enabled and we have members to split with, create ExpenseSplits
	if req.IsSplit && len(req.SplitWithMemberIDs) > 0 {
		totalPeople := len(req.SplitWithMemberIDs) + 1 // +1 for the payer
		splitAmount := req.Amount / float64(totalPeople)

		log.Printf("💰 SPLIT MODE:")
		log.Printf("   Total People: %d", totalPeople)
		log.Printf("   Split Amount: %.2f each", splitAmount)
		log.Printf("   Payer MemberID: %d", paidByMemberID)
		log.Printf("   Split With MemberIDs: %v", req.SplitWithMemberIDs)

		// Create split for the payer (already settled)
		now := time.Now()
		payerSplit := models.ExpenseSplit{
			ExpenseID: expense.ID,
			MemberID:  paidByMemberID,
			Amount:    splitAmount,
			IsSettled: true,
			SettledAt: &now,
		}
		if err := tx.Create(&payerSplit).Error; err != nil {
			tx.Rollback()
			log.Printf("❌ SPLIT: Failed to create payer split: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payer split"})
			return
		}
		log.Printf("   ✅ Payer split created: MemberID=%d, Amount=%.2f, IsSettled=true", paidByMemberID, splitAmount)

		// Create splits for other members (not settled)
		for _, otherMemberID := range req.SplitWithMemberIDs {
			split := models.ExpenseSplit{
				ExpenseID: expense.ID,
				MemberID:  otherMemberID,
				Amount:    splitAmount,
				IsSettled: false,
			}
			if err := tx.Create(&split).Error; err != nil {
				tx.Rollback()
				log.Printf("❌ SPLIT: Failed to create split for member %d: %v", otherMemberID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create split"})
				return
			}
			log.Printf("   ✅ Split created: MemberID=%d, Amount=%.2f, IsSettled=false", otherMemberID, splitAmount)
		}

		log.Printf("✅ ALL SPLITS CREATED SUCCESSFULLY")
	} else {
		log.Printf("ℹ️  No split requested (IsSplit=%t, SplitWithMemberIDs count=%d)", req.IsSplit, len(req.SplitWithMemberIDs))
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Printf("❌ EXPENSE CREATE: Transaction commit failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save expense"})
		return
	}

	log.Printf("✅ EXPENSE TRANSACTION COMMITTED")

	// Reload with relations
	h.db.Preload("Category").
		Preload("Property").
		Preload("Subcategory").
		Preload("Project").
		Preload("PaidBy").
		Preload("Splits").
		Preload("Splits.Member").
		First(&expense, expense.ID)

	c.JSON(http.StatusCreated, expense)
}

// Get returns a single expense by ID
// GET /api/v1/expenses/:id
func (h *ExpenseHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expense ID"})
		return
	}

	// Find property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var expense models.Expense
	if err := h.db.Where("id = ? AND property_id IN ?", id, memberPropertyIDs).
		Preload("Category").
		Preload("Property").
		Preload("Subcategory").
		Preload("Project").
		First(&expense).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expense"})
		}
		return
	}

	c.JSON(http.StatusOK, expense)
}

// Update updates an existing expense
// PUT /api/v1/expenses/:id
func (h *ExpenseHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expense ID"})
		return
	}

	// Find existing expense - only the creator can update
	var expense models.Expense
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&expense).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found or you are not the owner"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expense"})
		}
		return
	}

	// Block editing of auto-created expenses (must be managed via the bill)
	if expense.BillID != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Questa spesa è stata creata automaticamente dal pagamento di una bolletta. Per modificarla, aggiorna la relativa bolletta dalla sezione Utenze.",
		})
		return
	}

	// For split expenses: determine if fully settled (all splits paid)
	// Non-split expenses are never "settled" and always fully editable
	settled := false
	if expense.IsSplit {
		var unsettled int64
		h.db.Model(&models.ExpenseSplit{}).
			Where("expense_id = ? AND is_settled = false", expense.ID).
			Count(&unsettled)
		settled = (unsettled == 0)
	}

	var req UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build updates map
	updates := make(map[string]interface{})

	// Settled split expenses: only description, category and subcategory can be changed
	if !settled {
		if req.Amount != nil {
			if *req.Amount <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
				return
			}
			updates["amount"] = *req.Amount
		}

		if req.PropertyID != nil {
			var member models.HouseholdMember
			if err := h.db.Where("property_id = ? AND user_id = ?", *req.PropertyID, userID).First(&member).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of this property"})
				return
			}
			updates["property_id"] = *req.PropertyID
		}

		if req.ProjectID != nil {
			var project models.Project
			if err := h.db.Where(
				"id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))",
				*req.ProjectID, userID, userID,
			).First(&project).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project"})
				return
			}
			updates["project_id"] = *req.ProjectID
		}

		if req.Date != nil {
			date, err := time.Parse("2006-01-02", *req.Date)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
				return
			}
			updates["date"] = date
		}

		if req.AttachmentURL != nil {
			updates["attachment_url"] = *req.AttachmentURL
		}
	}

	// Always editable regardless of settlement status: description, category, subcategory
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if req.CategoryID != nil {
		var category models.Category
		if err := h.db.Where("id = ? AND (is_default = true OR user_id = ?)", *req.CategoryID, userID).First(&category).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
			return
		}
		updates["category_id"] = *req.CategoryID
	}

	if req.SubcategoryID != nil {
		updates["subcategory_id"] = *req.SubcategoryID
	}

	// Apply updates
	if len(updates) > 0 {
		if err := h.db.Model(&expense).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
			return
		}
	}

	// Reload with relations
	h.db.Preload("Category").
		Preload("Property").
		Preload("Subcategory").
		Preload("Project").
		First(&expense, expense.ID)

	c.JSON(http.StatusOK, expense)
}

// Delete deletes an expense (soft delete)
// DELETE /api/v1/expenses/:id
func (h *ExpenseHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expense ID"})
		return
	}

	// Only the creator can delete the expense
	var expense models.Expense
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found or you are not the owner"})
		return
	}

	// Block manual deletion of auto-created expenses (must be managed via the bill)
	if expense.BillID != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Questa spesa è stata creata automaticamente dal pagamento di una bolletta. Per eliminarla, elimina la relativa bolletta dalla sezione Utenze.",
		})
		return
	}

	// Block deletion of fully-settled split expenses
	if expense.IsSplit {
		var unsettled int64
		h.db.Model(&models.ExpenseSplit{}).
			Where("expense_id = ? AND is_settled = false", expense.ID).
			Count(&unsettled)
		if unsettled == 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Questa spesa è già stata completamente saldata e non può essere eliminata.",
			})
			return
		}
	}

	if err := h.db.Delete(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete expense"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Expense deleted successfully"})
}

// GetStats returns statistics for charts
// GET /api/v1/expenses/stats?period=6m&property_id=1&year=2024
func (h *ExpenseHandler) GetStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Find property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Determine date range
	now := time.Now()
	year := now.Year()
	if y := c.Query("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	// Default period is current year
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	// Handle period parameter (e.g., "6m" for last 6 months)
	if period := c.Query("period"); period != "" {
		switch period {
		case "1m":
			startDate = now.AddDate(0, -1, 0)
			endDate = now
		case "3m":
			startDate = now.AddDate(0, -3, 0)
			endDate = now
		case "6m":
			startDate = now.AddDate(0, -6, 0)
			endDate = now
		case "12m":
			startDate = now.AddDate(-1, 0, 0)
			endDate = now
		}
	}

	// Explicit from/to override year and period
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			startDate = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			endDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
		}
	}

	// "all=true" overrides everything: use first expense date → today
	if c.Query("all") == "true" {
		var minDateStr string
		h.db.Model(&models.Expense{}).
			Where("property_id IN ?", memberPropertyIDs).
			Select("strftime('%Y-%m-%d', MIN(date))").
			Scan(&minDateStr)
		if minDateStr != "" {
			if t, err := time.Parse("2006-01-02", minDateStr); err == nil {
				startDate = t
			}
		}
		endDate = now
	}

	// Build reusable WHERE conditions shared across all sub-queries
	baseWhere := "property_id IN ? AND date >= ? AND date <= ?"
	joinWhere := "expenses.property_id IN ? AND expenses.date >= ? AND expenses.date <= ?"
	baseArgs := []interface{}{memberPropertyIDs, startDate, endDate}

	categoryID := c.Query("category_id")
	if categoryID != "" {
		baseWhere += " AND category_id = ?"
		joinWhere += " AND expenses.category_id = ?"
		baseArgs = append(baseArgs, categoryID)
	}
	if propertyID := c.Query("property_id"); propertyID != "" {
		baseWhere += " AND property_id = ?"
		joinWhere += " AND expenses.property_id = ?"
		baseArgs = append(baseArgs, propertyID)
	}

	// Determine trend granularity from date range
	dayRange := int(endDate.Sub(startDate).Hours() / 24)
	granularity := "month"
	if dayRange <= 31 {
		granularity = "day"
	} else if dayRange > 365 {
		granularity = "quarter"
	}

	itMonths := []string{"", "Gen", "Feb", "Mar", "Apr", "Mag", "Giu", "Lug", "Ago", "Set", "Ott", "Nov", "Dic"}
	var trend []TrendPoint

	switch granularity {
	case "day":
		var rows []struct {
			Day    int     `json:"day"`
			Month  int     `json:"month"`
			Year   int     `json:"year"`
			Amount float64 `json:"amount"`
			Count  int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST(strftime('%d', date) AS INTEGER) as day, CAST(strftime('%m', date) AS INTEGER) as month, CAST(strftime('%Y', date) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("strftime('%Y-%m-%d', date)").
			Order("year, month, day").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			mon := ""
			if r.Month >= 1 && r.Month <= 12 {
				mon = itMonths[r.Month]
			}
			trend[i] = TrendPoint{Label: fmt.Sprintf("%d %s", r.Day, mon), Amount: r.Amount, Count: r.Count}
		}

	case "quarter":
		var rows []struct {
			Quarter int     `json:"quarter"`
			Year    int     `json:"year"`
			Amount  float64 `json:"amount"`
			Count   int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST((CAST(strftime('%m', date) AS INTEGER) + 2) / 3 AS INTEGER) as quarter, CAST(strftime('%Y', date) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("year, quarter").
			Order("year, quarter").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			trend[i] = TrendPoint{Label: fmt.Sprintf("Q%d %d", r.Quarter, r.Year), Amount: r.Amount, Count: r.Count}
		}

	default: // month
		var rows []struct {
			Month  int     `json:"month"`
			Year   int     `json:"year"`
			Amount float64 `json:"amount"`
			Count  int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST(strftime('%m', date) AS INTEGER) as month, CAST(strftime('%Y', date) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("strftime('%Y-%m', date)").
			Order("year, month").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			mon := ""
			if r.Month >= 1 && r.Month <= 12 {
				mon = itMonths[r.Month]
			}
			trend[i] = TrendPoint{Label: fmt.Sprintf("%s %d", mon, r.Year), Amount: r.Amount, Count: r.Count}
		}
	}

	// Category/Subcategory aggregation
	var totalAmount float64
	var byCategory []CategoryStats

	if categoryID != "" {
		// Subcategory breakdown when filtering by category
		var subResults []struct {
			SubcategoryID *uint   `json:"subcategory_id"`
			CategoryName  string  `json:"category_name"`
			Amount        float64 `json:"amount"`
			Count         int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("expenses.subcategory_id, COALESCE(subcategories.name, 'Senza sottocategoria') as category_name, SUM(expenses.amount) as amount, COUNT(*) as count").
			Joins("LEFT JOIN subcategories ON subcategories.id = expenses.subcategory_id").
			Where(joinWhere, baseArgs...).
			Group("expenses.subcategory_id").
			Order("amount DESC").
			Scan(&subResults)

		for _, r := range subResults {
			totalAmount += r.Amount
		}
		byCategory = make([]CategoryStats, len(subResults))
		for i, r := range subResults {
			percentage := 0.0
			if totalAmount > 0 {
				percentage = (r.Amount / totalAmount) * 100
			}
			id := uint(0)
			if r.SubcategoryID != nil {
				id = *r.SubcategoryID
			}
			byCategory[i] = CategoryStats{
				CategoryID:   id,
				CategoryName: r.CategoryName,
				Amount:       r.Amount,
				Count:        r.Count,
				Percentage:   percentage,
			}
		}
	} else {
		var categoryResults []struct {
			CategoryID   uint    `json:"category_id"`
			CategoryName string  `json:"category_name"`
			Amount       float64 `json:"amount"`
			Count        int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("expenses.category_id, categories.name as category_name, SUM(expenses.amount) as amount, COUNT(*) as count").
			Joins("JOIN categories ON categories.id = expenses.category_id").
			Where(joinWhere, baseArgs...).
			Group("expenses.category_id").
			Order("amount DESC").
			Scan(&categoryResults)

		for _, r := range categoryResults {
			totalAmount += r.Amount
		}
		byCategory = make([]CategoryStats, len(categoryResults))
		for i, r := range categoryResults {
			percentage := 0.0
			if totalAmount > 0 {
				percentage = (r.Amount / totalAmount) * 100
			}
			byCategory[i] = CategoryStats{
				CategoryID:   r.CategoryID,
				CategoryName: r.CategoryName,
				Amount:       r.Amount,
				Count:        r.Count,
				Percentage:   percentage,
			}
		}
	}

	// Current month total (respects category filter)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	monthWhere := "property_id IN ? AND date >= ? AND date <= ?"
	monthArgs := []interface{}{memberPropertyIDs, monthStart, monthEnd}
	if categoryID != "" {
		monthWhere += " AND category_id = ?"
		monthArgs = append(monthArgs, categoryID)
	}
	var totalMonth float64
	h.db.Model(&models.Expense{}).
		Where(monthWhere, monthArgs...).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalMonth)

	// Year total (respects category filter)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.UTC)
	yearWhere := "property_id IN ? AND date >= ? AND date <= ?"
	yearArgs := []interface{}{memberPropertyIDs, yearStart, yearEnd}
	if categoryID != "" {
		yearWhere += " AND category_id = ?"
		yearArgs = append(yearArgs, categoryID)
	}
	var totalYear float64
	h.db.Model(&models.Expense{}).
		Where(yearWhere, yearArgs...).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalYear)

	// Average per period point (day/month/quarter)
	var avgMonth float64
	if len(trend) > 0 {
		avgMonth = totalAmount / float64(len(trend))
	}

	c.JSON(http.StatusOK, gin.H{
		"trend":          trend,
		"granularity":    granularity,
		"by_category":    byCategory,
		"is_subcategory": categoryID != "",
		"total_month":    totalMonth,
		"total_year":     totalYear,
		"total_period":   totalAmount,
		"average_month":  avgMonth,
		"period": gin.H{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
	})
}
