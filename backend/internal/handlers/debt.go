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

// DebtHandler serves the long-term debt ledger: the big one-off imbalances
// (a mortgage down payment, say) that are deliberately kept out of the running
// household balance and repaid on their own schedule instead of silently
// swallowing every ordinary expense between the two members.
type DebtHandler struct {
	db *gorm.DB
}

// NewDebtHandler creates a new debt handler
func NewDebtHandler(db *gorm.DB) *DebtHandler {
	return &DebtHandler{db: db}
}

// DebtPaymentDetail is one repayment recorded against a long-term debt.
type DebtPaymentDetail struct {
	SettlementID  uint    `json:"settlement_id"`
	Date          string  `json:"date"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method,omitempty"`
	Note          string  `json:"note,omitempty"`
	// SourceLabel names the expense whose credit funded the repayment, set only
	// for compensations (PaymentMethod == "compensation").
	SourceLabel string `json:"source_label,omitempty"`
}

// DebtDetail is a single long-term debt between the current member and the
// other member: one non-payer share of an expense flagged IsLongTermDebt.
type DebtDetail struct {
	ExpenseID     uint    `json:"expense_id"`
	SplitID       uint    `json:"split_id"`
	Description   string  `json:"description"`
	Date          string  `json:"date"`
	ProjectName   string  `json:"project_name,omitempty"`
	ExpenseTotal  float64 `json:"expense_total"`
	Amount        float64 `json:"amount"`
	SettledAmount float64 `json:"settled_amount"`
	Remaining     float64 `json:"remaining_amount"`
	// IOwe is true when the current member is the debtor, false when they are
	// the creditor waiting to be repaid.
	IOwe            bool                `json:"i_owe"`
	CounterpartName string              `json:"counterpart_name"`
	IsFullyRepaid   bool                `json:"is_fully_repaid"`
	Payments        []DebtPaymentDetail `json:"payments"`
}

// DebtsResponse is the payload behind the Debiti tab.
type DebtsResponse struct {
	CurrentMemberID   uint         `json:"current_member_id"`
	CurrentMemberName string       `json:"current_member_name"`
	OtherMemberID     uint         `json:"other_member_id"`
	OtherMemberName   string       `json:"other_member_name"`
	Debts             []DebtDetail `json:"debts"`
	TotalIOwe         float64      `json:"total_i_owe"`
	TotalTheyOwe      float64      `json:"total_they_owe"`
}

// resolveMemberPair finds the current user's member for a property and the
// counterpart member, either explicitly requested via other_member_id or
// auto-detected as any other member of the same property.
func resolveMemberPair(db *gorm.DB, userID, propertyID uint, otherMemberIDStr string) (current, other models.HouseholdMember, err error) {
	if err = db.Where("property_id = ? AND user_id = ?", propertyID, userID).First(&current).Error; err != nil {
		return current, other, err
	}
	if otherMemberIDStr != "" {
		parsed, parseErr := strconv.ParseUint(otherMemberIDStr, 10, 32)
		if parseErr != nil {
			return current, other, parseErr
		}
		err = db.Where("id = ? AND property_id = ?", uint(parsed), propertyID).First(&other).Error
		return current, other, err
	}
	err = db.Where("property_id = ? AND id != ?", propertyID, current.ID).First(&other).Error
	return current, other, err
}

// List returns the long-term debts between the current member and another member
// GET /api/v1/properties/:id/debts?other_member_id=2
func (h *DebtHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	propertyIDParsed, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}
	propertyID := uint(propertyIDParsed)

	if !requirePropertyMember(c, h.db, userID, propertyID) {
		return
	}

	empty := DebtsResponse{Debts: []DebtDetail{}}

	current, other, err := resolveMemberPair(h.db, userID, propertyID, c.Query("other_member_id"))
	if err != nil {
		// No member profile or no counterpart yet — nothing to owe anyone.
		c.JSON(http.StatusOK, empty)
		return
	}

	empty.CurrentMemberID = current.ID
	empty.CurrentMemberName = current.Name
	empty.OtherMemberID = other.ID
	empty.OtherMemberName = other.Name

	// Every non-payer share of a flagged expense, in either direction.
	var splits []models.ExpenseSplit
	err = h.db.
		Preload("Expense").
		Preload("Expense.Project").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expenses.property_id = ?", propertyID).
		Where("expenses.deleted_at IS NULL").
		Where("expenses.is_long_term_debt = ?", true).
		Where("(expenses.paid_by_member_id = ? AND expense_splits.member_id = ?) OR (expenses.paid_by_member_id = ? AND expense_splits.member_id = ?)",
			current.ID, other.ID, other.ID, current.ID).
		Order("expenses.date DESC").
		Find(&splits).Error
	if err != nil {
		log.Printf("ERROR fetching long-term debts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch debts"})
		return
	}

	if len(splits) == 0 {
		c.JSON(http.StatusOK, empty)
		return
	}

	expenseIDs := make([]uint, 0, len(splits))
	for _, s := range splits {
		expenseIDs = append(expenseIDs, s.ExpenseID)
	}

	paymentsByExpense := h.paymentsFor(propertyID, current.ID, other.ID, expenseIDs)

	resp := empty
	resp.Debts = make([]DebtDetail, 0, len(splits))
	for _, s := range splits {
		remaining := s.Amount - s.SettledAmount
		if remaining < 0 {
			remaining = 0
		}
		iOwe := s.MemberID == current.ID
		counterpart := other.Name
		if !iOwe {
			counterpart = current.Name
		}

		detail := DebtDetail{
			ExpenseID:       s.ExpenseID,
			SplitID:         s.ID,
			Description:     s.Expense.Description,
			Date:            s.Expense.Date.Format("2006-01-02"),
			ExpenseTotal:    s.Expense.Amount,
			Amount:          s.Amount,
			SettledAmount:   s.SettledAmount,
			Remaining:       remaining,
			IOwe:            iOwe,
			CounterpartName: counterpart,
			IsFullyRepaid:   remaining <= settlementEpsilon,
			Payments:        paymentsByExpense[s.ExpenseID],
		}
		if s.Expense.Project != nil {
			detail.ProjectName = s.Expense.Project.Name
		}
		if detail.Payments == nil {
			detail.Payments = []DebtPaymentDetail{}
		}

		if iOwe {
			resp.TotalIOwe += remaining
		} else {
			resp.TotalTheyOwe += remaining
		}
		resp.Debts = append(resp.Debts, detail)
	}

	c.JSON(http.StatusOK, resp)
}

