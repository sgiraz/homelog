package handlers

import (
	"log"
	"math"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

// AddBill adds a bill for a utility
func (h *UtilityHandler) AddBill(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid utility ID"})
		return
	}

	// Verify access to utility
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", utilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	var input struct {
		BillNumber          string     `json:"bill_number" binding:"required"`
		UserReadingID       *uint      `json:"user_reading_id"`
		IssueDate           time.Time  `json:"issue_date" binding:"required"`
		PeriodStart         time.Time  `json:"period_start" binding:"required"`
		PeriodEnd           time.Time  `json:"period_end" binding:"required"`
		DueDate             time.Time  `json:"due_date" binding:"required"`
		ReadingStartDate    *time.Time `json:"reading_start_date"`
		ReadingStartValue   *float64   `json:"reading_start_value"`
		ReadingEndDate      *time.Time `json:"reading_end_date"`
		ReadingEndValue     *float64   `json:"reading_end_value"`
		ReadingType         string     `json:"reading_type"`
		ProviderReadingDate *time.Time `json:"provider_reading_date"`
		ProviderReadingF1   *float64   `json:"provider_reading_f1"`
		ProviderReadingF2   *float64   `json:"provider_reading_f2"`
		ProviderReadingF3   *float64   `json:"provider_reading_f3"`
		ProviderReading     *float64   `json:"provider_reading"`
		ConsumptionTotal      float64    `json:"consumption_total"`
		ConsumptionF1         *float64   `json:"consumption_f1"`
		ConsumptionF2         *float64   `json:"consumption_f2"`
		ConsumptionF3         *float64   `json:"consumption_f3"`
		AmountTotal           float64    `json:"amount_total"`
		// Original-currency snapshot. When the utility bills in a non-user currency
		// the frontend converts via exchangeAPI and sends both figures; server
		// persists them as-is. AmountTotal is always in user currency.
		OriginalAmount        *float64   `json:"original_amount"`
		OriginalCurrency      string     `json:"original_currency"`
		ConversionCoefficient *float64   `json:"conversion_coefficient"`
		EstimatedDate        *time.Time `json:"estimated_date"`
		EstimatedReading     *float64   `json:"estimated_reading"`
		EstimatedConsumption *float64   `json:"estimated_consumption"`
		AmountEnergy        *float64   `json:"amount_energy"`
		AmountFixed         *float64   `json:"amount_fixed"`
		AmountTaxes         *float64   `json:"amount_taxes"`
		AmountVAT           *float64   `json:"amount_vat"`
		IsPaid              bool       `json:"is_paid"`
		PaidDate            *time.Time `json:"paid_date"`
		PDFURL              string     `json:"pdf_url"`
		// Communication (optional note from bill/invoice)
		CommunicationText string `json:"communication_text"`
		// Installments: optional — when the utility is installment-based, the client
		// provides the breakdown. When empty, one implicit installment is created.
		Installments []struct {
			Number  int       `json:"number"`
			DueDate time.Time `json:"due_date"`
			Amount  float64   `json:"amount"`
			IsPaid  bool      `json:"is_paid"`
			PaidAt  *time.Time `json:"paid_at"`
		} `json:"installments"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate installment breakdown when the service is installment-based.
	if utility.IsInstallmentBased && len(input.Installments) > 1 {
		var sum float64
		for _, in := range input.Installments {
			sum += in.Amount
		}
		if math.Abs(sum-input.AmountTotal) > 0.01 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Installment amounts do not sum to bill total"})
			return
		}
		// Bill due_date = first installment due_date
		input.DueDate = input.Installments[0].DueDate
	}

	bill := models.Bill{
		UtilityID:             uint(utilityID),
		BillNumber:            input.BillNumber,
		UserReadingID:         input.UserReadingID,
		IssueDate:             input.IssueDate,
		PeriodStart:           input.PeriodStart,
		PeriodEnd:             input.PeriodEnd,
		DueDate:               input.DueDate,
		ReadingStartDate:      input.ReadingStartDate,
		ReadingStartValue:     input.ReadingStartValue,
		ReadingEndDate:        input.ReadingEndDate,
		ReadingEndValue:       input.ReadingEndValue,
		ReadingType:           input.ReadingType,
		ProviderReadingDate:   input.ProviderReadingDate,
		ProviderReadingF1:     input.ProviderReadingF1,
		ProviderReadingF2:     input.ProviderReadingF2,
		ProviderReadingF3:     input.ProviderReadingF3,
		ProviderReading:       input.ProviderReading,
		ConversionCoefficient: input.ConversionCoefficient,
		EstimatedDate:         input.EstimatedDate,
		EstimatedReading:      input.EstimatedReading,
		EstimatedConsumption:  input.EstimatedConsumption,
		ConsumptionTotal:      input.ConsumptionTotal,
		ConsumptionF1:         input.ConsumptionF1,
		ConsumptionF2:         input.ConsumptionF2,
		ConsumptionF3:         input.ConsumptionF3,
		AmountTotal:           input.AmountTotal,
		OriginalAmount:        input.OriginalAmount,
		OriginalCurrency:      input.OriginalCurrency,
		AmountEnergy:          input.AmountEnergy,
		AmountFixed:           input.AmountFixed,
		AmountTaxes:           input.AmountTaxes,
		AmountVAT:             input.AmountVAT,
		IsPaid:                input.IsPaid,
		PaidDate:              input.PaidDate,
		PDFURL:                input.PDFURL,
	}

	if err := h.db.Create(&bill).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add bill"})
		return
	}

	// Load utility + user reading relations
	h.db.Preload("Utility").Preload("UserReading").First(&bill, bill.ID)

	// Create ServiceCommunication if communication text was provided
	if input.CommunicationText != "" {
		comm := models.ServiceCommunication{
			UtilityID:   uint(utilityID),
			BillID:      &bill.ID,
			Type:        "info",
			Title:       "Comunicazione",
			Content:     input.CommunicationText,
			IsImportant: true,
		}
		if err := h.db.Create(&comm).Error; err != nil {
			log.Printf("⚠️  Failed to create communication for bill %d: %v", bill.ID, err)
		}
	}

	// Detect price changes for fixed-cost services
	if !utility.IsMetered {
		h.detectPriceChange(&utility, &bill)
	}

	// Create installments. Non-installment services (or missing breakdown) get
	// a single implicit installment carrying the full bill, so the payment flow
	// is uniform.
	var installments []models.BillInstallment
	if utility.IsInstallmentBased && len(input.Installments) > 0 {
		for _, in := range input.Installments {
			installments = append(installments, models.BillInstallment{
				BillID:  bill.ID,
				Number:  in.Number,
				DueDate: in.DueDate,
				Amount:  in.Amount,
				IsPaid:  in.IsPaid,
				PaidAt:  in.PaidAt,
			})
		}
	} else {
		var paidAt *time.Time
		if input.IsPaid {
			if input.PaidDate != nil {
				paidAt = input.PaidDate
			} else {
				now := time.Now()
				paidAt = &now
			}
		}
		installments = append(installments, models.BillInstallment{
			BillID:  bill.ID,
			Number:  1,
			DueDate: bill.DueDate,
			Amount:  bill.AmountTotal,
			IsPaid:  input.IsPaid,
			PaidAt:  paidAt,
		})
	}
	for i := range installments {
		if err := h.db.Create(&installments[i]).Error; err != nil {
			log.Printf("⚠️  Failed to create installment for bill %d: %v", bill.ID, err)
			continue
		}
		if installments[i].IsPaid {
			if err := h.autoCreateExpenseFromInstallment(userID, &installments[i]); err != nil {
				log.Printf("⚠️  Failed to auto-create expense for bill %d inst %d: %v", bill.ID, installments[i].ID, err)
			}
		}
	}

	// Sync bill.IsPaid with installments aggregate state
	h.syncBillPaidState(&bill)

	c.JSON(http.StatusCreated, bill)
}

// syncBillPaidState recomputes bill.IsPaid / bill.PaidDate from its installments
// and persists the change. For installment-based bills this is the source of truth.
func (h *UtilityHandler) syncBillPaidState(bill *models.Bill) {
	var installments []models.BillInstallment
	if err := h.db.Where("bill_id = ?", bill.ID).Find(&installments).Error; err != nil || len(installments) == 0 {
		return
	}
	allPaid := true
	var maxPaidAt *time.Time
	for _, in := range installments {
		if !in.IsPaid {
			allPaid = false
		}
		if in.PaidAt != nil && (maxPaidAt == nil || in.PaidAt.After(*maxPaidAt)) {
			t := *in.PaidAt
			maxPaidAt = &t
		}
	}
	updates := map[string]any{"is_paid": allPaid}
	if allPaid {
		updates["paid_date"] = maxPaidAt
	} else {
		updates["paid_date"] = nil
	}
	h.db.Model(bill).Updates(updates)
	bill.IsPaid = allPaid
	if allPaid {
		bill.PaidDate = maxPaidAt
	} else {
		bill.PaidDate = nil
	}
}

// isInstallmentLocked reports whether the installment's auto-expense has any
// non-payer split that is already settled. A locked installment must not be
// toggled paid→unpaid (would destroy settled splits) nor have its bill amount
// edited (would silently change the household balance).
//
// Matches expenses linked via bill_installment_id; for legacy single-installment
// bills (where the auto-expense was created before bill_installment_id existed)
// also matches by bill_id when this is the only installment.
func (h *UtilityHandler) isInstallmentLocked(instID uint) bool {
	var inst models.BillInstallment
	if err := h.db.First(&inst, instID).Error; err != nil {
		return false
	}
	var siblings int64
	h.db.Model(&models.BillInstallment{}).Where("bill_id = ?", inst.BillID).Count(&siblings)

	q := h.db.Model(&models.ExpenseSplit{}).
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id AND expenses.deleted_at IS NULL").
		Where("expense_splits.is_settled = ?", true).
		Where("expense_splits.member_id != expenses.paid_by_member_id")
	if siblings <= 1 {
		q = q.Where("expenses.bill_installment_id = ? OR (expenses.bill_installment_id IS NULL AND expenses.bill_id = ?)", instID, inst.BillID)
	} else {
		q = q.Where("expenses.bill_installment_id = ?", instID)
	}
	var count int64
	q.Count(&count)
	return count > 0
}

// isUtilityCurrencyLocked reports whether this utility has any bill with a
// settled non-payer split — i.e. a cross-member settlement already happened.
// When true, changing the utility's currency would silently rewrite historical
// balances, so the handler must refuse it.
func (h *UtilityHandler) isUtilityCurrencyLocked(utilityID uint) bool {
	var count int64
	h.db.Model(&models.ExpenseSplit{}).
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id AND expenses.deleted_at IS NULL").
		Joins("JOIN bills ON bills.id = expenses.bill_id AND bills.deleted_at IS NULL").
		Where("bills.utility_id = ?", utilityID).
		Where("expense_splits.is_settled = ?", true).
		Where("expense_splits.member_id != expenses.paid_by_member_id").
		Count(&count)
	return count > 0
}

// isBillLocked reports whether any expense linked to the bill has a non-payer
// settled split. Matches via bill_id directly so legacy expenses without a
// bill_installment_id are still detected.
func (h *UtilityHandler) isBillLocked(billID uint) bool {
	var count int64
	h.db.Model(&models.ExpenseSplit{}).
		Joins("JOIN expenses ON expenses.id = expense_splits.expense_id AND expenses.deleted_at IS NULL").
		Where("expenses.bill_id = ?", billID).
		Where("expense_splits.is_settled = ?", true).
		Where("expense_splits.member_id != expenses.paid_by_member_id").
		Count(&count)
	return count > 0
}

// populateBillLockState fills the IsLocked virtual field on each bill and its
// preloaded installments. Caller must have preloaded Installments.
func (h *UtilityHandler) populateBillLockState(bills []models.Bill) {
	for i := range bills {
		billLocked := false
		for j := range bills[i].Installments {
			if h.isInstallmentLocked(bills[i].Installments[j].ID) {
				bills[i].Installments[j].IsLocked = true
				billLocked = true
			}
		}
		bills[i].IsLocked = billLocked
	}
}

// GetBills returns all bills for a utility
func (h *UtilityHandler) GetBills(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid utility ID"})
		return
	}

	// Verify access to utility
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", utilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	var bills []models.Bill
	if err := h.db.Where("utility_id = ?", utilityID).
		Preload("UserReading").
		Preload("Installments", func(db *gorm.DB) *gorm.DB { return db.Order("number ASC") }).
		Order("period_end DESC").
		Find(&bills).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bills"})
		return
	}

	h.populateBillLockState(bills)
	c.JSON(http.StatusOK, bills)
}

// UpdateBill updates a bill
func (h *UtilityHandler) UpdateBill(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	billID, err := strconv.ParseUint(c.Param("billId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bill ID"})
		return
	}

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Find the bill and verify access through utility
	var bill models.Bill
	if err := h.db.Preload("Utility").First(&bill, billID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bill not found"})
		return
	}

	if !slices.Contains(memberPropertyIDs, bill.Utility.PropertyID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this bill"})
		return
	}

	var input struct {
		IsPaid   *bool      `json:"is_paid"`
		PaidDate *time.Time `json:"paid_date"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.IsPaid == nil {
		c.JSON(http.StatusOK, bill)
		return
	}

	// Load installments; this toggle endpoint only handles single-installment bills.
	var installments []models.BillInstallment
	h.db.Where("bill_id = ?", bill.ID).Order("number ASC").Find(&installments)
	if len(installments) > 1 {
		// Installment-based bills: ignore bulk toggle, state is derived from installments.
		h.syncBillPaidState(&bill)
		c.JSON(http.StatusOK, bill)
		return
	}
	if len(installments) == 0 {
		// Repair missing installment (legacy rows)
		inst := models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: bill.DueDate, Amount: bill.AmountTotal}
		h.db.Create(&inst)
		installments = []models.BillInstallment{inst}
	}

	inst := &installments[0]
	wasPaid := inst.IsPaid
	if *input.IsPaid && !wasPaid {
		inst.IsPaid = true
		if input.PaidDate != nil {
			inst.PaidAt = input.PaidDate
		} else {
			now := time.Now()
			inst.PaidAt = &now
		}
		h.db.Save(inst)
		if err := h.autoCreateExpenseFromInstallment(userID, inst); err != nil {
			log.Printf("⚠️  Failed to auto-create expense for bill %d: %v", bill.ID, err)
		}
	} else if !*input.IsPaid && wasPaid {
		if h.isInstallmentLocked(inst.ID) {
			c.JSON(http.StatusConflict, gin.H{"error": "La spesa collegata è già stata saldata da uno o più membri. Annulla i pagamenti dal Bilancio prima di modificare lo stato della bolletta."})
			return
		}
		inst.IsPaid = false
		inst.PaidAt = nil
		h.deleteExpenseForInstallment(inst)
		inst.ExpenseID = nil
		h.db.Save(inst)
	}

	h.syncBillPaidState(&bill)
	c.JSON(http.StatusOK, bill)
}


