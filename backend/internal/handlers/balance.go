package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
	Message           string  `json:"message"`
}

// UnsettledSplitDetail represents a single unsettled expense split
type UnsettledSplitDetail struct {
	ExpenseID   uint    `json:"expense_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	PaidByName  string  `json:"paid_by_name"`
	PaidByID    uint    `json:"paid_by_id"`
	Date        string  `json:"date"`
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
// GET /api/v1/properties/:id/balance?other_member_id=2
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	currentUserID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse property ID from URL
	propertyIDStr := c.Param("id")
	propertyIDParsed, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}
	propertyID := uint(propertyIDParsed)

	log.Printf("🔍 BALANCE REQUEST: PropertyID=%d, CurrentUserID=%d", propertyID, currentUserID)

	// Check split mode first
	var settings models.HouseholdSettings
	if err := h.db.Where("property_id = ?", propertyID).First(&settings).Error; err != nil {
		log.Printf("   ⚠️ No household settings found for property %d", propertyID)
		c.JSON(http.StatusOK, BalanceResponse{
			Balance:         0,
			CurrentMemberID: 0,
			OtherMemberID:   0,
			Message:         "Settings non trovate",
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
			Message:         "Split mode disattivato",
		})
		return
	}

	// Find the current user's member ID for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", propertyID, currentUserID).First(&currentMember).Error; err != nil {
		log.Printf("   ⚠️ Current user has no member profile for property %d", propertyID)
		c.JSON(http.StatusOK, BalanceResponse{
			Balance:         0,
			CurrentMemberID: 0,
			OtherMemberID:   0,
			Message:         "Profilo membro non trovato",
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid other_member_id"})
			return
		}
		otherMemberID = uint(otherMemberIDParsed)
		if err := h.db.First(&otherMember, otherMemberID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Other member not found"})
			return
		}
		log.Printf("   Other member specified: ID=%d, Name=%s", otherMember.ID, otherMember.Name)
	} else {
		// Auto-detect other member - find any member that is NOT the current user's member
		log.Printf("   Auto-detecting other member...")

		// Find any other member in the same property
		if err := h.db.Where("property_id = ? AND id != ?", propertyID, currentMember.ID).First(&otherMember).Error; err != nil {
			log.Printf("   ⚠️ No other member found for property %d", propertyID)
			c.JSON(http.StatusOK, BalanceResponse{
				Balance:           0,
				CurrentMemberID:   currentMember.ID,
				CurrentMemberName: currentMember.Name,
				OtherMemberID:     0,
				Message:           "Nessun altro membro nella casa",
			})
			return
		}
		otherMemberID = otherMember.ID
		log.Printf("   Found other member: ID=%d, Name=%s", otherMember.ID, otherMember.Name)
	}

	// Calculate balance
	balance, err := CalculateBalance(currentMember.ID, otherMemberID, propertyID, h.db)
	if err != nil {
		log.Printf("   ❌ Error calculating balance: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate balance"})
		return
	}

	// Generate message
	var message string
	if balance > 0 {
		message = fmt.Sprintf("%s ti deve €%.2f", otherMember.Name, balance)
	} else if balance < 0 {
		message = fmt.Sprintf("Devi a %s €%.2f", otherMember.Name, -balance)
	} else {
		message = "Siete pari"
	}

	log.Printf("   💰 FINAL BALANCE: %.2f - %s", balance, message)

	c.JSON(http.StatusOK, BalanceResponse{
		Balance:           balance,
		CurrentMemberID:   currentMember.ID,
		CurrentMemberName: currentMember.Name,
		OtherMemberID:     otherMemberID,
		OtherMemberName:   otherMember.Name,
		Message:           message,
	})
}

// GetBalanceDetails returns detailed balance with unsettled expenses and settlements history
// GET /api/v1/properties/:id/balance/details?other_member_id=2
func (h *BalanceHandler) GetBalanceDetails(c *gin.Context) {
	currentUserID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse property ID from URL
	propertyIDStr := c.Param("id")
	propertyIDParsed, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}
	propertyID := uint(propertyIDParsed)

	log.Printf("GetBalanceDetails - PropertyID: %d, CurrentUserID: %d", propertyID, currentUserID)

	// Find the current user's member ID for this property
	var currentMember models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", propertyID, currentUserID).First(&currentMember).Error; err != nil {
		log.Printf("GetBalanceDetails - No member profile for user %d in property %d", currentUserID, propertyID)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid other_member_id"})
			return
		}
		otherMemberID = uint(otherMemberIDParsed)
		if err := h.db.First(&otherMember, otherMemberID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Other member not found"})
			return
		}
	} else {
		// Auto-detect other member
		if err := h.db.Where("property_id = ? AND id != ?", propertyID, currentMember.ID).First(&otherMember).Error; err != nil {
			log.Printf("GetBalanceDetails - No other member found for property %d", propertyID)
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
	balance, err := CalculateBalance(currentMember.ID, otherMemberID, propertyID, h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate balance"})
		return
	}

	// Get unsettled splits between these members
	var splits []models.ExpenseSplit
	err = h.db.
		Preload("Expense").
		Preload("Expense.PaidBy").
		Preload("Member").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expense_splits.is_settled = ?", false).
		Where("expenses.property_id = ?", propertyID).
		Where("expenses.deleted_at IS NULL").
		Where("(expenses.paid_by_member_id = ? AND expense_splits.member_id = ?) OR (expenses.paid_by_member_id = ? AND expense_splits.member_id = ?)",
			currentMember.ID, otherMemberID, otherMemberID, currentMember.ID).
		Order("expenses.date DESC").
		Find(&splits).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unsettled splits"})
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
			ExpenseID:   split.ExpenseID,
			Description: split.Expense.Description,
			Amount:      split.Amount,
			PaidByName:  paidByName,
			PaidByID:    split.Expense.PaidByMemberID,
			Date:        split.Expense.Date.Format("2006-01-02"),
		})
	}

	// Get settlements between these members
	var settlements []models.Settlement
	err = h.db.
		Preload("FromMember").
		Preload("ToMember").
		Where("property_id = ?", propertyID).
		Where("(from_member_id = ? AND to_member_id = ?) OR (from_member_id = ? AND to_member_id = ?)",
			currentMember.ID, otherMemberID, otherMemberID, currentMember.ID).
		Order("date DESC").
		Find(&settlements).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settlements"})
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

// CalculateBalance calculates the balance between two members for a property
// Positive balance means other member owes current member
// Negative balance means current member owes other member
func CalculateBalance(currentMemberID, otherMemberID, propertyID uint, db *gorm.DB) (float64, error) {
	balance := 0.0

	log.Printf("CalculateBalance - CurrentMember: %d, OtherMember: %d, PropertyID: %d", currentMemberID, otherMemberID, propertyID)

	// 1. Check if split mode is enabled for this property
	var settings models.HouseholdSettings
	err := db.Where("property_id = ?", propertyID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("CalculateBalance - No settings found for property %d, split mode disabled by default", propertyID)
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

	// 2. Get unsettled ExpenseSplits for this property
	var splits []models.ExpenseSplit
	err = db.
		Preload("Expense").
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id").
		Where("expense_splits.is_settled = ?", false).
		Where("expenses.property_id = ?", propertyID).
		Where("expenses.deleted_at IS NULL").
		Find(&splits).Error
	if err != nil {
		log.Printf("CalculateBalance - Error fetching splits: %v", err)
		return 0, err
	}

	log.Printf("CalculateBalance - Found %d unsettled splits for property %d", len(splits), propertyID)

	// 3. Calculate from splits
	for _, split := range splits {
		log.Printf("CalculateBalance - Split: ExpenseID=%d, PaidByMemberID=%d, MemberID=%d, Amount=%.2f",
			split.ExpenseID, split.Expense.PaidByMemberID, split.MemberID, split.Amount)

		if split.Expense.PaidByMemberID == currentMemberID && split.MemberID == otherMemberID {
			// Other member owes me this amount
			balance += split.Amount
			log.Printf("CalculateBalance - OtherMember owes me %.2f", split.Amount)
		} else if split.Expense.PaidByMemberID == otherMemberID && split.MemberID == currentMemberID {
			// I owe other member this amount
			balance -= split.Amount
			log.Printf("CalculateBalance - I owe OtherMember %.2f", split.Amount)
		}
	}

	log.Printf("CalculateBalance - Final balance: %.2f", balance)

	// NOTE: Settlements are NOT counted in balance calculation because when a settlement is created,
	// the associated expense splits are marked as is_settled=true. This means they're already
	// excluded from the balance calculation above. Counting settlements again would be double-counting.

	return balance, nil
}
