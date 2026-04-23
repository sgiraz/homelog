package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sgiraz/homelog/internal/models"
)

// deleteExpenseForInstallment removes the auto-created expense (and its splits)
// linked to a given installment. Safe to call when no expense exists.
func (h *UtilityHandler) deleteExpenseForInstallment(inst *models.BillInstallment) {
	var linked models.Expense
	if err := h.db.Where("bill_installment_id = ?", inst.ID).First(&linked).Error; err != nil {
		return
	}
	h.db.Where("expense_id = ?", linked.ID).Delete(&models.ExpenseSplit{})
	h.db.Delete(&linked)
	log.Printf("🗑️  Auto-deleted expense ID=%d linked to installment ID=%d", linked.ID, inst.ID)
}

// autoCreateExpenseFromBill creates an expense automatically when a bill is marked as paid.
// Split logic: uses the paying user's default_split_with_member_ids setting.
// If no default split configured, expense is created without split.
// All split members (payer included) get is_settled=false.
// detectPriceChange compares the new bill's amount with the previous bill
// and creates a PriceChange record if the amount differs.
func (h *UtilityHandler) detectPriceChange(utility *models.Utility, newBill *models.Bill) {
	// Find the previous bill by period_start (the one just before this bill's period)
	var prevBill models.Bill
	err := h.db.Where("utility_id = ? AND id != ? AND period_start < ?",
		utility.ID, newBill.ID, newBill.PeriodStart).
		Order("period_start DESC").
		First(&prevBill).Error
	if err != nil {
		return // No previous bill to compare
	}

	if prevBill.AmountTotal == newBill.AmountTotal {
		return // Same price
	}

	// Check if this change was already recorded
	var existing models.PriceChange
	err = h.db.Where("utility_id = ? AND source_bill_id = ?", utility.ID, newBill.ID).
		First(&existing).Error
	if err == nil {
		return // Already recorded
	}

	change := models.PriceChange{
		UtilityID:    utility.ID,
		EffectiveDate: newBill.PeriodStart,
		OldAmount:    prevBill.AmountTotal,
		NewAmount:    newBill.AmountTotal,
		SourceBillID: &newBill.ID,
	}

	if err := h.db.Create(&change).Error; err != nil {
		log.Printf("⚠️  Failed to create price change for utility %d: %v", utility.ID, err)
		return
	}

	// Update utility recurring_amount to latest price
	h.db.Model(utility).Update("recurring_amount", newBill.AmountTotal)
	log.Printf("💰 Price change detected for utility %d: %.2f → %.2f", utility.ID, prevBill.AmountTotal, newBill.AmountTotal)
}