// paymentsFor loads the repayments recorded against the given debts, keyed by
// expense ID, resolving the funding expense label for compensations.
func (h *DebtHandler) paymentsFor(propertyID, currentMemberID, otherMemberID uint, expenseIDs []uint) map[uint][]DebtPaymentDetail {
	byExpense := make(map[uint][]DebtPaymentDetail)

	var settlements []models.Settlement
	if err := h.db.
		Where("property_id = ?", propertyID).
		Where("target_expense_id IN ?", expenseIDs).
		Where("(from_member_id = ? AND to_member_id = ?) OR (from_member_id = ? AND to_member_id = ?)",
			currentMemberID, otherMemberID, otherMemberID, currentMemberID).
		Order("date DESC").
		Find(&settlements).Error; err != nil {
		log.Printf("ERROR fetching debt payments: %v", err)
		return byExpense
	}
	if len(settlements) == 0 {
		return byExpense
	}

	// Resolve, in one query, which expense funded each compensation.
	settlementIDs := make([]uint, 0, len(settlements))
	for _, s := range settlements {
		settlementIDs = append(settlementIDs, s.ID)
	}
	type fundingRow struct {
		SettlementID uint
		Description  string
	}
	var fundingRows []fundingRow
	h.db.Model(&models.SettlementAllocation{}).
		Select("settlement_allocations.settlement_id AS settlement_id, expenses.description AS description").
		Joins("JOIN expense_splits ON expense_splits.id = settlement_allocations.expense_split_id").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("settlement_allocations.kind = ?", allocationKindFunding).
		Where("settlement_allocations.settlement_id IN ?", settlementIDs).
		Scan(&fundingRows)

	fundingLabels := make(map[uint]string, len(fundingRows))
	for _, r := range fundingRows {
		fundingLabels[r.SettlementID] = r.Description
	}

	for _, s := range settlements {
		if s.TargetExpenseID == nil {
			continue
		}
		byExpense[*s.TargetExpenseID] = append(byExpense[*s.TargetExpenseID], DebtPaymentDetail{
			SettlementID:  s.ID,
			Date:          s.Date.Format("2006-01-02"),
			Amount:        s.Amount,
			PaymentMethod: s.PaymentMethod,
			Note:          s.Note,
			SourceLabel:   fundingLabels[s.ID],
		})
	}
	return byExpense
}

// SetLongTermDebtRequest toggles the long-term debt flag on an expense.
type SetLongTermDebtRequest struct {
	IsLongTermDebt *bool `json:"is_long_term_debt" binding:"required"`
}

// SetLongTermDebt moves a split expense in or out of the long-term debt ledger
// PATCH /api/v1/expenses/:id/long-term-debt
func (h *DebtHandler) SetLongTermDebt(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	expenseID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expense ID"})
		return
	}

	var req SetLongTermDebtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var expense models.Expense
	if err := h.db.First(&expense, expenseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load expense"})
		}
		return
	}

	if expense.PropertyID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solo le spese di una casa possono diventare debiti a lungo termine"})
		return
	}
	if !requirePropertyMember(c, h.db, userID, *expense.PropertyID) {
		return
	}
	if !expense.IsSplit {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solo le spese divise possono diventare debiti a lungo termine"})
		return
	}

	// Refuse while money has already moved on this expense: the payment would
	// otherwise end up filed under a ledger it was never made against.
	var alreadyPaid int64
	h.db.Model(&models.ExpenseSplit{}).
		Where("expense_id = ? AND member_id <> ? AND settled_amount > 0", expense.ID, expense.PaidByMemberID).
		Count(&alreadyPaid)
	if alreadyPaid > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Ci sono già pagamenti registrati su questa spesa. Annullali prima di spostarla.",
		})
		return
	}

	if err := h.db.Model(&expense).Update("is_long_term_debt", *req.IsLongTermDebt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
		return
	}

	log.Printf("Expense ID=%d long-term debt flag → %v", expense.ID, *req.IsLongTermDebt)
	c.JSON(http.StatusOK, gin.H{
		"expense_id":        expense.ID,
		"is_long_term_debt": *req.IsLongTermDebt,
	})
}
