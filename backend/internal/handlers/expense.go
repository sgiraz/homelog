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
	Amount             float64  `json:"amount" binding:"required,gt=0"`
	OriginalAmount     *float64 `json:"original_amount"`
	OriginalCurrency   string   `json:"original_currency"`
	Description        string   `json:"description"`
	CategoryID         uint     `json:"category_id" binding:"required"`
	PropertyID         *uint    `json:"property_id"`
	SubcategoryID      *uint    `json:"subcategory_id"`
	ProjectID          *uint    `json:"project_id"`
	Date               string   `json:"date" binding:"required"` // Format: YYYY-MM-DD
	AttachmentURL      string   `json:"attachment_url"`
	PaidByMemberID     uint     `json:"paid_by_member_id"`
	IsSplit            bool     `json:"is_split"`
	SplitWithMemberIDs []uint   `json:"split_with_member_ids"`
}

// UpdateExpenseRequest represents the request body for updating an expense
type UpdateExpenseRequest struct {
	Amount           *float64 `json:"amount"`
	OriginalAmount   *float64 `json:"original_amount"`
	OriginalCurrency *string  `json:"original_currency"`
	Description      *string  `json:"description"`
	CategoryID       *uint    `json:"category_id"`
	PropertyID       *uint    `json:"property_id"`
	SubcategoryID    *uint    `json:"subcategory_id"`
	ProjectID        *uint    `json:"project_id"`
	Date             *string  `json:"date"`
	AttachmentURL    *string  `json:"attachment_url"`
}

