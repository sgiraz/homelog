package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

type UtilityHandler struct {
	db *gorm.DB
}

func NewUtilityHandler(db *gorm.DB) *UtilityHandler {
	return &UtilityHandler{db: db}
}

// List returns all utilities for the user's properties
func (h *UtilityHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utilities []models.Utility
	query := h.db.Where("property_id IN ?", memberPropertyIDs)

	// Filter by type if specified
	if utilityType := c.Query("type"); utilityType != "" {
		query = query.Where("type = ?", utilityType)
	}

	// Filter by property if specified
	if propertyID := c.Query("property_id"); propertyID != "" {
		query = query.Where("property_id = ?", propertyID)
	}

	// Only active utilities by default
	if c.Query("include_inactive") != "true" {
		query = query.Where("is_active = ?", true)
	}

	// No LIMIT in these preloads. GORM loads an association for every parent in
	// a single query (`... WHERE utility_id IN (1,2,3) ORDER BY ... LIMIT 3`),
	// so a limit caps the whole result set, not each service: the newest rows
	// all land on one or two services and the rest come back empty — which also
	// silently broke the overview KPIs and the dashboard's overdue-bill brief,
	// both of which sum over these lists.
	//
	// Volume is kept in check by selecting only the columns the list actually
	// renders, leaving out parsed_data (the raw PDF extraction blob, by far the
	// heaviest and read only in the per-bill detail views).
	billListColumns := []string{
		"id", "created_at", "updated_at", "deleted_at", "utility_id", "bill_number",
		"issue_date", "period_start", "period_end", "due_date",
		"consumption_total", "amount_total", "is_paid", "paid_date",
		"original_amount", "original_currency",
	}
	if err := query.Preload("Property").
		Preload("Bills", func(db *gorm.DB) *gorm.DB {
			return db.Select(billListColumns).Order("period_end DESC")
		}).
		Preload("Readings", func(db *gorm.DB) *gorm.DB {
			return db.Order("reading_date DESC")
		}).
		Order("type ASC, provider ASC").
		Find(&utilities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch utilities"})
		return
	}

	for i := range utilities {
		h.populateBillLockState(utilities[i].Bills)
		utilities[i].IsCurrencyLocked = h.isUtilityCurrencyLocked(utilities[i].ID)
	}
	c.JSON(http.StatusOK, utilities)
}