// DeleteBill removes a bill
func (h *UtilityHandler) DeleteBill(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	billID, err := strconv.ParseUint(c.Param("billId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bill ID"})
		return
	}

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Find the bill and verify access through utility
	var bill models.Bill
	if err := h.db.Preload("Utility").First(&bill, billID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bill not found"})
		return
	}

	if !slices.Contains(memberPropertyIDs, bill.Utility.PropertyID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this bill"})
		return
	}

	if h.isBillLocked(bill.ID) {
		c.JSON(http.StatusConflict, gin.H{"error": "La spesa collegata a questa bolletta è già stata saldata. Annulla i pagamenti dal Bilancio prima di eliminare la bolletta."})
		return
	}

	if err := h.db.Delete(&bill).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bill"})
		return
	}

	// Cascade delete: installments + linked expenses (across all installments) + splits
	bid := uint(billID)
	var linkedExpenses []models.Expense
	h.db.Where("bill_id = ?", bid).Find(&linkedExpenses)
	for _, e := range linkedExpenses {
		h.db.Where("expense_id = ?", e.ID).Delete(&models.ExpenseSplit{})
		h.db.Delete(&e)
		log.Printf("🗑️  Deleted auto-expense ID=%d linked to bill ID=%d", e.ID, billID)
	}
	h.db.Where("bill_id = ?", bid).Delete(&models.BillInstallment{})

	c.JSON(http.StatusOK, gin.H{"message": "Bill deleted successfully"})
}

