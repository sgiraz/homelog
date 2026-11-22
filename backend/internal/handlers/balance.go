package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// BalanceHandler handles balance calculations between members
type BalanceHandler struct {
	db *gorm.DB
}

// NewBalanceHandler creates a new balance handler
func NewBalanceHandler(db *gorm.DB) *BalanceHandler {
	return &BalanceHandler{db: db}
}

// BalanceResponse represents the balance calculation response
type BalanceResponse struct {
	Balance           float64 `json:"balance"`
	CurrentMemberID   uint    `json:"current_member_id"`
	CurrentMemberName string  `json:"current_member_name"`
	OtherMemberID     uint    `json:"other_member_id"`
	OtherMemberName   string  `json:"other_member_name"`
	// Counterparts is every other member with their own balance, the one
	// above first; the client lets the user switch between them.
	Counterparts []Counterpart `json:"counterparts"`
}

// UnsettledSplitDetail represents a single unsettled (or partially settled) expense split
type UnsettledSplitDetail struct {
	ExpenseID uint `json:"expense_id"`
	// SplitID identifies this exact share — the handle a compensation needs to
	// spend this credit against a long-term debt.
	SplitID       uint    `json:"split_id"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
	SettledAmount float64 `json:"settled_amount"`
	Remaining     float64 `json:"remaining_amount"`
	PaidByName    string  `json:"paid_by_name"`
	PaidByID      uint    `json:"paid_by_id"`
	Date          string  `json:"date"`
}

// SettlementDetail represents a settlement record
type SettlementDetail struct {
	ID             uint    `json:"id"`
	Date           string  `json:"date"`
	FromMemberName string  `json:"from_member_name"`
	FromMemberID   uint    `json:"from_member_id"`
	ToMemberName   string  `json:"to_member_name"`
	ToMemberID     uint    `json:"to_member_id"`
	Amount         float64 `json:"amount"`
	Note           string  `json:"note,omitempty"`
}

// BalanceDetailsResponse represents the detailed balance response
type BalanceDetailsResponse struct {
	Balance           float64                `json:"balance"`
	CurrentMemberID   uint                   `json:"current_member_id"`
	CurrentMemberName string                 `json:"current_member_name"`
	OtherMemberID     uint                   `json:"other_member_id"`
	OtherMemberName   string                 `json:"other_member_name"`
	Counterparts      []Counterpart          `json:"counterparts"`
	UnsettledSplits   []UnsettledSplitDetail `json:"unsettled_splits"`
	Settlements       []SettlementDetail     `json:"settlements"`
}

// GetBalance returns the running balance between the current user's member and
// one counterpart (other_member_id, or the one with the largest balance), plus
// every counterpart with its own balance.
// GET /api/v1/households/:id/balance?other_member_id=2
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	current, other, counterparts, ok := h.resolvePair(c, true)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, BalanceResponse{
		Balance:           other.Balance,
		CurrentMemberID:   current.ID,
		CurrentMemberName: current.Name,
		OtherMemberID:     other.MemberID,
		OtherMemberName:   other.Name,
		Counterparts:      counterparts,
	})
}

// resolvePair authorizes the request and resolves the member pair. ok is false
// when the response has already been written: an error, or an empty answer
// because the user has no member profile or no counterpart yet, or because
// splitModeOnly is set and the household does not split expenses.
func (h *BalanceHandler) resolvePair(c *gin.Context, splitModeOnly bool) (current models.HouseholdMember, other Counterpart, counterparts []Counterpart, ok bool) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}
	householdIDParsed, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_household_id", "Invalid household id")
		return
	}
	householdID := uint(householdIDParsed)

	// Only members may read a household's balance: without this, any signed-in
	// user gets a 200 for any household id.
	if !requireHouseholdMember(c, h.db, userID, householdID) {
		return
	}

	if splitModeOnly {
		var settings models.HouseholdSettings
		err := h.db.Where("household_id = ?", householdID).First(&settings).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to calculate balance")
			return
		}
		if err != nil || !settings.SplitMode {
			c.JSON(http.StatusOK, BalanceResponse{Counterparts: []Counterpart{}})
			return
		}
	}

	current, other, counterparts, err = resolveMemberPair(h.db, userID, householdID, c.Query("other_member_id"))
	switch {
	case err == nil:
		return current, other, counterparts, true
	case errors.Is(err, errInvalidOtherMember):
		apierr.Fail(c, http.StatusBadRequest, "invalid_other_member_id", "Invalid other member id")
	case errors.Is(err, errOtherMemberNotFound):
		apierr.Fail(c, http.StatusNotFound, "other_member_not_found", "The other member was not found")
	case errors.Is(err, gorm.ErrRecordNotFound):
		if counterparts == nil {
			counterparts = []Counterpart{}
		}
		c.JSON(http.StatusOK, BalanceDetailsResponse{
			CurrentMemberID:   current.ID,
			CurrentMemberName: current.Name,
			Counterparts:      counterparts,
			UnsettledSplits:   []UnsettledSplitDetail{},
			Settlements:       []SettlementDetail{},
		})
	default:
		log.Printf("ERROR resolving member pair for household %d: %v", householdID, err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to calculate balance")
	}
	return current, other, counterparts, false
}

// GetBalanceDetails returns detailed balance with unsettled expenses and settlements history
// GET /api/v1/households/:id/balance/details?other_member_id=2
func (h *BalanceHandler) GetBalanceDetails(c *gin.Context) {
	current, other, counterparts, ok := h.resolvePair(c, false)
	if !ok {
		return
	}
	householdID := current.HouseholdID
	otherMemberID := other.MemberID
	currentMember := current

	var err error
	// Get unsettled splits between these members (long-term debts live in their
	// own ledger — see DebtHandler.List)
	var splits []models.ExpenseSplit
	err = h.db.
		Preload("Expense").
		Preload("Expense.PaidBy").
		Preload("Member").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expense_splits.is_settled = ?", false).
		Where("expenses.household_id = ?", householdID).
		Where("expenses.deleted_at IS NULL").
		Where("expenses.is_long_term_debt = ?", false).
		Where("(expenses.paid_by_member_id = ? AND expense_splits.member_id = ?) OR (expenses.paid_by_member_id = ? AND expense_splits.member_id = ?)",
			currentMember.ID, otherMemberID, otherMemberID, currentMember.ID).
		Order("expenses.date DESC").
		Find(&splits).Error
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch unsettled splits")
		return
	}

	// Build unsettled splits response
	unsettledSplits := make([]UnsettledSplitDetail, 0, len(splits))
	for _, split := range splits {
		paidByName := ""
		if split.Expense.PaidBy != nil {
			paidByName = split.Expense.PaidBy.Name
		}
		unsettledSplits = append(unsettledSplits, UnsettledSplitDetail{
			ExpenseID:     split.ExpenseID,
			SplitID:       split.ID,
			Description:   split.Expense.Description,
			Amount:        split.Amount,
			SettledAmount: split.SettledAmount,
			Remaining:     remainingOwed(split),
			PaidByName:    paidByName,
			PaidByID:      split.Expense.PaidByMemberID,
			Date:          split.Expense.Date.Format("2006-01-02"),
		})
	}

	// Get settlements between these members. Payments aimed at a long-term debt
	// are listed under that debt instead, not in the running balance history.
	var settlements []models.Settlement
	err = h.db.
		Preload("FromMember").
		Preload("ToMember").
		Where("household_id = ?", householdID).
		Where("target_expense_id IS NULL").
		Where("(from_member_id = ? AND to_member_id = ?) OR (from_member_id = ? AND to_member_id = ?)",
			currentMember.ID, otherMemberID, otherMemberID, currentMember.ID).
		Order("date DESC").
		Find(&settlements).Error
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch settlements")
		return
	}

	// Build settlements response
	settlementDetails := make([]SettlementDetail, 0, len(settlements))
	for _, s := range settlements {
		settlementDetails = append(settlementDetails, SettlementDetail{
			ID:             s.ID,
			Date:           s.Date.Format("2006-01-02"),
			FromMemberName: s.FromMember.Name,
			FromMemberID:   s.FromMemberID,
			ToMemberName:   s.ToMember.Name,
			ToMemberID:     s.ToMemberID,
			Amount:         s.Amount,
			Note:           s.Note,
		})
	}

	c.JSON(http.StatusOK, BalanceDetailsResponse{
		Balance:           other.Balance,
		CurrentMemberID:   currentMember.ID,
		CurrentMemberName: currentMember.Name,
		OtherMemberID:     otherMemberID,
		OtherMemberName:   other.Name,
		Counterparts:      counterparts,
		UnsettledSplits:   unsettledSplits,
		Settlements:       settlementDetails,
	})
}