// Create adds a new utility
func (h *UtilityHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		PropertyID          uint       `json:"property_id" binding:"required"`
		Type                string     `json:"type" binding:"required,oneof=electricity gas water waste internet insurance affitto mutuo"`
		Provider            string     `json:"provider" binding:"required"`
		CustomerCode        string     `json:"customer_code"`
		ServiceCode         string     `json:"service_code"`
		Address             string     `json:"address"`
		StartDate           time.Time  `json:"start_date"`
		EndDate             *time.Time `json:"end_date"`
		IsActive            bool       `json:"is_active"`
		IsMetered           *bool      `json:"is_metered"`
		PowerCapacity       float64    `json:"power_capacity"`
		RecurringAmount     *float64   `json:"recurring_amount"`
		BillingInterval     *int       `json:"billing_interval"`                                           // default 1
		BillingUnit         string     `json:"billing_unit" binding:"omitempty,oneof=day week month year"` // default "month"
		PaidByMemberID      *uint      `json:"paid_by_member_id"`
		DefaultCategoryID   *uint      `json:"default_category_id"`
		SplitOverride       string     `json:"split_override" binding:"omitempty,oneof=no_split custom"`
		SplitMemberIDs      string     `json:"split_member_ids"`
		CustomerPortal      string     `json:"customer_portal"`
		Notes               string     `json:"notes"`
		AllowsSelfReading   *bool      `json:"allows_self_reading"`  // nil = true (default)
		ComparisonThreshold *float64   `json:"comparison_threshold"` // nil = 2.0 (default) - soglia base stesso giorno
		ThresholdPerDay     *float64   `json:"threshold_per_day"`    // nil = 1.0 (default) - tolleranza per giorno
		IsDomiciled         bool       `json:"is_domiciled"`
		IsInstallmentBased  bool       `json:"is_installment_based"`
		Currency            string     `json:"currency"` // ISO code, empty = use user's household currency
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user is an admin of the property
	if !requirePropertyAdmin(c, h.db, userID, input.PropertyID) {
		return
	}

	// Validate paid_by_member_id belongs to this property
	if input.PaidByMemberID != nil {
		var payerCount int64
		h.db.Model(&models.HouseholdMember{}).
			Where("id = ? AND property_id = ?", *input.PaidByMemberID, input.PropertyID).
			Count(&payerCount)
		if payerCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payer member does not belong to this property"})
			return
		}
	}

	// Validate split_member_ids: must be valid JSON array of member IDs belonging to this property
	if input.SplitOverride == "custom" && input.SplitMemberIDs != "" {
		var memberIDs []uint
		if err := json.Unmarshal([]byte(input.SplitMemberIDs), &memberIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_member_ids must be a valid JSON array of member IDs"})
			return
		}
		if len(memberIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_member_ids cannot be empty when split_override is custom"})
			return
		}
		var validCount int64
		h.db.Model(&models.HouseholdMember{}).
			Where("id IN ? AND property_id = ?", memberIDs, input.PropertyID).
			Count(&validCount)
		if validCount != int64(len(memberIDs)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Some split member IDs do not belong to this property"})
			return
		}
	}

	// Determine if service is metered based on type (unless explicitly set)
	// waste is listed explicitly at false to record that it is NOT an oversight:
	// TARI is billed on surface area, so it is a fixed-cost service.
	meteredTypes := map[string]bool{"electricity": true, "gas": true, "water": true, "waste": false}
	isMetered := meteredTypes[input.Type]
	if input.IsMetered != nil {
		isMetered = *input.IsMetered
	}

	// Check if there's already an active service of the same type for this property
	var existingCount int64
	h.db.Model(&models.Utility{}).
		Where("property_id = ? AND type = ? AND is_active = ?", input.PropertyID, input.Type, true).
		Count(&existingCount)

	if existingCount > 0 {
		typeLabels := map[string]string{
			"electricity": "Luce", "gas": "Gas", "water": "Acqua", "waste": "Rifiuti",
			"internet": "Internet", "insurance": "Assicurazione", "affitto": "Affitto", "mutuo": "Mutuo",
		}
		label := typeLabels[input.Type]
		if label == "" {
			label = input.Type
		}
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Esiste già un servizio %s attivo per questa proprietà. Disattiva quello esistente prima di crearne uno nuovo.", label),
		})
		return
	}

	// Default allows_self_reading to true if not specified
	allowsSelfReading := true
	if input.AllowsSelfReading != nil {
		allowsSelfReading = *input.AllowsSelfReading
	}

	// Default comparison_threshold to 2.0 if not specified (base threshold for same-day readings)
	comparisonThreshold := 2.0
	if input.ComparisonThreshold != nil {
		comparisonThreshold = *input.ComparisonThreshold
	}

	// Default threshold_per_day to 1.0 if not specified
	thresholdPerDay := 1.0
	if input.ThresholdPerDay != nil {
		thresholdPerDay = *input.ThresholdPerDay
	}

	// Default billing frequency
	billingInterval := 1
	if input.BillingInterval != nil && *input.BillingInterval > 0 {
		billingInterval = *input.BillingInterval
	}
	billingUnit := "month"
	if input.BillingUnit != "" {
		billingUnit = input.BillingUnit
	}

	utility := models.Utility{
		UserID:              userID,
		PropertyID:          input.PropertyID,
		Type:                input.Type,
		Provider:            input.Provider,
		CustomerCode:        input.CustomerCode,
		ServiceCode:         input.ServiceCode,
		Address:             input.Address,
		StartDate:           input.StartDate,
		EndDate:             input.EndDate,
		IsActive:            true,
		IsMetered:           isMetered,
		PowerCapacity:       input.PowerCapacity,
		RecurringAmount:     input.RecurringAmount,
		BillingInterval:     billingInterval,
		BillingUnit:         billingUnit,
		PaidByMemberID:      input.PaidByMemberID,
		DefaultCategoryID:   input.DefaultCategoryID,
		SplitOverride:       input.SplitOverride,
		SplitMemberIDs:      input.SplitMemberIDs,
		CustomerPortal:      input.CustomerPortal,
		Notes:               input.Notes,
		AllowsSelfReading:   &allowsSelfReading,
		ComparisonThreshold: comparisonThreshold,
		ThresholdPerDay:     thresholdPerDay,
		IsDomiciled:         input.IsDomiciled,
		IsInstallmentBased:  input.IsInstallmentBased,
		Currency:            input.Currency,
	}

	if err := h.db.Create(&utility).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create utility"})
		return
	}

	// Load relations
	h.db.Preload("Property").First(&utility, utility.ID)

	c.JSON(http.StatusCreated, utility)
}