// autoCreateExpenseFromInstallment creates an expense automatically for a single installment.
// Non-installment bills carry exactly one installment so the flow is uniform.
func (h *UtilityHandler) autoCreateExpenseFromInstallment(userID uint, inst *models.BillInstallment) error {
	// Load bill
	var bill models.Bill
	if err := h.db.First(&bill, inst.BillID).Error; err != nil {
		return fmt.Errorf("could not load bill: %w", err)
	}
	// Count installments (for "Rata N/M" description)
	var installmentCount int64
	h.db.Model(&models.BillInstallment{}).Where("bill_id = ?", bill.ID).Count(&installmentCount)

	// Load utility
	var utility models.Utility
	if err := h.db.First(&utility, bill.UtilityID).Error; err != nil {
		return fmt.Errorf("could not load utility: %w", err)
	}

	// Determine the payer: use utility's configured PaidByMemberID if set, otherwise the logged-in user
	var payerMember models.HouseholdMember
	payerUserID := userID // default: logged-in user owns the expense
	if utility.PaidByMemberID != nil {
		// Use the configured default payer for this service (constrained to same property)
		if err := h.db.Preload("User").
			Where("id = ? AND property_id = ?", *utility.PaidByMemberID, utility.PropertyID).
			First(&payerMember).Error; err != nil {
			return fmt.Errorf("configured payer member %d not found in property %d: %w", *utility.PaidByMemberID, utility.PropertyID, err)
		}
		if payerMember.UserID != nil {
			payerUserID = *payerMember.UserID
		}
	} else {
		// Fallback: find the logged-in user's member for this property
		if err := h.db.Where("property_id = ? AND user_id = ?", utility.PropertyID, userID).
			First(&payerMember).Error; err != nil {
			return fmt.Errorf("user has no member profile for property %d: %w", utility.PropertyID, err)
		}
	}

	// Find the "Casa" category
	var casaCategory models.Category
	if err := h.db.Where("name = ? AND is_default = true", "Casa").First(&casaCategory).Error; err != nil {
		return fmt.Errorf("could not find 'Casa' category: %w", err)
	}

	// Find the "Utenze" subcategory under Casa
	var utenzeSubcat models.Subcategory
	_ = h.db.Where("category_id = ? AND name = ?", casaCategory.ID, "Utenze").First(&utenzeSubcat)

	// Build description: "Bolletta Luce - Mar 2025"
	typeNames := map[string]string{
		"electricity": "Luce",
		"gas":         "Gas",
		"water":       "Acqua",
		"waste":       "Rifiuti",
	}
	italianMonths := []string{"Gen", "Feb", "Mar", "Apr", "Mag", "Giu", "Lug", "Ago", "Set", "Ott", "Nov", "Dic"}
	typeName := typeNames[utility.Type]
	if typeName == "" {
		typeName = utility.Type
	}
	month := italianMonths[bill.PeriodEnd.Month()-1]
	year := bill.PeriodEnd.Year()
	description := fmt.Sprintf("Bolletta %s - %s %d", typeName, month, year)
	if installmentCount > 1 {
		description = fmt.Sprintf("%s (Rata %d/%d)", description, inst.Number, installmentCount)
	}

	// Payment date: use installment's paid_at or today
	expenseDate := time.Now()
	if inst.PaidAt != nil {
		expenseDate = *inst.PaidAt
	}

	// Subcategory ID (optional)
	var subcatID *uint
	if utenzeSubcat.ID != 0 {
		id := utenzeSubcat.ID
		subcatID = &id
	}

	// Determine split behavior: per-service override takes priority over global settings
	var splitMemberIDs []uint
	isSplit := false

	switch utility.SplitOverride {
	case "no_split":
		// Explicitly disabled for this service — no split regardless of global settings
		isSplit = false

	case "custom":
		// Per-service custom split members
		if utility.SplitMemberIDs != "" {
			if jsonErr := json.Unmarshal([]byte(utility.SplitMemberIDs), &splitMemberIDs); jsonErr != nil {
				splitMemberIDs = nil
			}
		}
		if len(splitMemberIDs) > 0 {
			isSplit = true
		}

	default:
		// "" = use global household split mode + payer's default split members
		var householdSettings models.HouseholdSettings
		splitMode := false
		if err := h.db.Where("property_id = ?", utility.PropertyID).First(&householdSettings).Error; err == nil {
			splitMode = householdSettings.SplitMode
		}
		if splitMode {
			var userSettings models.UserSettings
			if err := h.db.Where("user_id = ?", payerUserID).First(&userSettings).Error; err == nil {
				if userSettings.DefaultSplitWithMemberIDs != "" {
					if jsonErr := json.Unmarshal([]byte(userSettings.DefaultSplitWithMemberIDs), &splitMemberIDs); jsonErr != nil {
						splitMemberIDs = nil
					}
				}
			}
			if len(splitMemberIDs) > 0 {
				isSplit = true
			}
		}
	}

	// Propagate the bill's original-currency snapshot to the expense, scaled by
	// this installment's share of the bill total. Kept lossless: scaling is a
	// pure float multiply of audit data — no FX conversion on this path.
	var originalAmount *float64
	originalCurrency := ""
	if bill.OriginalAmount != nil && bill.OriginalCurrency != "" && bill.AmountTotal > 0 {
		scaled := *bill.OriginalAmount * (inst.Amount / bill.AmountTotal)
		originalAmount = &scaled
		originalCurrency = bill.OriginalCurrency
	}

	billID := bill.ID
	propID := utility.PropertyID
	instID := inst.ID
	expense := models.Expense{
		UserID:            userID, // logged-in user for visibility; PaidByMemberID tracks the actual payer
		PropertyID:        &propID,
		CategoryID:        casaCategory.ID,
		SubcategoryID:     subcatID,
		BillID:            &billID,
		BillInstallmentID: &instID,
		Amount:            inst.Amount,
		OriginalAmount:    originalAmount,
		OriginalCurrency:  originalCurrency,
		Date:              expenseDate,
		Description:       description,
		PaidByMemberID:    payerMember.ID,
		IsSplit:           isSplit,
	}

	if err := h.db.Create(&expense).Error; err != nil {
		return fmt.Errorf("could not create expense: %w", err)
	}

	// Create split records: payer split is already settled (owes nothing to self),
	// other members' splits are unsettled until a Settlement is recorded.
	if isSplit {
		// Build full member list: payer + split-with members (dedup)
		allMemberIDs := make([]uint, 0, len(splitMemberIDs)+1)
		allMemberIDs = append(allMemberIDs, payerMember.ID)
		for _, mid := range splitMemberIDs {
			if mid != payerMember.ID {
				allMemberIDs = append(allMemberIDs, mid)
			}
		}
		splitAmount := inst.Amount / float64(len(allMemberIDs))
		now := time.Now()
		for _, mid := range allMemberIDs {
			split := models.ExpenseSplit{
				ExpenseID: expense.ID,
				MemberID:  mid,
				Amount:    splitAmount,
				IsSettled: mid == payerMember.ID,
			}
			if split.IsSettled {
				split.SettledAt = &now
			}
			h.db.Create(&split)
		}
		log.Printf("✅ Auto-created expense ID=%d '%s' €%.2f for bill ID=%d inst ID=%d (split among %d members)", expense.ID, description, expense.Amount, bill.ID, inst.ID, len(allMemberIDs))
	} else {
		log.Printf("✅ Auto-created expense ID=%d '%s' €%.2f for bill ID=%d inst ID=%d (no split)", expense.ID, description, expense.Amount, bill.ID, inst.ID)
	}

	// Back-link the expense on the installment
	expenseID := expense.ID
	inst.ExpenseID = &expenseID
	h.db.Model(inst).Update("expense_id", expenseID)
	return nil
}

