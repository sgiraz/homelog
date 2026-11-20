package handlers

import (
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
	UnsettledSplits   []UnsettledSplitDetail `json:"unsettled_splits"`
	Settlements       []SettlementDetail     `json:"settlements"`
}

// GetBalance calculates and returns the balance between current user's member and another member
// GET /api/v1/households/:id/balance?other_member_id=2
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	currentUserID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	householdIDStr := c.Param("id")
	householdIDParsed, err := strconv.ParseUint(householdIDStr, 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_household_id", "Invalid household id")
		return
	}
	householdID := uint(householdIDParsed)

	// Only members may read a household's balance: without this, any signed-in
	// user gets a 200 for any household id.
	if !requireHouseholdMember(c, h.db, currentUserID, householdID) {
		return
	}

	log.Printf("🔍 BALANCE REQUEST: HouseholdID=%d, CurrentUserID=%d", householdID, currentUserID)

	// Check split mode first
	var settings models.HouseholdSettings
	if err := h.db.Where("household_id = ?", householdID).First(&settings).Error; err != nil {
		log.Printf("   ⚠️ No household settings found for household %d", householdID)
		c.JSON(http.StatusOK, BalanceResponse{
			Balance:         0,
			CurrentMemberID: 0,
			OtherMemberID:   0,
		})
		return
	}

	log.Printf("   Split Mode: %t", settings.SplitMode)

	if !settings.SplitMode {
		log.Printf("   ℹ️ Split mode disabled, returning 0")
		c.JSON(http.StatusOK, BalanceResponse{
			Balance:         0,
			CurrentMemberID: 0,
			OtherMemberID:   0,
		})
		return
	}

	// Find the current user's member ID for this household
	var currentMember models.HouseholdMember
	if err := h.db.Where("household_id = ? AND user_id = ?", householdID, currentUserID).First(&currentMember).Error; err != nil {
		log.Printf("   ⚠️ Current user has no member profile for household %d", householdID)
		c.JSON(http.StatusOK, BalanceResponse{
			Balance:         0,
			CurrentMemberID: 0,
			OtherMemberID:   0,
		})
		return
	}

	log.Printf("   Current member: ID=%d, Name=%s", currentMember.ID, currentMember.Name)

	// Parse other member ID from query (optional - can auto-detect)
	var otherMemberID uint
	var otherMember models.HouseholdMember
	if otherMemberIDStr := c.Query("other_member_id"); otherMemberIDStr != "" {
		otherMemberIDParsed, err := strconv.ParseUint(otherMemberIDStr, 10, 32)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_other_member_id", "Invalid other member id")
			return
		}
		otherMemberID = uint(otherMemberIDParsed)
		if err := h.db.Where("id = ? AND household_id = ?", otherMemberID, householdID).First(&otherMember).Error; err != nil {
			apierr.Fail(c, http.StatusNotFound, "other_member_not_found", "The other member was not found")
			return
		}
		log.Printf("   Other member specified: ID=%d, Name=%s", otherMember.ID, otherMember.Name)
	} else {
		// Auto-detect other member - find any member that is NOT the current user's member
		log.Printf("   Auto-detecting other member...")

		// Find any other member in the same household
		if err := h.db.Where("household_id = ? AND id != ?", householdID, currentMember.ID).First(&otherMember).Error; err != nil {
			log.Printf("   ⚠️ No other member found for household %d", householdID)
			c.JSON(http.StatusOK, BalanceResponse{
				Balance:           0,
				CurrentMemberID:   currentMember.ID,
				CurrentMemberName: currentMember.Name,
				OtherMemberID:     0,
			})
			return
		}
		otherMemberID = otherMember.ID
		log.Printf("   Found other member: ID=%d, Name=%s", otherMember.ID, otherMember.Name)
	}

	// Calculate balance
	balance, err := CalculateBalance(currentMember.ID, otherMemberID, householdID, h.db)
	if err != nil {
		log.Printf("   ❌ Error calculating balance: %v", err)
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to calculate balance")
		return
	}

	c.JSON(http.StatusOK, BalanceResponse{
		Balance:           balance,
		CurrentMemberID:   currentMember.ID,
		CurrentMemberName: currentMember.Name,
		OtherMemberID:     otherMemberID,
		OtherMemberName:   otherMember.Name,
	})
}

