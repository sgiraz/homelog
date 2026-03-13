package handlers

import (
	"encoding/json"
	"fmt"
	"log"
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

	if err := query.Preload("Property").
		Preload("Bills", func(db *gorm.DB) *gorm.DB {
			return db.Order("period_end DESC").Limit(3)
		}).
		Preload("Readings", func(db *gorm.DB) *gorm.DB {
			return db.Order("reading_date DESC").Limit(3)
		}).
		Order("type ASC, provider ASC").
		Find(&utilities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch utilities"})
		return
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
		BillingInterval     *int       `json:"billing_interval"`                                                        // default 1
		BillingUnit         string     `json:"billing_unit" binding:"omitempty,oneof=day week month year"`               // default "month"
		PaidByMemberID      *uint      `json:"paid_by_member_id"`
		DefaultCategoryID   *uint      `json:"default_category_id"`
		CustomerPortal      string     `json:"customer_portal"`
		Notes               string     `json:"notes"`
		AllowsSelfReading   *bool      `json:"allows_self_reading"`  // nil = true (default)
		ComparisonThreshold *float64   `json:"comparison_threshold"` // nil = 2.0 (default) - soglia base stesso giorno
		ThresholdPerDay     *float64   `json:"threshold_per_day"`    // nil = 1.0 (default) - tolleranza per giorno
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user is a member of the property
	var memberCount int64
	h.db.Model(&models.HouseholdMember{}).
		Where("property_id = ? AND user_id = ?", input.PropertyID, userID).
		Count(&memberCount)

	if memberCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this property"})
		return
	}

	// Determine if service is metered based on type (unless explicitly set)
	meteredTypes := map[string]bool{"electricity": true, "gas": true, "water": true, "waste": true}
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
		CustomerPortal:      input.CustomerPortal,
		Notes:               input.Notes,
		AllowsSelfReading:   &allowsSelfReading,
		ComparisonThreshold: comparisonThreshold,
		ThresholdPerDay:     thresholdPerDay,
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

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", id, memberPropertyIDs).
		First(&utility).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch utility"})
		}
		return
	}

	var input struct {
		Provider            string     `json:"provider"`
		CustomerCode        string     `json:"customer_code"`
		ServiceCode         string     `json:"service_code"`
		Address             string     `json:"address"`
		StartDate           *time.Time `json:"start_date"`
		EndDate             *time.Time `json:"end_date"`
		IsActive            *bool      `json:"is_active"`
		PowerCapacity       *float64   `json:"power_capacity"`
		CustomerPortal      string     `json:"customer_portal"`
		Notes               string     `json:"notes"`
		AllowsSelfReading   *bool      `json:"allows_self_reading"`
		ComparisonThreshold *float64   `json:"comparison_threshold"`
		ThresholdPerDay     *float64   `json:"threshold_per_day"`
		RecurringAmount     *float64   `json:"recurring_amount"`
		BillingInterval     *int       `json:"billing_interval"`
		BillingUnit         string     `json:"billing_unit" binding:"omitempty,oneof=day week month year"`
		PaidByMemberID        *uint      `json:"paid_by_member_id"`
		DefaultCategoryID     *uint      `json:"default_category_id"`
		DefaultBillTemplateID *uint      `json:"default_bill_template_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
		Preload("Readings", func(db *gorm.DB) *gorm.DB {
			return db.Order("reading_date DESC").Limit(3)
		}).
		First(&utility, utility.ID)

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

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	result := h.db.Where("id = ? AND property_id IN ?", id, memberPropertyIDs).
		Delete(&models.Utility{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete utility"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Utility deleted successfully"})
}

// AddReading adds a meter reading for a utility
func (h *UtilityHandler) AddReading(c *gin.Context) {
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
		ReadingDate time.Time `json:"reading_date" binding:"required"`
		ValueF1     *float64  `json:"value_f1"` // Electricity F1 (peak)
		ValueF2     *float64  `json:"value_f2"` // Electricity F2 (mid)
		ValueF3     *float64  `json:"value_f3"` // Electricity F3 (off-peak)
		Value       *float64  `json:"value"`    // Gas/Water single reading (mc/Smc)
		Source      string    `json:"source"`   // manual, submitted
		PhotoURL    string    `json:"photo_url"`
		Notes       string    `json:"notes"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	source := "manual"
	if input.Source != "" {
		source = input.Source
	}

	reading := models.MeterReading{
		UtilityID:   uint(utilityID),
		ReadingDate: input.ReadingDate,
		ValueF1:     input.ValueF1,
		ValueF2:     input.ValueF2,
		ValueF3:     input.ValueF3,
		Value:       input.Value,
		Source:      source,
		PhotoURL:    input.PhotoURL,
		Notes:       input.Notes,
	}

	if err := h.db.Create(&reading).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reading"})
		return
	}

	c.JSON(http.StatusCreated, reading)
}