// Get returns a single utility by ID
func (h *UtilityHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid utility ID"})
		return
	}

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", id, memberPropertyIDs).
		Preload("Property").
		Preload("PaidByMember").
		Preload("DefaultBillTemplate").
		Preload("Bills", func(db *gorm.DB) *gorm.DB {
			return db.Order("period_end DESC")
		}).
		Preload("Bills.Installments", func(db *gorm.DB) *gorm.DB {
			return db.Order("number ASC")
		}).
		Preload("Readings", func(db *gorm.DB) *gorm.DB {
			return db.Order("reading_date DESC")
		}).
		Preload("PriceChanges", func(db *gorm.DB) *gorm.DB {
			return db.Order("effective_date DESC")
		}).
		First(&utility).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch utility"})
		}
		return
	}

	h.populateBillLockState(utility.Bills)
	utility.IsCurrencyLocked = h.isUtilityCurrencyLocked(utility.ID)
	c.JSON(http.StatusOK, utility)
}

// Update modifies an existing utility
func (h *UtilityHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid utility ID"})
		return
	}

	var utility models.Utility
	if err := h.db.First(&utility, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch utility"})
		}
		return
	}

	if !requirePropertyAdmin(c, h.db, userID, utility.PropertyID) {
		return
	}

	var input struct {
		Provider              string     `json:"provider"`
		CustomerCode          string     `json:"customer_code"`
		ServiceCode           string     `json:"service_code"`
		Address               string     `json:"address"`
		StartDate             *time.Time `json:"start_date"`
		EndDate               *time.Time `json:"end_date"`
		IsActive              *bool      `json:"is_active"`
		PowerCapacity         *float64   `json:"power_capacity"`
		CustomerPortal        string     `json:"customer_portal"`
		Notes                 string     `json:"notes"`
		AllowsSelfReading     *bool      `json:"allows_self_reading"`
		ComparisonThreshold   *float64   `json:"comparison_threshold"`
		ThresholdPerDay       *float64   `json:"threshold_per_day"`
		RecurringAmount       *float64   `json:"recurring_amount"`
		BillingInterval       *int       `json:"billing_interval"`
		BillingUnit           string     `json:"billing_unit" binding:"omitempty,oneof=day week month year"`
		PaidByMemberID        *uint      `json:"paid_by_member_id"`
		DefaultCategoryID     *uint      `json:"default_category_id"`
		DefaultBillTemplateID *uint      `json:"default_bill_template_id"`
		SplitOverride         *string    `json:"split_override"`
		SplitMemberIDs        *string    `json:"split_member_ids"`
		IsDomiciled           *bool      `json:"is_domiciled"`
		IsInstallmentBased    *bool      `json:"is_installment_based"`
		Currency              *string    `json:"currency"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate split_override values
	if input.SplitOverride != nil {
		switch *input.SplitOverride {
		case "", "no_split", "custom":
			// valid
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_override must be '', 'no_split', or 'custom'"})
			return
		}
	}

	// Validate paid_by_member_id belongs to this property
	if input.PaidByMemberID != nil {
		var payerCount int64
		h.db.Model(&models.HouseholdMember{}).
			Where("id = ? AND property_id = ?", *input.PaidByMemberID, utility.PropertyID).
			Count(&payerCount)
		if payerCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payer member does not belong to this property"})
			return
		}
	}

	// Validate split_member_ids when split_override is custom
	effectiveSplitOverride := utility.SplitOverride
	if input.SplitOverride != nil {
		effectiveSplitOverride = *input.SplitOverride
	}
	if effectiveSplitOverride == "custom" && input.SplitMemberIDs != nil && *input.SplitMemberIDs != "" {
		var memberIDs []uint
		if err := json.Unmarshal([]byte(*input.SplitMemberIDs), &memberIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_member_ids must be a valid JSON array of member IDs"})
			return
		}
		if len(memberIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "split_member_ids cannot be empty when split_override is custom"})
			return
		}
		var validCount int64
		h.db.Model(&models.HouseholdMember{}).
			Where("id IN ? AND property_id = ?", memberIDs, utility.PropertyID).
			Count(&validCount)
		if validCount != int64(len(memberIDs)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Some split member IDs do not belong to this property"})
			return
		}
	}

	// Update fields
	if input.Provider != "" {
		utility.Provider = input.Provider
	}
	if input.CustomerCode != "" {
		utility.CustomerCode = input.CustomerCode
	}
	if input.ServiceCode != "" {
		utility.ServiceCode = input.ServiceCode
	}
	if input.Address != "" {
		utility.Address = input.Address
	}
	if input.StartDate != nil {
		utility.StartDate = *input.StartDate
	}
	utility.EndDate = input.EndDate
	if input.IsActive != nil {
		// If trying to activate, check for existing active utility of same type
		if *input.IsActive && !utility.IsActive {
			var existingCount int64
			h.db.Model(&models.Utility{}).
				Where("property_id = ? AND type = ? AND is_active = ? AND id != ?",
					utility.PropertyID, utility.Type, true, utility.ID).
				Count(&existingCount)

			if existingCount > 0 {
				typeLabels := map[string]string{
					"electricity": "Luce",
					"gas":         "Gas",
					"water":       "Acqua",
					"waste":       "Rifiuti",
				}
				label := typeLabels[utility.Type]
				if label == "" {
					label = utility.Type
				}
				c.JSON(http.StatusConflict, gin.H{
					"error": fmt.Sprintf("Esiste già un'utenza %s attiva per questa proprietà. Disattiva quella esistente prima di attivare questa.", label),
				})
				return
			}
		}
		utility.IsActive = *input.IsActive
	}
	if input.PowerCapacity != nil {
		utility.PowerCapacity = *input.PowerCapacity
	}
	if input.CustomerPortal != "" {
		utility.CustomerPortal = input.CustomerPortal
	}
	utility.Notes = input.Notes
	if input.AllowsSelfReading != nil {
		utility.AllowsSelfReading = input.AllowsSelfReading
	}
	if input.ComparisonThreshold != nil {
		utility.ComparisonThreshold = *input.ComparisonThreshold
	}
	if input.ThresholdPerDay != nil {
		utility.ThresholdPerDay = *input.ThresholdPerDay
	}
	if input.RecurringAmount != nil {
		utility.RecurringAmount = input.RecurringAmount
	}
	if input.BillingInterval != nil && *input.BillingInterval > 0 {
		utility.BillingInterval = *input.BillingInterval
	}
	if input.BillingUnit != "" {
		utility.BillingUnit = input.BillingUnit
	}
	utility.PaidByMemberID = input.PaidByMemberID
	utility.DefaultCategoryID = input.DefaultCategoryID
	utility.DefaultBillTemplateID = input.DefaultBillTemplateID
	if input.SplitOverride != nil {
		utility.SplitOverride = *input.SplitOverride
	}
	if input.SplitMemberIDs != nil {
		utility.SplitMemberIDs = *input.SplitMemberIDs
	}
	if input.IsDomiciled != nil {
		utility.IsDomiciled = *input.IsDomiciled
	}
	if input.IsInstallmentBased != nil {
		utility.IsInstallmentBased = *input.IsInstallmentBased
	}
	if input.Currency != nil && *input.Currency != utility.Currency {
		if h.isUtilityCurrencyLocked(utility.ID) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Non puoi cambiare la valuta di un servizio con bollette già saldate: la conversione non può essere propagata sullo storico.",
			})
			return
		}
		utility.Currency = *input.Currency
	}

	if err := h.db.Save(&utility).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update utility"})
		return
	}

	// Reload with relations
	h.db.Preload("Property").
		Preload("PaidByMember").
		Preload("DefaultBillTemplate").
		Preload("Bills", func(db *gorm.DB) *gorm.DB {
			return db.Order("period_end DESC").Limit(3)
		}).
		Preload("Bills.Installments", func(db *gorm.DB) *gorm.DB {
			return db.Order("number ASC")
		}).
		Preload("Readings", func(db *gorm.DB) *gorm.DB {
			return db.Order("reading_date DESC").Limit(3)
		}).
		First(&utility, utility.ID)

	utility.IsCurrencyLocked = h.isUtilityCurrencyLocked(utility.ID)
	c.JSON(http.StatusOK, utility)
}

// Delete removes a utility
func (h *UtilityHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid utility ID"})
		return
	}

	var utility models.Utility
	if err := h.db.First(&utility, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch utility"})
		}
		return
	}

	if !requirePropertyAdmin(c, h.db, userID, utility.PropertyID) {
		return
	}

	if err := h.db.Delete(&utility).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete utility"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Utility deleted successfully"})
}
