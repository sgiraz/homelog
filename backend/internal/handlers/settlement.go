package handlers

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// settlementEpsilon absorbs float64 rounding noise when comparing amounts.
const settlementEpsilon = 0.005

// Allocation kinds — see models.SettlementAllocation.
const (
	allocationKindPayment = "payment" // reduces the debt this settlement pays
	allocationKindFunding = "funding" // the credit consumed to fund a compensation
)

// paymentMethodCompensation marks a settlement where no money moved: a credit
// the debtor was owed was applied to their debt instead of being reimbursed.
const paymentMethodCompensation = "compensation"

// SettlementHandler handles settlement operations
type SettlementHandler struct {
	db *gorm.DB
}

// NewSettlementHandler creates a new settlement handler
func NewSettlementHandler(db *gorm.DB) *SettlementHandler {
	return &SettlementHandler{db: db}
}

// applyAllocation records how much of a settlement covered a given split and
// moves that split's settled amount by the same figure, flipping it to fully
// settled once nothing is left owed.
func applyAllocation(tx *gorm.DB, settlementID uint, split models.ExpenseSplit, amount float64, kind string, now time.Time) error {
	allocation := models.SettlementAllocation{
		SettlementID:   settlementID,
		ExpenseSplitID: split.ID,
		Amount:         amount,
		Kind:           kind,
	}
	if err := tx.Create(&allocation).Error; err != nil {
		return err
	}

	newSettledAmount := split.SettledAmount + amount
	updates := map[string]any{
		"settled_amount": newSettledAmount,
		"settlement_id":  settlementID,
	}
	if newSettledAmount >= split.Amount-settlementEpsilon {
		updates["is_settled"] = true
		updates["settled_at"] = now
	}
	return tx.Model(&models.ExpenseSplit{}).Where("id = ?", split.ID).Updates(updates).Error
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
	// TargetExpenseID aims the payment at a single long-term debt. When absent
	// the payment settles the ordinary running balance instead.
	TargetExpenseID *uint `json:"target_expense_id"`
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

	// Fetch outstanding splits between the pair, oldest expense first — the
	// payment is applied as a ledger, oldest debt paid down before newer ones.
	query := h.db.
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expense_splits.is_settled = ?", false).
		Where("expenses.property_id = ?", req.PropertyID).
		Where("expenses.deleted_at IS NULL")

	if req.TargetExpenseID != nil {
		// Aimed at one long-term debt: only that expense's share, and only in
		// the direction the payer actually owes — paying Alice must not clear
		// what Alice owes you.
		var target models.Expense
		if err := h.db.First(&target, *req.TargetExpenseID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Debito non trovato"})
			return
		}
		if !target.IsLongTermDebt {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Questa spesa non è un debito a lungo termine"})
			return
		}
		query = query.
			Where("expenses.id = ?", target.ID).
			Where("expenses.paid_by_member_id = ? AND expense_splits.member_id = ?", req.ToMemberID, req.FromMemberID)
	} else {
		// Ordinary balance: long-term debts are repaid explicitly, never swept
		// up by a generic settlement.
		query = query.
			Where("expenses.is_long_term_debt = ?", false).
			Where("(expenses.paid_by_member_id = ? AND expense_splits.member_id = ?) OR (expenses.paid_by_member_id = ? AND expense_splits.member_id = ?)",
				req.ToMemberID, req.FromMemberID, req.FromMemberID, req.ToMemberID)
	}

	var splits []models.ExpenseSplit
	if err := query.Order("expenses.date ASC").Find(&splits).Error; err != nil {
		log.Printf("ERROR finding expense splits to settle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find expense splits"})
		return
	}

	var totalOwed float64
	for _, s := range splits {
		totalOwed += s.Amount - s.SettledAmount
	}
	if req.Amount > totalOwed+settlementEpsilon {
		c.JSON(http.StatusBadRequest, gin.H{"error": "L'importo supera il debito residuo tra questi membri"})
		return
	}

	// Start transaction
	tx := h.db.Begin()

	// Create Settlement record
	settlement := models.Settlement{
		PropertyID:      req.PropertyID,
		FromMemberID:    req.FromMemberID,
		ToMemberID:      req.ToMemberID,
		Amount:          req.Amount,
		Date:            date,
		PaymentMethod:   req.PaymentMethod,
		Note:            req.Note,
		TargetExpenseID: req.TargetExpenseID,
	}

	if err := tx.Create(&settlement).Error; err != nil {
		tx.Rollback()
		log.Printf("ERROR creating settlement: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create settlement"})
		return
	}

	log.Printf("Settlement created - ID: %d, From: %d, To: %d, Amount: %.2f",
		settlement.ID, req.FromMemberID, req.ToMemberID, req.Amount)

	// Apply the payment against the outstanding splits, oldest first. Each
	// split touched gets a SettlementAllocation recording exactly how much of
	// this settlement covered it; a split not fully covered stays unsettled
	// with a reduced remaining balance instead of being force-marked settled.
	now := time.Now()
	remaining := req.Amount
	touched := 0
	for i := range splits {
		if remaining <= settlementEpsilon {
			break
		}
		owed := splits[i].Amount - splits[i].SettledAmount
		if owed <= settlementEpsilon {
			continue
		}
		alloc := math.Min(owed, remaining)

		if err := applyAllocation(tx, settlement.ID, splits[i], alloc, allocationKindPayment, now); err != nil {
			tx.Rollback()
			log.Printf("ERROR allocating settlement to split %d: %v", splits[i].ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to allocate settlement"})
			return
		}

		remaining -= alloc
		touched++
	}

	log.Printf("✅ SETTLEMENT: Allocated %.2f across %d splits", req.Amount-remaining, touched)

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
		Preload("Allocations").
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

	// Reverse only this settlement's own allocations — a split may have
	// received contributions from other settlements too (before or after this
	// one), which must stay untouched.
	var allocations []models.SettlementAllocation
	if err := tx.Where("settlement_id = ?", settlement.ID).Find(&allocations).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load settlement allocations"})
		return
	}

	for _, alloc := range allocations {
		var split models.ExpenseSplit
		if err := tx.First(&split, alloc.ExpenseSplitID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load expense split"})
			return
		}

		newSettledAmount := split.SettledAmount - alloc.Amount
		if newSettledAmount < settlementEpsilon {
			newSettledAmount = 0
		}
		if err := tx.Model(&models.ExpenseSplit{}).Where("id = ?", split.ID).Updates(map[string]any{
			"settled_amount": newSettledAmount,
			"is_settled":     false,
			"settled_at":     nil,
			"settlement_id":  nil,
		}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsettle expense split"})
			return
		}
	}

	if err := tx.Where("settlement_id = ?", settlement.ID).Delete(&models.SettlementAllocation{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete settlement allocations"})
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

// CompensateRequest applies a credit the debtor is owed against one of their
// long-term debts, instead of being reimbursed for it in cash.
type CompensateRequest struct {
	PropertyID      uint   `json:"property_id" binding:"required"`
	SourceSplitID   uint   `json:"source_split_id" binding:"required"`
	TargetExpenseID uint   `json:"target_expense_id" binding:"required"`
	Date            string `json:"date"`
	Note            string `json:"note"`
}

// Compensate offsets an outstanding credit against a long-term debt
// POST /api/v1/settlements/compensate
//
// No money moves: the share the counterpart owed on an ordinary expense is
// cancelled, and the same figure comes off the debt owed back to them.
func (h *SettlementHandler) Compensate(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CompensateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !requirePropertyMember(c, h.db, userID, req.PropertyID) {
		return
	}

	// The credit being spent: `source.MemberID` owes this share to whoever paid
	// the expense, and it's that payer's own debt we're about to shrink.
	var source models.ExpenseSplit
	if err := h.db.Preload("Expense").First(&source, req.SourceSplitID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Spesa a credito non trovata"})
		return
	}
	if source.Expense.PropertyID == nil || *source.Expense.PropertyID != req.PropertyID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La spesa non appartiene a questa casa"})
		return
	}
	if source.Expense.IsLongTermDebt {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Un debito a lungo termine non può finanziare una compensazione"})
		return
	}
	sourceRemaining := source.Amount - source.SettledAmount
	if sourceRemaining <= settlementEpsilon {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Questa quota è già stata saldata"})
		return
	}

	// debtor repays their own debt using the credit; holder is owed both.
	debtorID := source.Expense.PaidByMemberID
	holderID := source.MemberID

	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", req.PropertyID, userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be a member of this property"})
		return
	}
	if currentMember.ID != debtorID && currentMember.ID != holderID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must be involved in the compensation"})
		return
	}

	// The debt to shrink: same pair, opposite direction, flagged long-term.
	var debt models.ExpenseSplit
	if err := h.db.
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expenses.id = ?", req.TargetExpenseID).
		Where("expenses.is_long_term_debt = ?", true).
		Where("expenses.deleted_at IS NULL").
		Where("expenses.paid_by_member_id = ? AND expense_splits.member_id = ?", holderID, debtorID).
		First(&debt).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Debito a lungo termine non trovato per questa coppia"})
		return
	}
	debtRemaining := debt.Amount - debt.SettledAmount
	if debtRemaining <= settlementEpsilon {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Questo debito è già estinto"})
		return
	}

	amount := math.Min(sourceRemaining, debtRemaining)

	date := time.Now()
	if req.Date != "" {
		parsed, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
			return
		}
		date = parsed
	}

	tx := h.db.Begin()

	targetExpenseID := req.TargetExpenseID
	settlement := models.Settlement{
		PropertyID:      req.PropertyID,
		FromMemberID:    debtorID,
		ToMemberID:      holderID,
		Amount:          amount,
		Date:            date,
		PaymentMethod:   paymentMethodCompensation,
		Note:            req.Note,
		TargetExpenseID: &targetExpenseID,
	}
	if err := tx.Create(&settlement).Error; err != nil {
		tx.Rollback()
		log.Printf("ERROR creating compensation settlement: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create compensation"})
		return
	}

	now := time.Now()
	// Both sides move by the same figure: the debt shrinks, and the credit that
	// paid for it is consumed. Delete() reverses both together.
	if err := applyAllocation(tx, settlement.ID, debt, amount, allocationKindPayment, now); err != nil {
		tx.Rollback()
		log.Printf("ERROR allocating compensation to debt split %d: %v", debt.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply compensation"})
		return
	}
	if err := applyAllocation(tx, settlement.ID, source, amount, allocationKindFunding, now); err != nil {
		tx.Rollback()
		log.Printf("ERROR consuming credit split %d: %v", source.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply compensation"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete compensation"})
		return
	}

	log.Printf("✅ COMPENSATION: %.2f from credit split %d applied to debt expense %d", amount, source.ID, targetExpenseID)

	h.db.Preload("FromMember").Preload("ToMember").First(&settlement, settlement.ID)
	c.JSON(http.StatusCreated, settlement)
}
