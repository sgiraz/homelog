package handlers

import (
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
		Type                string     `json:"type" binding:"required,oneof=electricity gas water waste"`
		Provider            string     `json:"provider" binding:"required"`
		CustomerCode        string     `json:"customer_code"`
		ServiceCode         string     `json:"service_code"`
		Address             string     `json:"address"`
		StartDate           time.Time  `json:"start_date"`
		EndDate             *time.Time `json:"end_date"`
		IsActive            bool       `json:"is_active"`
		PowerCapacity       float64    `json:"power_capacity"`
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

	// Check if there's already an active utility of the same type for this property
	var existingCount int64
	h.db.Model(&models.Utility{}).
		Where("property_id = ? AND type = ? AND is_active = ?", input.PropertyID, input.Type, true).
		Count(&existingCount)

	if existingCount > 0 {
		typeLabels := map[string]string{
			"electricity": "Luce",
			"gas":         "Gas",
			"water":       "Acqua",
			"waste":       "Rifiuti",
		}
		label := typeLabels[input.Type]
		if label == "" {
			label = input.Type
		}
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Esiste già un'utenza %s attiva per questa proprietà. Disattiva quella esistente prima di crearne una nuova.", label),
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
		PowerCapacity:       input.PowerCapacity,
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
		Preload("Bills", func(db *gorm.DB) *gorm.DB {
			return db.Order("period_end DESC")
		}).
		Preload("Readings", func(db *gorm.DB) *gorm.DB {
			return db.Order("reading_date DESC")
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

	if err := h.db.Save(&utility).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update utility"})
		return
	}

	// Reload with relations
	h.db.Preload("Property").
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

	c.JSON(http.StatusOK, readings)
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
		BillNumber          string     `json:"bill_number"`
		IssueDate           time.Time  `json:"issue_date" binding:"required"`
		PeriodStart         time.Time  `json:"period_start" binding:"required"`
		PeriodEnd           time.Time  `json:"period_end" binding:"required"`
		DueDate             time.Time  `json:"due_date" binding:"required"`
		ReadingStartDate    *time.Time `json:"reading_start_date"`
		ReadingStartValue   *float64   `json:"reading_start_value"`
		ReadingEndDate      *time.Time `json:"reading_end_date"`
		ReadingEndValue     *float64   `json:"reading_end_value"`
		ReadingType         string     `json:"reading_type"`
		ProviderReadingDate *time.Time `json:"provider_reading_date"` // Data lettura fornitore
		ProviderReadingF1   *float64   `json:"provider_reading_f1"`   // Lettura F1 (electricity)
		ProviderReadingF2   *float64   `json:"provider_reading_f2"`   // Lettura F2 (electricity)
		ProviderReadingF3   *float64   `json:"provider_reading_f3"`   // Lettura F3 (electricity)
		ProviderReading     *float64   `json:"provider_reading"`      // Lettura singola (gas/water)
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
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bill := models.Bill{
		UtilityID:             uint(utilityID),
		BillNumber:            input.BillNumber,
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

	// Load utility relation
	h.db.Preload("Utility").First(&bill, bill.ID)

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
		BillNumber            string     `json:"bill_number"`
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
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update all fields
	bill.BillNumber = input.BillNumber
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

	// Get all bills with provider readings
	var bills []models.Bill
	h.db.Where("utility_id = ?", utilityID).
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

// calculateConsumptionAnalysis calculates consumption differences between consecutive readings
func (h *UtilityHandler) calculateConsumptionAnalysis(utilityType string, bills []models.Bill, readings []models.MeterReading, threshold float64) ([]ConsumptionPeriod, *ConsumptionSummary) {
	// Need at least 1 bill for provider data, and at least 2 readings for user consumption calculation
	if len(bills) == 0 || len(readings) < 2 {
		return nil, nil
	}

	var periods []ConsumptionPeriod
	summary := &ConsumptionSummary{}

	// Sort bills by period_end ascending for chronological processing
	sortedBills := make([]models.Bill, len(bills))
	copy(sortedBills, bills)
	for i := 0; i < len(sortedBills)-1; i++ {
		for j := i + 1; j < len(sortedBills); j++ {
			if sortedBills[i].PeriodEnd.After(sortedBills[j].PeriodEnd) {
				sortedBills[i], sortedBills[j] = sortedBills[j], sortedBills[i]
			}
		}
	}

	// Sort readings by date ascending
	sortedReadings := make([]models.MeterReading, len(readings))
	copy(sortedReadings, readings)
	for i := 0; i < len(sortedReadings)-1; i++ {
		for j := i + 1; j < len(sortedReadings); j++ {
			if sortedReadings[i].ReadingDate.After(sortedReadings[j].ReadingDate) {
				sortedReadings[i], sortedReadings[j] = sortedReadings[j], sortedReadings[i]
			}
		}
	}

	// Calculate provider consumption from consumption_total on each bill
	// This is the actual billed consumption, which includes the first bill
	providerConsumptions := make(map[time.Time]struct {
		F1, F2, F3, Total float64
		BillID            uint
	})

	for _, bill := range sortedBills {
		consumption := struct {
			F1, F2, F3, Total float64
			BillID            uint
		}{BillID: bill.ID}

		// Use consumption_total from the bill (what provider actually billed)
		consumption.Total = bill.ConsumptionTotal
		summary.TotalProvider += consumption.Total

		// For electricity, also track per-band consumption if we have consecutive readings
		// This is for display purposes in the detail breakdown
		if utilityType == "electricity" {
			// Find previous bill to calculate per-band consumption
			var prevBill *models.Bill
			for j := range sortedBills {
				if sortedBills[j].ID == bill.ID && j > 0 {
					prevBill = &sortedBills[j-1]
					break
				}
			}

			if prevBill != nil {
				if prevBill.ProviderReadingF1 != nil && bill.ProviderReadingF1 != nil {
					consumption.F1 = *bill.ProviderReadingF1 - *prevBill.ProviderReadingF1
					summary.TotalProviderF1 += consumption.F1
				}
				if prevBill.ProviderReadingF2 != nil && bill.ProviderReadingF2 != nil {
					consumption.F2 = *bill.ProviderReadingF2 - *prevBill.ProviderReadingF2
					summary.TotalProviderF2 += consumption.F2
				}
				if prevBill.ProviderReadingF3 != nil && bill.ProviderReadingF3 != nil {
					consumption.F3 = *bill.ProviderReadingF3 - *prevBill.ProviderReadingF3
					summary.TotalProviderF3 += consumption.F3
				}
			} else {
				// First bill: estimate F1/F2/F3 distribution from total if we can't calculate
				// For now, leave them at 0 - the total is what matters for comparison
			}
		}

		providerConsumptions[bill.PeriodEnd] = consumption
	}

	// Calculate user consumption (from consecutive self-readings)
	userConsumptions := make(map[time.Time]struct {
		F1, F2, F3, Total float64
	})

	for i := 1; i < len(sortedReadings); i++ {
		prev := sortedReadings[i-1]
		curr := sortedReadings[i]

		consumption := struct {
			F1, F2, F3, Total float64
		}{}

		if utilityType == "electricity" {
			if prev.ValueF1 != nil && curr.ValueF1 != nil {
				consumption.F1 = *curr.ValueF1 - *prev.ValueF1
				summary.TotalUserF1 += consumption.F1
			}
			if prev.ValueF2 != nil && curr.ValueF2 != nil {
				consumption.F2 = *curr.ValueF2 - *prev.ValueF2
				summary.TotalUserF2 += consumption.F2
			}
			if prev.ValueF3 != nil && curr.ValueF3 != nil {
				consumption.F3 = *curr.ValueF3 - *prev.ValueF3
				summary.TotalUserF3 += consumption.F3
			}
			consumption.Total = consumption.F1 + consumption.F2 + consumption.F3
		} else {
			if prev.Value != nil && curr.Value != nil {
				consumption.Total = *curr.Value - *prev.Value
			}
		}

		summary.TotalUser += consumption.Total
		userConsumptions[curr.ReadingDate] = consumption
	}

	// Create periods for ALL bills (including the first one)
	for i, bill := range sortedBills {
		period := ConsumptionPeriod{
			PeriodStart: bill.PeriodStart,
			PeriodEnd:   bill.PeriodEnd,
			BillID:      &bill.ID,
		}

		// Provider consumption for this period (from consumption_total)
		if prov, ok := providerConsumptions[bill.PeriodEnd]; ok {
			if utilityType == "electricity" && i > 0 {
				// Per-band breakdown only available for bills after the first
				period.ProviderConsumptionF1 = &prov.F1
				period.ProviderConsumptionF2 = &prov.F2
				period.ProviderConsumptionF3 = &prov.F3
			}
			period.ProviderConsumption = &prov.Total
		}

		// Find user consumption that matches this bill period
		// Look for user readings that fall within or close to this bill's period
		for readingDate, user := range userConsumptions {
			// User reading should be within the bill period (with some tolerance)
			periodStart := bill.PeriodStart.AddDate(0, 0, -7) // 7 days before period start
			periodEnd := bill.PeriodEnd.AddDate(0, 0, 7)      // 7 days after period end

			if !readingDate.Before(periodStart) && !readingDate.After(periodEnd) {
				if utilityType == "electricity" {
					period.UserConsumptionF1 = &user.F1
					period.UserConsumptionF2 = &user.F2
					period.UserConsumptionF3 = &user.F3
				}
				period.UserConsumption = &user.Total

				// Calculate differences
				if period.ProviderConsumption != nil && period.UserConsumption != nil {
					diff := *period.ProviderConsumption - *period.UserConsumption
					period.Difference = &diff
				}
				if utilityType == "electricity" {
					if period.ProviderConsumptionF1 != nil && period.UserConsumptionF1 != nil {
						diff := *period.ProviderConsumptionF1 - *period.UserConsumptionF1
						period.DifferenceF1 = &diff
					}
					if period.ProviderConsumptionF2 != nil && period.UserConsumptionF2 != nil {
						diff := *period.ProviderConsumptionF2 - *period.UserConsumptionF2
						period.DifferenceF2 = &diff
					}
					if period.ProviderConsumptionF3 != nil && period.UserConsumptionF3 != nil {
						diff := *period.ProviderConsumptionF3 - *period.UserConsumptionF3
						period.DifferenceF3 = &diff
					}
				}
				break
			}
		}

		periods = append(periods, period)
	}

	// Calculate cumulative differences
	summary.CumulativeDifferenceF1 = summary.TotalProviderF1 - summary.TotalUserF1
	summary.CumulativeDifferenceF2 = summary.TotalProviderF2 - summary.TotalUserF2
	summary.CumulativeDifferenceF3 = summary.TotalProviderF3 - summary.TotalUserF3
	summary.CumulativeDifference = summary.TotalProvider - summary.TotalUser

	// Set period range based on actual data available
	// Use the earliest of first bill or first reading, and latest of last bill or last reading
	if len(sortedBills) > 0 {
		summary.FirstPeriod = sortedBills[0].PeriodStart
		summary.LastPeriod = sortedBills[len(sortedBills)-1].PeriodEnd
	}
	if len(sortedReadings) > 0 {
		if summary.FirstPeriod.IsZero() || sortedReadings[0].ReadingDate.Before(summary.FirstPeriod) {
			summary.FirstPeriod = sortedReadings[0].ReadingDate
		}
		if summary.LastPeriod.IsZero() || sortedReadings[len(sortedReadings)-1].ReadingDate.After(summary.LastPeriod) {
			summary.LastPeriod = sortedReadings[len(sortedReadings)-1].ReadingDate
		}
	}

	// Check for cumulative alert
	// IMPORTANT: We only alert when provider charged MORE than actual consumption (positive difference)
	// When provider charged less, it's not a problem (at worst, there might be a year-end adjustment)
	numPeriods := len(periods)
	if numPeriods > 0 {
		cumulativeThreshold := threshold * float64(numPeriods)

		// Only alert when provider charged MORE (positive difference)
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
			// Provider charged less - just informational, no alert level
			summary.CumulativeMessage = fmt.Sprintf("Il fornitore ha fatturato %.1f unità in meno rispetto ai consumi rilevati in %d periodi. Potrebbe esserci un conguaglio a fine anno.",
				-summary.CumulativeDifference, numPeriods)
		}
	}

	return periods, summary
}