// GetReadings returns all readings for a utility
func (h *UtilityHandler) GetReadings(c *gin.Context) {
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

	var readings []models.MeterReading
	if err := h.db.Where("utility_id = ?", utilityID).
		Order("reading_date DESC").
		Find(&readings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch readings"})
		return
	}

	// Build map: reading_id → bill_number for readings that are associated to a bill
	type billRef struct {
		UserReadingID uint
		BillNumber    string
	}
	var billRefs []billRef
	h.db.Model(&models.Bill{}).
		Select("user_reading_id, bill_number").
		Where("utility_id = ? AND user_reading_id IS NOT NULL AND deleted_at IS NULL", utilityID).
		Scan(&billRefs)

	billByReadingID := make(map[uint]string, len(billRefs))
	for _, br := range billRefs {
		billByReadingID[br.UserReadingID] = br.BillNumber
	}

	// Return readings enriched with associated_bill_number
	type ReadingResponse struct {
		models.MeterReading
		AssociatedBillNumber string `json:"associated_bill_number,omitempty"`
	}
	result := make([]ReadingResponse, len(readings))
	for i, r := range readings {
		result[i] = ReadingResponse{MeterReading: r, AssociatedBillNumber: billByReadingID[r.ID]}
	}

	c.JSON(http.StatusOK, result)
}

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
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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

	// Auto-create expense if bill is already marked as paid on creation
	if input.IsPaid {
		if err := h.autoCreateExpenseFromBill(c, userID, &bill); err != nil {
			log.Printf("⚠️  Failed to auto-create expense for bill %d: %v", bill.ID, err)
		}
	}

	c.JSON(http.StatusCreated, bill)
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
		Order("period_end DESC").
		Find(&bills).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bills"})
		return
	}

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

	// Check if utility's property is accessible
	found := false
	for _, pid := range memberPropertyIDs {
		if pid == bill.Utility.PropertyID {
			found = true
			break
		}
	}
	if !found {
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

	wasAlreadyPaid := bill.IsPaid

	if input.IsPaid != nil {
		bill.IsPaid = *input.IsPaid
	}
	if input.PaidDate != nil {
		bill.PaidDate = input.PaidDate
	}

	if err := h.db.Save(&bill).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bill"})
		return
	}

	// Auto-create expense when a bill is marked as paid for the first time
	if bill.IsPaid && !wasAlreadyPaid {
		if err := h.autoCreateExpenseFromBill(c, userID, &bill); err != nil {
			// Log but don't fail the request — bill payment already saved
			log.Printf("⚠️  Failed to auto-create expense for bill %d: %v", bill.ID, err)
		}
	}

	// Auto-delete linked expense when a bill is marked as unpaid
	if wasAlreadyPaid && !bill.IsPaid {
		bid := uint(billID)
		var linkedExpense models.Expense
		if err := h.db.Where("bill_id = ?", bid).First(&linkedExpense).Error; err == nil {
			h.db.Where("expense_id = ?", linkedExpense.ID).Delete(&models.ExpenseSplit{})
			h.db.Delete(&linkedExpense)
			log.Printf("🗑️  Auto-deleted expense ID=%d linked to bill ID=%d (bill marked unpaid)", linkedExpense.ID, bid)
		}
	}

	c.JSON(http.StatusOK, bill)
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

