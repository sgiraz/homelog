package handlers

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// settlementEpsilon absorbs float64 rounding noise when comparing amounts.
const settlementEpsilon = 0.005

// remainingOwed is what is still due on a split once the settlement ledger is
// applied, clamped at zero. Successive partial allocations accumulate float64
// error, so a fully repaid split can land a hair past its own amount; the
// negative residue must never reach the API, the balance, or the UI.
func remainingOwed(split models.ExpenseSplit) float64 {
	remaining := split.Amount - split.SettledAmount
	if remaining < 0 {
		return 0
	}
	return remaining
}

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
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var req CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR binding settlement JSON: %v", err)
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Validate: from_member != to_member
	if req.FromMemberID == req.ToMemberID {
		apierr.Fail(c, http.StatusBadRequest, "settlement_same_member", "A settlement needs two different members")
		return
	}

	// Find the current user's member for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", req.PropertyID, userID).First(&currentMember).Error; err != nil {
		apierr.Fail(c, http.StatusForbidden, "not_property_member", "You are not a member of this property")
		return
	}

	// Validate: current user's member is involved in the settlement
	if currentMember.ID != req.FromMemberID && currentMember.ID != req.ToMemberID {
		apierr.Fail(c, http.StatusForbidden, "not_in_settlement", "You are not involved in this settlement")
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_date", "Invalid date format")
		return
	}

	// Verify property exists
	var property models.Property
	if err := h.db.First(&property, req.PropertyID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "property_not_found", "Property not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to verify property")
		}
		return
	}

	// Outstanding shares between the pair, oldest expense first — the payment
	// is applied as a ledger, oldest debt paid down before newer ones.
	outstanding := func() *gorm.DB {
		return h.db.
			Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
			Where("expense_splits.is_settled = ?", false).
			Where("expenses.property_id = ?", req.PropertyID).
			Where("expenses.deleted_at IS NULL")
	}

	// debtQuery: shares the payer still owes the recipient — what this payment
	// actually repays. Only ever this direction: paying Alice must not clear
	// what Alice owes you.
	debtQuery := outstanding().
		Where("expenses.paid_by_member_id = ? AND expense_splits.member_id = ?", req.ToMemberID, req.FromMemberID)

	// credits: shares the recipient owes the payer. No money moves for them —
	// they cancel against the debts above (see the netting below). Empty for a
	// payment aimed at a single long-term debt, which is never netted.
	var credits []models.ExpenseSplit

	if req.TargetExpenseID != nil {
		var target models.Expense
		if err := h.db.First(&target, *req.TargetExpenseID).Error; err != nil {
			apierr.Fail(c, http.StatusNotFound, "debt_not_found", "Debt not found")
			return
		}
		if !target.IsLongTermDebt {
			apierr.Fail(c, http.StatusBadRequest, "expense_not_long_term_debt", "This expense is not a long-term debt")
			return
		}
		debtQuery = debtQuery.Where("expenses.id = ?", target.ID)
	} else {
		// Ordinary balance: long-term debts are repaid explicitly, never swept
		// up by a generic settlement.
		debtQuery = debtQuery.Where("expenses.is_long_term_debt = ?", false)

		if err := outstanding().
			Where("expenses.is_long_term_debt = ?", false).
			Where("expenses.paid_by_member_id = ? AND expense_splits.member_id = ?", req.FromMemberID, req.ToMemberID).
			Order("expenses.date ASC").
			Find(&credits).Error; err != nil {
			log.Printf("ERROR finding counter-shares to net: %v", err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to find expense splits")
			return
		}
	}

	var splits []models.ExpenseSplit
	if err := debtQuery.Order("expenses.date ASC").Find(&splits).Error; err != nil {
		log.Printf("ERROR finding expense splits to settle: %v", err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to find expense splits")
		return
	}

	var owedByPayer, owedToPayer float64
	for _, s := range splits {
		owedByPayer += remainingOwed(s)
	}
	for _, s := range credits {
		owedToPayer += remainingOwed(s)
	}

	// The balance between two members is netted (see CalculateBalance), so the
	// most that can legitimately change hands is the *net* debt. Validating
	// against the gross total would accept a payment larger than what is owed.
	netOwed := owedByPayer - owedToPayer
	if req.Amount > netOwed+settlementEpsilon {
		apierr.Fail(c, http.StatusBadRequest, "amount_exceeds_debt", "The amount is larger than the debt left between these members")
		return
	}

	tx := h.db.Begin()

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
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create settlement")
		return
	}

	log.Printf("Settlement created - ID: %d, From: %d, To: %d, Amount: %.2f",
		settlement.ID, req.FromMemberID, req.ToMemberID, req.Amount)

	now := time.Now()

	// Cancel the counter-shares first. The recipient owes these back to the
	// payer, so no cash covers them — they are offset against the debts this
	// settlement repays, exactly as the netted balance already presents them.
	// Netting them in full always fits (owedByPayer >= req.Amount + owedToPayer)
	// and leaves the outstanding shares equal to the balance after the payment.
	netted := 0.0
	for i := range credits {
		owed := remainingOwed(credits[i])
		if owed <= settlementEpsilon {
			continue
		}
		if err := applyAllocation(tx, settlement.ID, credits[i], owed, allocationKindFunding, now); err != nil {
			tx.Rollback()
			log.Printf("ERROR netting counter-share split %d: %v", credits[i].ID, err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to allocate settlement")
			return
		}
		netted += owed
	}

	// Apply the payment against the outstanding splits, oldest first. Each
	// split touched gets a SettlementAllocation recording exactly how much of
	// this settlement covered it; a split not fully covered stays unsettled
	// with a reduced remaining balance instead of being force-marked settled.
	// The cash is joined by whatever the counter-shares cancelled out.
	remaining := req.Amount + netted
	toAllocate := remaining
	touched := 0
	for i := range splits {
		if remaining <= settlementEpsilon {
			break
		}
		owed := remainingOwed(splits[i])
		if owed <= settlementEpsilon {
			continue
		}
		alloc := math.Min(owed, remaining)

		if err := applyAllocation(tx, settlement.ID, splits[i], alloc, allocationKindPayment, now); err != nil {
			tx.Rollback()
			log.Printf("ERROR allocating settlement to split %d: %v", splits[i].ID, err)
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to allocate settlement")
			return
		}

		remaining -= alloc
		touched++
	}

	log.Printf("✅ SETTLEMENT: Allocated %.2f across %d splits (%.2f cash + %.2f netted from counter-shares)",
		toAllocate-remaining, touched, req.Amount, netted)

	if err := tx.Commit().Error; err != nil {
		log.Printf("ERROR committing transaction: %v", err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to complete settlement")
		return
	}

	h.db.Preload("FromMember").Preload("ToMember").First(&settlement, settlement.ID)

	c.JSON(http.StatusCreated, settlement)
}

// List returns settlements for a property
// GET /api/v1/settlements?property_id=1
func (h *SettlementHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	propertyIDStr := c.Query("property_id")
	if propertyIDStr == "" {
		apierr.Fail(c, http.StatusBadRequest, "property_id_required", "A property is required")
		return
	}

	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_property_id", "Invalid property id")
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
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch settlements")
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
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_settlement_id", "Invalid settlement id")
		return
	}

	var settlement models.Settlement
	if err := h.db.
		Preload("FromMember").
		Preload("ToMember").
		Preload("Allocations").
		First(&settlement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "settlement_not_found", "Settlement not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch settlement")
		}
		return
	}

	// Find the current user's member for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", settlement.PropertyID, userID).First(&currentMember).Error; err != nil {
		apierr.Fail(c, http.StatusForbidden, "not_property_member", "You are not a member of this property")
		return
	}

	// Verify user is involved
	if settlement.FromMemberID != currentMember.ID && settlement.ToMemberID != currentMember.ID {
		apierr.Fail(c, http.StatusForbidden, "not_in_settlement", "You are not involved in this settlement")
		return
	}

	c.JSON(http.StatusOK, settlement)
}