// RunDomiciliationSweep finds unpaid installments of domiciled utilities whose
// due_date has passed and auto-marks them paid, creating the linked expense.
// Safe to call repeatedly — a second pass finds nothing to do.
func (h *UtilityHandler) RunDomiciliationSweep() {
	now := time.Now()
	var installments []models.BillInstallment
	// Join bills → utilities to find domiciled services with due installments
	err := h.db.
		Joins("JOIN bills ON bills.id = bill_installments.bill_id AND bills.deleted_at IS NULL").
		Joins("JOIN utilities ON utilities.id = bills.utility_id AND utilities.deleted_at IS NULL").
		Where("bill_installments.is_paid = ?", false).
		Where("bill_installments.due_date <= ?", now).
		Where("utilities.is_domiciled = ?", true).
		Where("utilities.is_active = ?", true).
		Find(&installments).Error
	if err != nil {
		log.Printf("⚠️  Domiciliation sweep query failed: %v", err)
		return
	}
	if len(installments) == 0 {
		return
	}
	log.Printf("🔁 Domiciliation sweep: %d installment(s) to auto-pay", len(installments))
	for i := range installments {
		inst := &installments[i]
		// Pick a user to attribute the expense to: fall back to utility.paid_by_member's user
		var bill models.Bill
		if err := h.db.First(&bill, inst.BillID).Error; err != nil {
			continue
		}
		var utility models.Utility
		if err := h.db.First(&utility, bill.UtilityID).Error; err != nil {
			continue
		}
		var userID uint
		if utility.PaidByMemberID != nil {
			var member models.HouseholdMember
			if err := h.db.First(&member, *utility.PaidByMemberID).Error; err == nil && member.UserID != nil {
				userID = *member.UserID
			}
		}
		if userID == 0 {
			// Fallback: any member of the property
			var member models.HouseholdMember
			if err := h.db.Where("property_id = ? AND user_id IS NOT NULL", utility.PropertyID).
				First(&member).Error; err == nil && member.UserID != nil {
				userID = *member.UserID
			}
		}
		if userID == 0 {
			log.Printf("⚠️  Domiciliation sweep: could not determine user for installment %d", inst.ID)
			continue
		}

		inst.IsPaid = true
		paidAt := inst.DueDate
		if paidAt.After(now) {
			paidAt = now
		}
		inst.PaidAt = &paidAt
		h.db.Save(inst)
		if err := h.autoCreateExpenseFromInstallment(userID, inst); err != nil {
			log.Printf("⚠️  Domiciliation sweep: auto-create expense failed for installment %d: %v", inst.ID, err)
			continue
		}
		// Refresh bill paid state
		if err := h.db.First(&bill, inst.BillID).Error; err == nil {
			h.syncBillPaidState(&bill)
		}
	}
}