// UpdateBillInstallment marks a single installment as paid/unpaid.
// On paid: creates an expense via autoCreateExpenseFromInstallment.
// On unpaid: deletes the auto-created expense + splits.
// After the change, bill.IsPaid / bill.PaidDate are recomputed from all installments.
func (h *UtilityHandler) UpdateBillInstallment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	billID, err := strconv.ParseUint(c.Param("billId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bill ID"})
		return
	}
	instID, err := strconv.ParseUint(c.Param("instId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid installment ID"})
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var bill models.Bill
	if err := h.db.Preload("Utility").First(&bill, billID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bill not found"})
		return
	}
	if !slices.Contains(memberPropertyIDs, bill.Utility.PropertyID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	var inst models.BillInstallment
	if err := h.db.Where("id = ? AND bill_id = ?", instID, billID).First(&inst).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Installment not found"})
		return
	}

	var input struct {
		IsPaid bool       `json:"is_paid"`
		PaidAt *time.Time `json:"paid_at"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wasPaid := inst.IsPaid
	if input.IsPaid && !wasPaid {
		inst.IsPaid = true
		if input.PaidAt != nil {
			inst.PaidAt = input.PaidAt
		} else {
			now := time.Now()
			inst.PaidAt = &now
		}
		h.db.Save(&inst)
		if err := h.autoCreateExpenseFromInstallment(userID, &inst); err != nil {
			log.Printf("⚠️  Failed to auto-create expense for installment %d: %v", inst.ID, err)
		}
	} else if !input.IsPaid && wasPaid {
		if h.isInstallmentLocked(inst.ID) {
			c.JSON(http.StatusConflict, gin.H{"error": "La spesa collegata a questa rata è già stata saldata da uno o più membri. Annulla i pagamenti dal Bilancio prima di modificare lo stato della rata."})
			return
		}
		inst.IsPaid = false
		inst.PaidAt = nil
		h.deleteExpenseForInstallment(&inst)
		inst.ExpenseID = nil
		h.db.Save(&inst)
	}

	h.syncBillPaidState(&bill)
	// Return fresh bill with installments for the client
	h.db.Preload("Installments").First(&bill, bill.ID)
	c.JSON(http.StatusOK, bill)
}

// UpdateBillFull updates all fields of a bill
func (h *UtilityHandler) UpdateBillFull(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	billID, err := strconv.ParseUint(c.Param("billId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bill ID"})
		return
	}

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Find the bill and verify access through utility
	var bill models.Bill
	if err := h.db.Preload("Utility").First(&bill, billID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bill not found"})
		return
	}

	if !slices.Contains(memberPropertyIDs, bill.Utility.PropertyID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this bill"})
		return
	}

	var input struct {
		BillNumber            string     `json:"bill_number" binding:"required"`
		UserReadingID         *uint      `json:"user_reading_id"`
		IssueDate             time.Time  `json:"issue_date"`
		PeriodStart           time.Time  `json:"period_start"`
		PeriodEnd             time.Time  `json:"period_end"`
		DueDate               time.Time  `json:"due_date"`
		ConsumptionTotal      float64    `json:"consumption_total"`
		AmountTotal           float64    `json:"amount_total"`
		OriginalAmount        *float64   `json:"original_amount"`
		OriginalCurrency      string     `json:"original_currency"`
		ConversionCoefficient *float64   `json:"conversion_coefficient"`
		EstimatedDate        *time.Time `json:"estimated_date"`
		EstimatedReading     *float64   `json:"estimated_reading"`
		EstimatedConsumption *float64   `json:"estimated_consumption"`
		ReadingType           string     `json:"reading_type"`
		IsPaid                bool       `json:"is_paid"`
		PaidDate              *time.Time `json:"paid_date"`
		ProviderReadingDate   *time.Time `json:"provider_reading_date"`
		ProviderReadingF1     *float64   `json:"provider_reading_f1"`
		ProviderReadingF2     *float64   `json:"provider_reading_f2"`
		ProviderReadingF3     *float64   `json:"provider_reading_f3"`
		ProviderReading       *float64   `json:"provider_reading"`
		// Communication (optional note from bill/invoice)
		CommunicationText string `json:"communication_text"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load installments to decide how to handle the is_paid toggle.
	var installments []models.BillInstallment
	h.db.Where("bill_id = ?", bill.ID).Order("number ASC").Find(&installments)

	// Per-field lock: a locked bill can still be edited (metadata, dates,
	// readings, communication...) but the amount and the single-installment
	// paid toggle would corrupt already-settled splits.
	if h.isBillLocked(bill.ID) {
		if input.AmountTotal != bill.AmountTotal {
			c.JSON(http.StatusConflict, gin.H{"error": "Bolletta saldata: l'importo totale non può essere modificato. Annulla i pagamenti dal Bilancio per sbloccare il campo."})
			return
		}
		if len(installments) <= 1 && input.IsPaid != bill.IsPaid {
			c.JSON(http.StatusConflict, gin.H{"error": "Bolletta saldata: lo stato di pagamento non può essere modificato. Annulla i pagamenti dal Bilancio per sbloccarlo."})
			return
		}
	}

	// Update all fields (except is_paid/paid_date: those are derived from installments)
	bill.BillNumber = input.BillNumber
	bill.UserReadingID = input.UserReadingID
	bill.IssueDate = input.IssueDate
	bill.PeriodStart = input.PeriodStart
	bill.PeriodEnd = input.PeriodEnd
	bill.DueDate = input.DueDate
	bill.ConsumptionTotal = input.ConsumptionTotal
	bill.AmountTotal = input.AmountTotal
	bill.OriginalAmount = input.OriginalAmount
	bill.OriginalCurrency = input.OriginalCurrency
	bill.ConversionCoefficient = input.ConversionCoefficient
	bill.EstimatedDate = input.EstimatedDate
	bill.EstimatedReading = input.EstimatedReading
	bill.EstimatedConsumption = input.EstimatedConsumption
	bill.ReadingType = input.ReadingType
	bill.ProviderReadingDate = input.ProviderReadingDate
	bill.ProviderReadingF1 = input.ProviderReadingF1
	bill.ProviderReadingF2 = input.ProviderReadingF2
	bill.ProviderReadingF3 = input.ProviderReadingF3
	bill.ProviderReading = input.ProviderReading

	if err := h.db.Save(&bill).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bill"})
		return
	}

	// For single-installment bills, propagate the is_paid toggle to the installment.
	// For installment-based bills (N>1), is_paid is derived from installments — ignore payload.
	if len(installments) <= 1 {
		if len(installments) == 0 {
			inst := models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: bill.DueDate, Amount: bill.AmountTotal}
			h.db.Create(&inst)
			installments = []models.BillInstallment{inst}
		}
		inst := &installments[0]
		// Keep single installment amount/due_date in sync with bill edits
		inst.Amount = bill.AmountTotal
		inst.DueDate = bill.DueDate
		wasPaid := inst.IsPaid
		if input.IsPaid && !wasPaid {
			inst.IsPaid = true
			if input.PaidDate != nil {
				inst.PaidAt = input.PaidDate
			} else {
				now := time.Now()
				inst.PaidAt = &now
			}
			h.db.Save(inst)
			if err := h.autoCreateExpenseFromInstallment(userID, inst); err != nil {
				log.Printf("⚠️  Failed to auto-create expense for bill %d: %v", bill.ID, err)
			}
		} else if !input.IsPaid && wasPaid {
			inst.IsPaid = false
			inst.PaidAt = nil
			h.deleteExpenseForInstallment(inst)
			inst.ExpenseID = nil
			h.db.Save(inst)
		} else {
			h.db.Save(inst)
		}
	}

	h.syncBillPaidState(&bill)

	// Handle communication text: update/create/delete
	var existingComm models.ServiceCommunication
	commExists := h.db.Where("bill_id = ?", bill.ID).First(&existingComm).Error == nil

	if input.CommunicationText != "" {
		if commExists {
			existingComm.Content = input.CommunicationText
			h.db.Save(&existingComm)
		} else {
			comm := models.ServiceCommunication{
				UtilityID:   bill.UtilityID,
				BillID:      &bill.ID,
				Type:        "info",
				Title:       "Comunicazione",
				Content:     input.CommunicationText,
				IsImportant: true,
			}
			h.db.Create(&comm)
		}
	} else if commExists {
		// Empty text means user wants to clear the communication
		h.db.Delete(&existingComm)
	}

	c.JSON(http.StatusOK, bill)
}