// Delete removes a settlement (soft delete)
// DELETE /api/v1/settlements/:id
func (h *SettlementHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_settlement_id", "Invalid settlement id")
		return
	}

	// Find the settlement
	var settlement models.Settlement
	if err := h.db.First(&settlement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "settlement_not_found", "Settlement not found")
		} else {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to find settlement")
		}
		return
	}

	// Find the current user's member for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", settlement.PropertyID, userID).First(&currentMember).Error; err != nil {
		apierr.Fail(c, http.StatusForbidden, "not_property_member", "You are not a member of this property")
		return
	}

	// Verify user is involved
	if settlement.FromMemberID != currentMember.ID && settlement.ToMemberID != currentMember.ID {
		apierr.Fail(c, http.StatusForbidden, "not_in_settlement", "You are not involved in this settlement")
		return
	}

	tx := h.db.Begin()

	// Reverse only this settlement's own allocations — a split may have
	// received contributions from other settlements too (before or after this
	// one), which must stay untouched.
	var allocations []models.SettlementAllocation
	if err := tx.Where("settlement_id = ?", settlement.ID).Find(&allocations).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to load settlement allocations")
		return
	}

	for _, alloc := range allocations {
		var split models.ExpenseSplit
		if err := tx.First(&split, alloc.ExpenseSplitID).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to load expense split")
			return
		}

		newSettledAmount := split.SettledAmount - alloc.Amount
		if newSettledAmount < settlementEpsilon {
			newSettledAmount = 0
		}

		// is_settled / settled_at / settlement_id are derived from the ledger,
		// so rebuild them from what survives this reversal rather than blanking
		// them: a split funded by several settlements is still (possibly fully)
		// settled by the others, and must keep pointing at the most recent one.
		updates := map[string]any{
			"settled_amount": newSettledAmount,
			"is_settled":     false,
			"settled_at":     nil,
			"settlement_id":  nil,
		}

		var survivor models.SettlementAllocation
		err := tx.
			Joins("JOIN settlements ON settlements.id = settlement_allocations.settlement_id AND settlements.deleted_at IS NULL").
			Where("settlement_allocations.expense_split_id = ? AND settlement_allocations.settlement_id != ?", split.ID, settlement.ID).
			Order("settlements.date DESC, settlement_allocations.id DESC").
			First(&survivor).Error
		switch {
		case err == nil:
			updates["settlement_id"] = survivor.SettlementID
			if newSettledAmount >= split.Amount-settlementEpsilon {
				updates["is_settled"] = true
				updates["settled_at"] = survivor.CreatedAt
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			// This settlement was the split's only funder — fully unsettled.
		default:
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to load remaining allocations")
			return
		}

		if err := tx.Model(&models.ExpenseSplit{}).Where("id = ?", split.ID).Updates(updates).Error; err != nil {
			tx.Rollback()
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to unsettle expense split")
			return
		}
	}

	if err := tx.Where("settlement_id = ?", settlement.ID).Delete(&models.SettlementAllocation{}).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete settlement allocations")
		return
	}

	if err := tx.Delete(&settlement).Error; err != nil {
		tx.Rollback()
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete settlement")
		return
	}

	if err := tx.Commit().Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to complete deletion")
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
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var req CompensateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if !requirePropertyMember(c, h.db, userID, req.PropertyID) {
		return
	}

	// The credit being spent: `source.MemberID` owes this share to whoever paid
	// the expense, and it's that payer's own debt we're about to shrink.
	var source models.ExpenseSplit
	if err := h.db.Preload("Expense").First(&source, req.SourceSplitID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "credit_expense_not_found", "Credit expense not found")
		return
	}
	if source.Expense.PropertyID == nil || *source.Expense.PropertyID != req.PropertyID {
		apierr.Fail(c, http.StatusBadRequest, "expense_not_in_property", "This expense does not belong to this household")
		return
	}
	if source.Expense.IsLongTermDebt {
		apierr.Fail(c, http.StatusBadRequest, "long_term_debt_cannot_fund", "A long-term debt cannot fund a compensation")
		return
	}
	sourceRemaining := remainingOwed(source)
	if sourceRemaining <= settlementEpsilon {
		apierr.Fail(c, http.StatusBadRequest, "share_already_settled", "This share is already settled")
		return
	}

	// debtor repays their own debt using the credit; holder is owed both.
	debtorID := source.Expense.PaidByMemberID
	holderID := source.MemberID

	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", req.PropertyID, userID).First(&currentMember).Error; err != nil {
		apierr.Fail(c, http.StatusForbidden, "not_property_member", "You are not a member of this property")
		return
	}
	if currentMember.ID != debtorID && currentMember.ID != holderID {
		apierr.Fail(c, http.StatusForbidden, "not_in_compensation", "You are not involved in this compensation")
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
		apierr.Fail(c, http.StatusNotFound, "long_term_debt_not_found", "No long-term debt exists between these two members")
		return
	}
	debtRemaining := remainingOwed(debt)
	if debtRemaining <= settlementEpsilon {
		apierr.Fail(c, http.StatusBadRequest, "debt_already_settled", "This debt is already paid off")
		return
	}

	amount := math.Min(sourceRemaining, debtRemaining)

	date := time.Now()
	if req.Date != "" {
		parsed, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_date", "Invalid date format")
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
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create compensation")
		return
	}

	now := time.Now()
	// Both sides move by the same figure: the debt shrinks, and the credit that
	// paid for it is consumed. Delete() reverses both together.
	if err := applyAllocation(tx, settlement.ID, debt, amount, allocationKindPayment, now); err != nil {
		tx.Rollback()
		log.Printf("ERROR allocating compensation to debt split %d: %v", debt.ID, err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to apply compensation")
		return
	}
	if err := applyAllocation(tx, settlement.ID, source, amount, allocationKindFunding, now); err != nil {
		tx.Rollback()
		log.Printf("ERROR consuming credit split %d: %v", source.ID, err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to apply compensation")
		return
	}

	if err := tx.Commit().Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to complete compensation")
		return
	}

	log.Printf("✅ COMPENSATION: %.2f from credit split %d applied to debt expense %d", amount, source.ID, targetExpenseID)

	h.db.Preload("FromMember").Preload("ToMember").First(&settlement, settlement.ID)
	c.JSON(http.StatusCreated, settlement)
}