// GetBalanceDetails returns detailed balance with unsettled expenses and settlements history
// GET /api/v1/households/:id/balance/details?other_member_id=2
func (h *BalanceHandler) GetBalanceDetails(c *gin.Context) {
	currentUserID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	householdIDStr := c.Param("id")
	householdIDParsed, err := strconv.ParseUint(householdIDStr, 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_household_id", "Invalid household id")
		return
	}
	householdID := uint(householdIDParsed)

	// Only members may read a household's balance: without this, any signed-in
	// user gets a 200 for any household id.
	if !requireHouseholdMember(c, h.db, currentUserID, householdID) {
		return
	}

	log.Printf("GetBalanceDetails - HouseholdID: %d, CurrentUserID: %d", householdID, currentUserID)

	// Find the current user's member ID for this household
	var currentMember models.HouseholdMember
	if err := h.db.Where("household_id = ? AND user_id = ?", householdID, currentUserID).First(&currentMember).Error; err != nil {
		log.Printf("GetBalanceDetails - No member profile for user %d in household %d", currentUserID, householdID)
		c.JSON(http.StatusOK, BalanceDetailsResponse{
			Balance:         0,
			CurrentMemberID: 0,
			UnsettledSplits: []UnsettledSplitDetail{},
			Settlements:     []SettlementDetail{},
		})
		return
	}

	// Parse other member ID from query (optional)
	var otherMemberID uint
	var otherMember models.HouseholdMember
	if otherMemberIDStr := c.Query("other_member_id"); otherMemberIDStr != "" {
		otherMemberIDParsed, err := strconv.ParseUint(otherMemberIDStr, 10, 32)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_other_member_id", "Invalid other member id")
			return
		}
		otherMemberID = uint(otherMemberIDParsed)
		if err := h.db.Where("id = ? AND household_id = ?", otherMemberID, householdID).First(&otherMember).Error; err != nil {
			apierr.Fail(c, http.StatusNotFound, "other_member_not_found", "The other member was not found")
			return
		}
	} else {
		// Auto-detect other member
		if err := h.db.Where("household_id = ? AND id != ?", householdID, currentMember.ID).First(&otherMember).Error; err != nil {
			log.Printf("GetBalanceDetails - No other member found for household %d", householdID)
			c.JSON(http.StatusOK, BalanceDetailsResponse{
				Balance:           0,
				CurrentMemberID:   currentMember.ID,
				CurrentMemberName: currentMember.Name,
				OtherMemberID:     0,
				UnsettledSplits:   []UnsettledSplitDetail{},
				Settlements:       []SettlementDetail{},
			})
			return
		}
		otherMemberID = otherMember.ID
	}

	// Calculate balance
	balance, err := CalculateBalance(currentMember.ID, otherMemberID, householdID, h.db)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to calculate balance")
		return
	}

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
		Balance:           balance,
		CurrentMemberID:   currentMember.ID,
		CurrentMemberName: currentMember.Name,
		OtherMemberID:     otherMemberID,
		OtherMemberName:   otherMember.Name,
		UnsettledSplits:   unsettledSplits,
		Settlements:       settlementDetails,
	})
}

// CalculateBalance calculates the balance between two members for a household
// Positive balance means other member owes current member
// Negative balance means current member owes other member
func CalculateBalance(currentMemberID, otherMemberID, householdID uint, db *gorm.DB) (float64, error) {
	balance := 0.0

	log.Printf("CalculateBalance - CurrentMember: %d, OtherMember: %d, HouseholdID: %d", currentMemberID, otherMemberID, householdID)

	// 1. Check if split mode is enabled for this household
	var settings models.HouseholdSettings
	err := db.Where("household_id = ?", householdID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("CalculateBalance - No settings found for household %d, split mode disabled by default", householdID)
			return 0.0, nil
		}
		log.Printf("CalculateBalance - Error fetching settings: %v", err)
		return 0, err
	}

	log.Printf("CalculateBalance - SplitMode: %v", settings.SplitMode)

	if !settings.SplitMode {
		log.Printf("CalculateBalance - Split mode disabled, returning 0")
		return 0.0, nil
	}

	// 2. Get unsettled ExpenseSplits for this household. Long-term debts are
	// excluded by design: they are repaid on their own schedule from the Debiti
	// tab, and netting them here would swallow every ordinary expense until the
	// big debt is gone (see Expense.IsLongTermDebt).
	var splits []models.ExpenseSplit
	err = db.
		Preload("Expense").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expense_splits.is_settled = ?", false).
		Where("expenses.household_id = ?", householdID).
		Where("expenses.deleted_at IS NULL").
		Where("expenses.is_long_term_debt = ?", false).
		Find(&splits).Error
	if err != nil {
		log.Printf("CalculateBalance - Error fetching splits: %v", err)
		return 0, err
	}

	log.Printf("CalculateBalance - Found %d unsettled splits for household %d", len(splits), householdID)

	// 3. Calculate from splits (remaining balance, not the original split amount —
	// a split may have been partially paid down via the settlement ledger)
	for _, split := range splits {
		remaining := remainingOwed(split)
		log.Printf("CalculateBalance - Split: ExpenseID=%d, PaidByMemberID=%d, MemberID=%d, Amount=%.2f, Remaining=%.2f",
			split.ExpenseID, split.Expense.PaidByMemberID, split.MemberID, split.Amount, remaining)

		if split.Expense.PaidByMemberID == currentMemberID && split.MemberID == otherMemberID {
			// Other member owes me this amount
			balance += remaining
			log.Printf("CalculateBalance - OtherMember owes me %.2f", remaining)
		} else if split.Expense.PaidByMemberID == otherMemberID && split.MemberID == currentMemberID {
			// I owe other member this amount
			balance -= remaining
			log.Printf("CalculateBalance - I owe OtherMember %.2f", remaining)
		}
	}

	log.Printf("CalculateBalance - Final balance: %.2f", balance)

	// NOTE: Settlements are NOT counted separately in the balance calculation because a settlement's
	// amount is applied directly against ExpenseSplit.SettledAmount when it's created (see
	// SettlementHandler.Create). The loop above already reads the post-allocation remaining balance,
	// so summing settlement amounts too would be double-counting.

	return balance, nil
}