// MonthlyStats represents monthly expense statistics
type MonthlyStats struct {
	Month  string  `json:"month"`
	Year   int     `json:"year"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
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

	// Parse date filters once — reused for both query and countQuery.
	// Compare against the date-only part via SQLite's date() function so the
	// comparison is independent of the time component and timezone offset
	// that GORM/glebarez may serialize into the stored RFC3339 value.
	var fromDate, toDate string
	if from := c.Query("from"); from != "" {
		if _, err := time.Parse("2006-01-02", from); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' date format. Use YYYY-MM-DD"})
			return
		}
		fromDate = from
		query = query.Where("date(date) >= ?", fromDate)
	}

	if to := c.Query("to"); to != "" {
		if _, err := time.Parse("2006-01-02", to); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' date format. Use YYYY-MM-DD"})
			return
		}
		toDate = to
		query = query.Where("date(date) <= ?", toDate)
	}

	if search := c.Query("search"); search != "" {
		query = query.Where("description LIKE ?", "%"+search+"%")
	}

	// Filter: only unsettled split expenses
	unsettledOnly := c.Query("unsettled_only") == "true"
	if unsettledOnly {
		query = query.Where("is_split = ? AND EXISTS (SELECT 1 FROM expense_splits WHERE expense_splits.expense_id = expenses.id AND expense_splits.is_settled = ?)", true, false)
	}

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
	if fromDate != "" {
		countQuery = countQuery.Where("date(date) >= ?", fromDate)
	}
	if toDate != "" {
		countQuery = countQuery.Where("date(date) <= ?", toDate)
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

	log.Printf("📝 EXPENSE CREATE REQUEST:")
	log.Printf("   UserID: %d", userID)
	log.Printf("   Amount: %.2f", req.Amount)
	log.Printf("   Description: %s", req.Description)
	log.Printf("   CategoryID: %d", req.CategoryID)
	log.Printf("   PropertyID: %v", req.PropertyID)
	log.Printf("   PaidByMemberID: %d", req.PaidByMemberID)
	log.Printf("   IsSplit: %t", req.IsSplit)
	log.Printf("   SplitWithMemberIDs: %v", req.SplitWithMemberIDs)

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

	tx := h.db.Begin()

	expense := models.Expense{
		UserID:           userID,
		Amount:           req.Amount,
		OriginalAmount:   req.OriginalAmount,
		OriginalCurrency: req.OriginalCurrency,
		Description:      req.Description,
		CategoryID:       req.CategoryID,
		PropertyID:       req.PropertyID,
		SubcategoryID:    req.SubcategoryID,
		ProjectID:        req.ProjectID,
		Date:             date,
		AttachmentURL:    req.AttachmentURL,
		PaidByMemberID:   paidByMemberID,
		IsSplit:          req.IsSplit,
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
			ExpenseID:     expense.ID,
			MemberID:      paidByMemberID,
			Amount:        splitAmount,
			SettledAmount: splitAmount,
			IsSettled:     true,
			SettledAt:     &now,
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

	if err := tx.Commit().Error; err != nil {
		log.Printf("❌ EXPENSE CREATE: Transaction commit failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save expense"})
		return
	}

	log.Printf("✅ EXPENSE TRANSACTION COMMITTED")

	// Notify members involved in the split (except the creator)
	if req.IsSplit && len(req.SplitWithMemberIDs) > 0 {
		var payer models.HouseholdMember
		h.db.First(&payer, paidByMemberID)
		splitAmount := req.Amount / float64(len(req.SplitWithMemberIDs)+1)

		for _, memberID := range req.SplitWithMemberIDs {
			var member models.HouseholdMember
			h.db.First(&member, memberID)
			if member.UserID == nil || *member.UserID == userID {
				continue
			}
			relID := expense.ID
			createNotification(h.db, *member.UserID, "expense_shared",
				fmt.Sprintf("Nuova spesa condivisa: %s", req.Description),
				fmt.Sprintf("%s ha inserito una spesa. La tua quota: %.2f.", payer.Name, splitAmount),
				&relID, expense.PropertyID)
		}
	}

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

	updates := make(map[string]any)

	// Guardrail for split-amount propagation: reject if any non-payer split has
	// already received money (fully or partially settled via the settlement
	// ledger), otherwise the new per-person quota would silently rewrite
	// historical balances out from under existing payments.
	amountChanging := !settled && req.Amount != nil && *req.Amount != expense.Amount
	if amountChanging && expense.IsSplit {
		var settledNonPayer int64
		h.db.Model(&models.ExpenseSplit{}).
			Where("expense_id = ? AND member_id <> ? AND settled_amount > 0", expense.ID, expense.PaidByMemberID).
			Count(&settledNonPayer)
		if settledNonPayer > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Alcune quote sono già state saldate. Annulla i pagamenti dal Bilancio per modificare l'importo."})
			return
		}
	}

	// Settled split expenses: only description, category and subcategory can be changed
	if !settled {
		if req.Amount != nil {
			if *req.Amount <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
				return
			}
			updates["amount"] = *req.Amount
		}
		if req.OriginalAmount != nil {
			updates["original_amount"] = *req.OriginalAmount
		}
		if req.OriginalCurrency != nil {
			updates["original_currency"] = *req.OriginalCurrency
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

	// Apply updates atomically — if amount is changing on a split expense we
	// also need to recompute every split's quota in the same transaction so
	// Bilancio stays consistent with the new total.
	if len(updates) > 0 || (amountChanging && expense.IsSplit) {
		err := h.db.Transaction(func(tx *gorm.DB) error {
			if len(updates) > 0 {
				if err := tx.Model(&expense).Updates(updates).Error; err != nil {
					return err
				}
			}
			if amountChanging && expense.IsSplit {
				var count int64
				if err := tx.Model(&models.ExpenseSplit{}).
					Where("expense_id = ?", expense.ID).
					Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					splitAmount := *req.Amount / float64(count)
					if err := tx.Model(&models.ExpenseSplit{}).
						Where("expense_id = ?", expense.ID).
						Update("amount", splitAmount).Error; err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
			return
		}
	}

	h.db.Preload("Category").
		Preload("Property").
		Preload("Subcategory").
		Preload("Project").
		Preload("Splits").
		Preload("Splits.Member").
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

	// Block deletion once any non-payer share has received money (fully or
	// partially, via the settlement ledger) — deleting would silently erase
	// the debt those payments were made against.
	if expense.IsSplit {
		var settledNonPayer int64
		h.db.Model(&models.ExpenseSplit{}).
			Where("expense_id = ? AND member_id <> ? AND settled_amount > 0", expense.ID, expense.PaidByMemberID).
			Count(&settledNonPayer)
		if settledNonPayer > 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Questa spesa ha quote già saldate (anche parzialmente) e non può essere eliminata. Annulla i pagamenti dal Bilancio prima di eliminarla.",
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