func (h *UtilityHandler) autoCreateExpenseFromBill(c *gin.Context, userID uint, bill *models.Bill) error {
	// Load utility
	var utility models.Utility
	if err := h.db.First(&utility, bill.UtilityID).Error; err != nil {
		return fmt.Errorf("could not load utility: %w", err)
	}

	// Find current user's member for the utility's property
	var member models.HouseholdMember
	if err := h.db.Where("property_id = ? AND user_id = ?", utility.PropertyID, userID).
		First(&member).Error; err != nil {
		return fmt.Errorf("user has no member profile for property %d: %w", utility.PropertyID, err)
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

	// Payment date: use bill's paid_date or today
	expenseDate := time.Now()
	if bill.PaidDate != nil {
		expenseDate = *bill.PaidDate
	}

	// Subcategory ID (optional)
	var subcatID *uint
	if utenzeSubcat.ID != 0 {
		id := utenzeSubcat.ID
		subcatID = &id
	}

	// Check household split mode
	var householdSettings models.HouseholdSettings
	splitMode := false
	if err := h.db.Where("property_id = ?", utility.PropertyID).First(&householdSettings).Error; err == nil {
		splitMode = householdSettings.SplitMode
	}

	// Determine split members from the payer's user settings (default_split_with_member_ids)
	var splitMemberIDs []uint
	isSplit := false
	if splitMode {
		var userSettings models.UserSettings
		if err := h.db.Where("user_id = ?", userID).First(&userSettings).Error; err == nil {
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

	billID := bill.ID
	propID := utility.PropertyID
	expense := models.Expense{
		UserID:         userID,
		PropertyID:     &propID,
		CategoryID:     casaCategory.ID,
		SubcategoryID:  subcatID,
		BillID:         &billID,
		Amount:         bill.AmountTotal,
		Date:           expenseDate,
		Description:    description,
		PaidByMemberID: member.ID,
		IsSplit:        isSplit,
	}

	if err := h.db.Create(&expense).Error; err != nil {
		return fmt.Errorf("could not create expense: %w", err)
	}

	// Create split records: payer + default split members, all is_settled=false
	if isSplit {
		// Build full member list: payer + split-with members (dedup)
		allMemberIDs := make([]uint, 0, len(splitMemberIDs)+1)
		allMemberIDs = append(allMemberIDs, member.ID)
		for _, mid := range splitMemberIDs {
			if mid != member.ID {
				allMemberIDs = append(allMemberIDs, mid)
			}
		}
		splitAmount := bill.AmountTotal / float64(len(allMemberIDs))
		for _, mid := range allMemberIDs {
			split := models.ExpenseSplit{
				ExpenseID: expense.ID,
				MemberID:  mid,
				Amount:    splitAmount,
				IsSettled: false,
			}
			h.db.Create(&split)
		}
		log.Printf("✅ Auto-created expense ID=%d '%s' €%.2f for bill ID=%d (split among %d members)", expense.ID, description, expense.Amount, bill.ID, len(allMemberIDs))
	} else {
		log.Printf("✅ Auto-created expense ID=%d '%s' €%.2f for bill ID=%d (no split)", expense.ID, description, expense.Amount, bill.ID)
	}
	return nil
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

	// Check if utility's property is accessible
	found := false
	for _, pid := range memberPropertyIDs {
		if pid == bill.Utility.PropertyID {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this bill"})
		return
	}

	if err := h.db.Delete(&bill).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bill"})
		return
	}

	// Delete auto-created expense linked to this bill (if any)
	bid := uint(billID)
	var linkedExpense models.Expense
	if err := h.db.Where("bill_id = ?", bid).First(&linkedExpense).Error; err == nil {
		// Delete splits first
		h.db.Where("expense_id = ?", linkedExpense.ID).Delete(&models.ExpenseSplit{})
		// Delete expense
		h.db.Delete(&linkedExpense)
		log.Printf("🗑️  Deleted auto-expense ID=%d linked to bill ID=%d", linkedExpense.ID, billID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bill deleted successfully"})
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

	// Check if utility's property is accessible
	found := false
	for _, pid := range memberPropertyIDs {
		if pid == bill.Utility.PropertyID {
			found = true
			break
		}
	}
	if !found {
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

	// Update all fields
	bill.BillNumber = input.BillNumber
	bill.UserReadingID = input.UserReadingID
	bill.IssueDate = input.IssueDate
	bill.PeriodStart = input.PeriodStart
	bill.PeriodEnd = input.PeriodEnd
	bill.DueDate = input.DueDate
	bill.ConsumptionTotal = input.ConsumptionTotal
	bill.AmountTotal = input.AmountTotal
	bill.ConversionCoefficient = input.ConversionCoefficient
	bill.EstimatedDate = input.EstimatedDate
	bill.EstimatedReading = input.EstimatedReading
	bill.EstimatedConsumption = input.EstimatedConsumption
	bill.ReadingType = input.ReadingType
	bill.IsPaid = input.IsPaid
	bill.PaidDate = input.PaidDate
	bill.ProviderReadingDate = input.ProviderReadingDate
	bill.ProviderReadingF1 = input.ProviderReadingF1
	bill.ProviderReadingF2 = input.ProviderReadingF2
	bill.ProviderReadingF3 = input.ProviderReadingF3
	bill.ProviderReading = input.ProviderReading

	if err := h.db.Save(&bill).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bill"})
		return
	}

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

// UpdateReading updates a meter reading
func (h *UtilityHandler) UpdateReading(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	readingID, err := strconv.ParseUint(c.Param("readingId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reading ID"})
		return
	}

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Find the reading
	var reading models.MeterReading
	if err := h.db.First(&reading, readingID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reading not found"})
		return
	}

	// Verify access through utility
	var utility models.Utility
	if err := h.db.First(&utility, reading.UtilityID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	found := false
	for _, pid := range memberPropertyIDs {
		if pid == utility.PropertyID {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this reading"})
		return
	}

	var input struct {
		ReadingDate time.Time `json:"reading_date"`
		ValueF1     *float64  `json:"value_f1"`
		ValueF2     *float64  `json:"value_f2"`
		ValueF3     *float64  `json:"value_f3"`
		Value       *float64  `json:"value"`
		Source      string    `json:"source"`
		Notes       string    `json:"notes"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reading.ReadingDate = input.ReadingDate
	reading.ValueF1 = input.ValueF1
	reading.ValueF2 = input.ValueF2
	reading.ValueF3 = input.ValueF3
	reading.Value = input.Value
	if input.Source != "" {
		reading.Source = input.Source
	}
	reading.Notes = input.Notes

	if err := h.db.Save(&reading).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reading"})
		return
	}

	c.JSON(http.StatusOK, reading)
}

// DeleteReading removes a meter reading
func (h *UtilityHandler) DeleteReading(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	readingID, err := strconv.ParseUint(c.Param("readingId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reading ID"})
		return
	}

	// Get property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Find the reading
	var reading models.MeterReading
	if err := h.db.First(&reading, readingID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reading not found"})
		return
	}

	// Verify access through utility
	var utility models.Utility
	if err := h.db.First(&utility, reading.UtilityID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	found := false
	for _, pid := range memberPropertyIDs {
		if pid == utility.PropertyID {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this reading"})
		return
	}

	if err := h.db.Delete(&reading).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reading"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reading deleted successfully"})
}

// ReadingComparison represents the comparison between provider and user readings
type ReadingComparison struct {
	BillID              uint       `json:"bill_id"`
	BillNumber          string     `json:"bill_number"`
	PeriodEnd           time.Time  `json:"period_end"`
	UtilityType         string     `json:"utility_type"`
	ReadingType         string     `json:"reading_type"` // actual or estimated
	ProviderReadingDate *time.Time `json:"provider_reading_date"`
	UserReadingDate     *time.Time `json:"user_reading_date"`
	DaysDifference      int        `json:"days_difference"`     // Days between user and provider readings
	EffectiveThreshold  float64    `json:"effective_threshold"` // Threshold adjusted for days difference
	// For electricity (F1/F2/F3)
	ProviderF1   *float64 `json:"provider_f1,omitempty"`
	ProviderF2   *float64 `json:"provider_f2,omitempty"`
	ProviderF3   *float64 `json:"provider_f3,omitempty"`
	UserF1       *float64 `json:"user_f1,omitempty"`
	UserF2       *float64 `json:"user_f2,omitempty"`
	UserF3       *float64 `json:"user_f3,omitempty"`
	DifferenceF1 *float64 `json:"difference_f1,omitempty"` // Absolute difference in kWh
	DifferenceF2 *float64 `json:"difference_f2,omitempty"`
	DifferenceF3 *float64 `json:"difference_f3,omitempty"`
	// For gas/water (single value)
	ProviderReading *float64 `json:"provider_reading,omitempty"`
	UserReading     *float64 `json:"user_reading,omitempty"`
	Difference      *float64 `json:"difference,omitempty"` // Absolute difference in mc/Smc
	// Status
	Status       string `json:"status"`        // ok, warning, alert, no_data
	AlertMessage string `json:"alert_message"` // Human readable message
}

// ConsumptionPeriod represents consumption for a period comparing user vs provider readings
type ConsumptionPeriod struct {
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	BillID      *uint     `json:"bill_id,omitempty"`
	// User consumption (from self-readings difference)
	UserConsumptionF1 *float64 `json:"user_consumption_f1,omitempty"`
	UserConsumptionF2 *float64 `json:"user_consumption_f2,omitempty"`
	UserConsumptionF3 *float64 `json:"user_consumption_f3,omitempty"`
	UserConsumption   *float64 `json:"user_consumption,omitempty"` // For gas/water
	// Provider consumption (from billed readings difference)
	ProviderConsumptionF1 *float64 `json:"provider_consumption_f1,omitempty"`
	ProviderConsumptionF2 *float64 `json:"provider_consumption_f2,omitempty"`
	ProviderConsumptionF3 *float64 `json:"provider_consumption_f3,omitempty"`
	ProviderConsumption   *float64 `json:"provider_consumption,omitempty"` // For gas/water
	// Differences
	DifferenceF1 *float64 `json:"difference_f1,omitempty"`
	DifferenceF2 *float64 `json:"difference_f2,omitempty"`
	DifferenceF3 *float64 `json:"difference_f3,omitempty"`
	Difference   *float64 `json:"difference,omitempty"`
}

// ConsumptionSummary contains cumulative consumption analysis
type ConsumptionSummary struct {
	// Cumulative totals from user self-readings
	TotalUserF1 float64 `json:"total_user_f1"`
	TotalUserF2 float64 `json:"total_user_f2"`
	TotalUserF3 float64 `json:"total_user_f3"`
	TotalUser   float64 `json:"total_user"` // For gas/water or sum of F1+F2+F3
	// Cumulative totals from provider readings (billed)
	TotalProviderF1 float64 `json:"total_provider_f1"`
	TotalProviderF2 float64 `json:"total_provider_f2"`
	TotalProviderF3 float64 `json:"total_provider_f3"`
	TotalProvider   float64 `json:"total_provider"` // For gas/water or sum of F1+F2+F3
	// Cumulative differences (positive = provider charged more than actual consumption)
	CumulativeDifferenceF1 float64 `json:"cumulative_difference_f1"`
	CumulativeDifferenceF2 float64 `json:"cumulative_difference_f2"`
	CumulativeDifferenceF3 float64 `json:"cumulative_difference_f3"`
	CumulativeDifference   float64 `json:"cumulative_difference"` // Total difference
	// Period covered
	FirstPeriod time.Time `json:"first_period"`
	LastPeriod  time.Time `json:"last_period"`
	// Alert if cumulative difference is significant
	HasCumulativeAlert   bool   `json:"has_cumulative_alert"`
	CumulativeAlertLevel string `json:"cumulative_alert_level,omitempty"` // warning, alert
	CumulativeMessage    string `json:"cumulative_message,omitempty"`
}

// CompareReadings compares provider readings from bills with user's manual readings
// GET /api/v1/utilities/:id/compare-readings
func (h *UtilityHandler) CompareReadings(c *gin.Context) {
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

	// Get all bills with provider readings, preloading the explicit user reading association
	var bills []models.Bill
	h.db.Where("utility_id = ?", utilityID).
		Preload("UserReading").
		Order("period_end DESC").
		Find(&bills)

	log.Printf("CompareReadings: Found %d bills for utility %d (type: %s)", len(bills), utilityID, utility.Type)

	// Get all user readings
	var readings []models.MeterReading
	h.db.Where("utility_id = ?", utilityID).
		Order("reading_date DESC").
		Find(&readings)

	log.Printf("CompareReadings: Found %d user readings", len(readings))

	// Use utility's comparison threshold (base threshold for same-day readings)
	baseThreshold := utility.ComparisonThreshold
	if baseThreshold == 0 {
		baseThreshold = 2.0
	}
	// Per-day tolerance
	thresholdPerDay := utility.ThresholdPerDay
	if thresholdPerDay == 0 {
		thresholdPerDay = 1.0
	}
	// Allow override via query params
	if t := c.Query("threshold"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil {
			baseThreshold = parsed
		}
	}
	if t := c.Query("threshold_per_day"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil {
			thresholdPerDay = parsed
		}
	}

	var comparisons []ReadingComparison

	for _, bill := range bills {
		// Skip bills without provider readings
		hasProviderReading := false
		if utility.Type == "electricity" {
			hasProviderReading = bill.ProviderReadingF1 != nil || bill.ProviderReadingF2 != nil || bill.ProviderReadingF3 != nil
			log.Printf("Bill %d: F1=%v, F2=%v, F3=%v, hasReading=%v",
				bill.ID, bill.ProviderReadingF1, bill.ProviderReadingF2, bill.ProviderReadingF3, hasProviderReading)
		} else {
			hasProviderReading = bill.ProviderReading != nil
			log.Printf("Bill %d: ProviderReading=%v, hasReading=%v",
				bill.ID, bill.ProviderReading, hasProviderReading)
		}

		if !hasProviderReading {
			log.Printf("Skipping bill %d - no provider readings", bill.ID)
			continue
		}

		comparison := ReadingComparison{
			BillID:              bill.ID,
			BillNumber:          bill.BillNumber,
			PeriodEnd:           bill.PeriodEnd,
			UtilityType:         utility.Type,
			ReadingType:         bill.ReadingType,
			ProviderReadingDate: bill.ProviderReadingDate,
			Status:              "ok",
			EffectiveThreshold:  baseThreshold, // Will be adjusted based on days difference
		}

		// Find user reading that falls within the bill period
		// The user's autolettura should be within or close to the billing period
		var closestReading *models.MeterReading
		var bestScore int = -1 // Higher is better

		log.Printf("Bill %d: period %v - %v", bill.ID, bill.PeriodStart.Format("2006-01-02"), bill.PeriodEnd.Format("2006-01-02"))

		for i := range readings {
			readingDate := readings[i].ReadingDate
			score := 0

			// Best match: reading date is within the bill period
			if !readingDate.Before(bill.PeriodStart) && !readingDate.After(bill.PeriodEnd) {
				score = 100
			} else {
				// Also consider readings slightly before period start or after period end
				// (user might read meter a few days early/late)
				daysBefore := bill.PeriodStart.Sub(readingDate).Hours() / 24
				daysAfter := readingDate.Sub(bill.PeriodEnd).Hours() / 24

				if daysBefore > 0 && daysBefore <= 15 {
					score = 50 - int(daysBefore) // Within 15 days before period start
				} else if daysAfter > 0 && daysAfter <= 15 {
					score = 50 - int(daysAfter) // Within 15 days after period end
				}
			}

			log.Printf("  Reading %d: date=%v, score=%d (F1=%v, F2=%v, F3=%v)",
				readings[i].ID, readingDate.Format("2006-01-02"), score,
				readings[i].ValueF1, readings[i].ValueF2, readings[i].ValueF3)

			if score > bestScore {
				bestScore = score
				closestReading = &readings[i]
			}
		}

		if closestReading != nil {
			log.Printf("Best match: Reading %d with score %d", closestReading.ID, bestScore)
		} else {
			log.Printf("No matching reading found for bill %d", bill.ID)
		}

		if closestReading != nil {
			comparison.UserReadingDate = &closestReading.ReadingDate

			// Calculate days difference between user reading and provider reading
			providerDate := bill.PeriodEnd // Default to period end if no specific date
			if bill.ProviderReadingDate != nil {
				providerDate = *bill.ProviderReadingDate
			}
			daysDiff := int(providerDate.Sub(closestReading.ReadingDate).Hours() / 24)
			if daysDiff < 0 {
				daysDiff = -daysDiff
			}
			comparison.DaysDifference = daysDiff

			// Calculate effective threshold: base + (days * perDay)
			comparison.EffectiveThreshold = baseThreshold + float64(daysDiff)*thresholdPerDay
			log.Printf("Days difference: %d, Effective threshold: %.2f (base: %.2f + %d * %.2f)",
				daysDiff, comparison.EffectiveThreshold, baseThreshold, daysDiff, thresholdPerDay)
		}

		// Compare based on utility type
		switch utility.Type {
		case "electricity":
			comparison.ProviderF1 = bill.ProviderReadingF1
			comparison.ProviderF2 = bill.ProviderReadingF2
			comparison.ProviderF3 = bill.ProviderReadingF3

			if closestReading != nil {
				comparison.UserF1 = closestReading.ValueF1
				comparison.UserF2 = closestReading.ValueF2
				comparison.UserF3 = closestReading.ValueF3

				// Calculate absolute differences for each band
				maxAbsDiff := 0.0

				if bill.ProviderReadingF1 != nil && closestReading.ValueF1 != nil {
					diff := *bill.ProviderReadingF1 - *closestReading.ValueF1
					comparison.DifferenceF1 = &diff
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}
					if absDiff > maxAbsDiff {
						maxAbsDiff = absDiff
					}
				}

				if bill.ProviderReadingF2 != nil && closestReading.ValueF2 != nil {
					diff := *bill.ProviderReadingF2 - *closestReading.ValueF2
					comparison.DifferenceF2 = &diff
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}
					if absDiff > maxAbsDiff {
						maxAbsDiff = absDiff
					}
				}

				if bill.ProviderReadingF3 != nil && closestReading.ValueF3 != nil {
					diff := *bill.ProviderReadingF3 - *closestReading.ValueF3
					comparison.DifferenceF3 = &diff
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}
					if absDiff > maxAbsDiff {
						maxAbsDiff = absDiff
					}
				}

				// Determine status based on absolute difference using effective threshold
				effectiveThreshold := comparison.EffectiveThreshold
				if maxAbsDiff > effectiveThreshold*2 {
					comparison.Status = "alert"
					comparison.AlertMessage = fmt.Sprintf("Discrepanza di %.1f kWh (soglia effettiva: %.1f kWh per %d giorni di differenza)",
						maxAbsDiff, effectiveThreshold, comparison.DaysDifference)
				} else if maxAbsDiff > effectiveThreshold {
					comparison.Status = "warning"
					comparison.AlertMessage = fmt.Sprintf("Differenza di %.1f kWh (soglia effettiva: %.1f kWh per %d giorni di differenza)",
						maxAbsDiff, effectiveThreshold, comparison.DaysDifference)
				}
			} else {
				comparison.Status = "no_data"
				comparison.AlertMessage = "Nessuna autolettura disponibile per il confronto"
			}

		case "gas", "water":
			comparison.ProviderReading = bill.ProviderReading

			if closestReading != nil && closestReading.Value != nil {
				comparison.UserReading = closestReading.Value

				if bill.ProviderReading != nil {
					diff := *bill.ProviderReading - *closestReading.Value
					comparison.Difference = &diff

					// Determine status based on absolute difference using effective threshold
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}

					unit := "mc"
					if utility.Type == "gas" {
						unit = "Smc"
					}

					effectiveThreshold := comparison.EffectiveThreshold
					if absDiff > effectiveThreshold*2 {
						comparison.Status = "alert"
						comparison.AlertMessage = fmt.Sprintf("Discrepanza di %.1f %s (soglia effettiva: %.1f %s per %d giorni di differenza)",
							absDiff, unit, effectiveThreshold, unit, comparison.DaysDifference)
					} else if absDiff > effectiveThreshold {
						comparison.Status = "warning"
						comparison.AlertMessage = fmt.Sprintf("Differenza di %.1f %s (soglia effettiva: %.1f %s per %d giorni di differenza)",
							absDiff, unit, effectiveThreshold, unit, comparison.DaysDifference)
					}
				}
			} else {
				comparison.Status = "no_data"
				comparison.AlertMessage = "Nessuna autolettura disponibile per il confronto"
			}
		}

		comparisons = append(comparisons, comparison)
	}

	// Calculate consumption analysis (differences between consecutive readings)
	consumptionPeriods, consumptionSummary := h.calculateConsumptionAnalysis(utility.Type, bills, readings, baseThreshold)

	c.JSON(http.StatusOK, gin.H{
		"comparisons":         comparisons,
		"base_threshold":      baseThreshold,
		"threshold_per_day":   thresholdPerDay,
		"utility_type":        utility.Type,
		"consumption_periods": consumptionPeriods,
		"consumption_summary": consumptionSummary,
	})
}

// calculateConsumptionAnalysis computes period-by-period consumption comparison.
//
// Algorithm (deterministic):
//   - Sort bills by period_end ascending.
//   - For each bill define its “effective user reading value”:
//     • If bill.UserReading != nil → use the explicitly associated self-reading value.
//     • Otherwise              → fall back to bill.ProviderReading (so Δeffettivo = Δfatturato).
//   - Consumo Fatturato[i]  = providerReading[i]  − providerReading[i-1]  (consecutive absolute bill readings)
//   - Consumo Effettivo[i]  = effectiveReading[i] − effectiveReading[i-1] (consecutive effective readings)
//   - Period rows start from index 1 (bill[0] is the reference anchor; no row generated for it).
//   - Guard: need at least 2 bills with provider_reading set.
func (h *UtilityHandler) calculateConsumptionAnalysis(utilityType string, bills []models.Bill, readings []models.MeterReading, threshold float64) ([]ConsumptionPeriod, *ConsumptionSummary) {
	if len(bills) < 2 {
		return nil, nil
	}

	// Sort bills by period_end ascending
	sortedBills := make([]models.Bill, len(bills))
	copy(sortedBills, bills)
	for i := 0; i < len(sortedBills)-1; i++ {
		for j := i + 1; j < len(sortedBills); j++ {
			if sortedBills[i].PeriodEnd.After(sortedBills[j].PeriodEnd) {
				sortedBills[i], sortedBills[j] = sortedBills[j], sortedBills[i]
			}
		}
	}

	// effectiveValue returns the user reading value for a bill.
	// Uses the explicitly associated UserReading when available;
	// falls back to ProviderReading so the period contribution is neutral (Δ=0).
	effectiveValue := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.Value != nil {
			return b.UserReading.Value
		}
		return b.ProviderReading
	}
	effectiveValueF1 := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.ValueF1 != nil {
			return b.UserReading.ValueF1
		}
		return b.ProviderReadingF1
	}
	effectiveValueF2 := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.ValueF2 != nil {
			return b.UserReading.ValueF2
		}
		return b.ProviderReadingF2
	}
	effectiveValueF3 := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.ValueF3 != nil {
			return b.UserReading.ValueF3
		}
		return b.ProviderReadingF3
	}

	var periods []ConsumptionPeriod
	summary := &ConsumptionSummary{}

	for i := 1; i < len(sortedBills); i++ {
		bill := sortedBills[i]
		prev := sortedBills[i-1]
		billID := bill.ID
		period := ConsumptionPeriod{
			PeriodStart: bill.PeriodStart,
			PeriodEnd:   bill.PeriodEnd,
			BillID:      &billID,
		}
		if utilityType == "electricity" {
			// --- Provider consumption (consecutive absolute F1/F2/F3 readings) ---
			provTotal := 0.0
			hasProvF1 := bill.ProviderReadingF1 != nil && prev.ProviderReadingF1 != nil
			hasProvF2 := bill.ProviderReadingF2 != nil && prev.ProviderReadingF2 != nil
			hasProvF3 := bill.ProviderReadingF3 != nil && prev.ProviderReadingF3 != nil
			if hasProvF1 {
				f1 := *bill.ProviderReadingF1 - *prev.ProviderReadingF1
				period.ProviderConsumptionF1 = &f1
				summary.TotalProviderF1 += f1
				provTotal += f1
			}
			if hasProvF2 {
				f2 := *bill.ProviderReadingF2 - *prev.ProviderReadingF2
				period.ProviderConsumptionF2 = &f2
				summary.TotalProviderF2 += f2
				provTotal += f2
			}
			if hasProvF3 {
				f3 := *bill.ProviderReadingF3 - *prev.ProviderReadingF3
				period.ProviderConsumptionF3 = &f3
				summary.TotalProviderF3 += f3
				provTotal += f3
			}
			if hasProvF1 || hasProvF2 || hasProvF3 {
				period.ProviderConsumption = &provTotal
				summary.TotalProvider += provTotal
			}

			// --- User consumption (consecutive effective F1/F2/F3 values) ---
			ef1Cur, ef1Prv := effectiveValueF1(bill), effectiveValueF1(prev)
			ef2Cur, ef2Prv := effectiveValueF2(bill), effectiveValueF2(prev)
			ef3Cur, ef3Prv := effectiveValueF3(bill), effectiveValueF3(prev)
			userTotal := 0.0
			hasUser := false
			if ef1Cur != nil && ef1Prv != nil {
				f1 := *ef1Cur - *ef1Prv
				period.UserConsumptionF1 = &f1
				summary.TotalUserF1 += f1
				userTotal += f1
				hasUser = true
			}
			if ef2Cur != nil && ef2Prv != nil {
				f2 := *ef2Cur - *ef2Prv
				period.UserConsumptionF2 = &f2
				summary.TotalUserF2 += f2
				userTotal += f2
				hasUser = true
			}
			if ef3Cur != nil && ef3Prv != nil {
				f3 := *ef3Cur - *ef3Prv
				period.UserConsumptionF3 = &f3
				summary.TotalUserF3 += f3
				userTotal += f3
				hasUser = true
			}
			if hasUser {
				period.UserConsumption = &userTotal
				summary.TotalUser += userTotal
			}

			// Per-band differences
			if period.ProviderConsumptionF1 != nil && period.UserConsumptionF1 != nil {
				d := *period.ProviderConsumptionF1 - *period.UserConsumptionF1
				period.DifferenceF1 = &d
			}
			if period.ProviderConsumptionF2 != nil && period.UserConsumptionF2 != nil {
				d := *period.ProviderConsumptionF2 - *period.UserConsumptionF2
				period.DifferenceF2 = &d
			}
			if period.ProviderConsumptionF3 != nil && period.UserConsumptionF3 != nil {
				d := *period.ProviderConsumptionF3 - *period.UserConsumptionF3
				period.DifferenceF3 = &d
			}

		} else {
			// --- Gas / Water ---
			// Provider: consecutive absolute ProviderReading
			if bill.ProviderReading != nil && prev.ProviderReading != nil {
				provTotal := *bill.ProviderReading - *prev.ProviderReading
				period.ProviderConsumption = &provTotal
				summary.TotalProvider += provTotal
			}

			// User: consecutive effective reading values
			efCur, efPrv := effectiveValue(bill), effectiveValue(prev)
			if efCur != nil && efPrv != nil {
				userTotal := *efCur - *efPrv
				period.UserConsumption = &userTotal
				summary.TotalUser += userTotal
			}
		}

		// Overall difference for this period
		if period.ProviderConsumption != nil && period.UserConsumption != nil {
			diff := *period.ProviderConsumption - *period.UserConsumption
			period.Difference = &diff
		}

		periods = append(periods, period)
	}
	summary.CumulativeDifferenceF1 = summary.TotalProviderF1 - summary.TotalUserF1
	summary.CumulativeDifferenceF2 = summary.TotalProviderF2 - summary.TotalUserF2
	summary.CumulativeDifferenceF3 = summary.TotalProviderF3 - summary.TotalUserF3
	summary.CumulativeDifference = summary.TotalProvider - summary.TotalUser

	if len(sortedBills) > 0 {
		summary.FirstPeriod = sortedBills[0].PeriodStart
		summary.LastPeriod = sortedBills[len(sortedBills)-1].PeriodEnd
	}

	numPeriods := len(periods)
	if numPeriods > 0 {
		cumulativeThreshold := threshold * float64(numPeriods)
		if summary.CumulativeDifference > cumulativeThreshold*2 {
			summary.HasCumulativeAlert = true
			summary.CumulativeAlertLevel = "alert"
			summary.CumulativeMessage = fmt.Sprintf("ATTENZIONE: il fornitore ha fatturato %.1f unità IN PIÙ rispetto ai consumi effettivi rilevati dalle autoletture in %d periodi. Stai pagando più del dovuto!",
				summary.CumulativeDifference, numPeriods)
		} else if summary.CumulativeDifference > cumulativeThreshold {
			summary.HasCumulativeAlert = true
			summary.CumulativeAlertLevel = "warning"
			summary.CumulativeMessage = fmt.Sprintf("Il fornitore ha fatturato %.1f unità in più rispetto ai consumi effettivi in %d periodi. Tieni sotto controllo questa differenza.",
				summary.CumulativeDifference, numPeriods)
		} else if summary.CumulativeDifference < -cumulativeThreshold {
			summary.CumulativeMessage = fmt.Sprintf("Il fornitore ha fatturato %.1f unità in meno rispetto ai consumi rilevati in %d periodi. Potrebbe esserci un conguaglio a fine anno.",
				-summary.CumulativeDifference, numPeriods)
		}
	}

	return periods, summary
}

// GetCommunications returns all communications for a utility
func (h *UtilityHandler) GetCommunications(c *gin.Context) {
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

	// Verify access
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

	var comms []models.ServiceCommunication
	h.db.Where("utility_id = ?", utilityID).Order("created_at DESC").Find(&comms)

	c.JSON(http.StatusOK, comms)
}

// AddCommunication creates a new communication for a utility
func (h *UtilityHandler) AddCommunication(c *gin.Context) {
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

	// Verify access
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
		BillID         *uint      `json:"bill_id"`
		Type           string     `json:"type" binding:"required,oneof=price_change contract_modification info privacy"`
		Title          string     `json:"title" binding:"required"`
		Content        string     `json:"content"`
		ActionDeadline *time.Time `json:"action_deadline"`
		IsImportant    bool       `json:"is_important"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comm := models.ServiceCommunication{
		UtilityID:      uint(utilityID),
		BillID:         input.BillID,
		Type:           input.Type,
		Title:          input.Title,
		Content:        input.Content,
		ActionDeadline: input.ActionDeadline,
		IsImportant:    input.IsImportant,
	}

	if err := h.db.Create(&comm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create communication"})
		return
	}

	c.JSON(http.StatusCreated, comm)
}

// MarkCommunicationRead marks a communication as read
func (h *UtilityHandler) MarkCommunicationRead(c *gin.Context) {
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

	commID, err := strconv.ParseUint(c.Param("commId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid communication ID"})
		return
	}

	// Find the communication and verify it belongs to the specified utility
	var comm models.ServiceCommunication
	if err := h.db.First(&comm, commID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Communication not found"})
		return
	}

	if comm.UtilityID != uint(utilityID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Communication does not belong to this utility"})
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", comm.UtilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	comm.IsRead = true
	h.db.Save(&comm)

	c.JSON(http.StatusOK, comm)
}

// DeleteCommunication deletes a communication
func (h *UtilityHandler) DeleteCommunication(c *gin.Context) {
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

	commID, err := strconv.ParseUint(c.Param("commId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid communication ID"})
		return
	}

	var comm models.ServiceCommunication
	if err := h.db.First(&comm, commID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Communication not found"})
		return
	}

	if comm.UtilityID != uint(utilityID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Communication does not belong to this utility"})
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", comm.UtilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	h.db.Unscoped().Delete(&comm)
	c.JSON(http.StatusOK, gin.H{"message": "Communication deleted"})
}

// DeleteReadCommunications bulk-deletes all read communications for the user
func (h *UtilityHandler) DeleteReadCommunications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utilityIDs []uint
	h.db.Model(&models.Utility{}).
		Where("property_id IN ?", memberPropertyIDs).
		Pluck("id", &utilityIDs)

	result := h.db.Unscoped().
		Where("utility_id IN ? AND is_read = ?", utilityIDs, true).
		Delete(&models.ServiceCommunication{})

	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

// GetAllCommunications returns all communications across user's utilities
func (h *UtilityHandler) GetAllCommunications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utilityIDs []uint
	h.db.Model(&models.Utility{}).
		Where("property_id IN ?", memberPropertyIDs).
		Pluck("id", &utilityIDs)

	// Auto-cleanup: delete communications older than retention days
	var settings models.UserSettings
	if err := h.db.Where("user_id = ?", userID).First(&settings).Error; err == nil {
		retentionDays := settings.NotificationRetentionDays
		if retentionDays == 0 {
			retentionDays = 90
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		h.db.Unscoped().
			Where("utility_id IN ? AND created_at < ?", utilityIDs, cutoff).
			Delete(&models.ServiceCommunication{})
	}

	query := h.db.Where("utility_id IN ?", utilityIDs).
		Preload("Utility").
		Order("created_at DESC")

	// Filter by unread only
	if c.Query("unread_only") == "true" {
		query = query.Where("is_read = ?", false)
	}

	// Filter by read only
	if c.Query("read_only") == "true" {
		query = query.Where("is_read = ?", true)
	}

	// Limit
	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	query = query.Limit(limit)

	var comms []models.ServiceCommunication
	query.Find(&comms)

	c.JSON(http.StatusOK, comms)
}

// GetUnreadCount returns the count of unread communications
func (h *UtilityHandler) GetUnreadCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utilityIDs []uint
	h.db.Model(&models.Utility{}).
		Where("property_id IN ?", memberPropertyIDs).
		Pluck("id", &utilityIDs)

	var count int64
	h.db.Model(&models.ServiceCommunication{}).
		Where("utility_id IN ? AND is_read = ?", utilityIDs, false).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"count": count})
}
