package handlers

import (
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
		PropertyID     uint       `json:"property_id" binding:"required"`
		Type           string     `json:"type" binding:"required,oneof=electricity gas water waste"`
		Provider       string     `json:"provider" binding:"required"`
		CustomerCode   string     `json:"customer_code"`
		ServiceCode    string     `json:"service_code"`
		Address        string     `json:"address"`
		StartDate      time.Time  `json:"start_date"`
		EndDate        *time.Time `json:"end_date"`
		IsActive       bool       `json:"is_active"`
		PowerCapacity  float64    `json:"power_capacity"`
		CustomerPortal string     `json:"customer_portal"`
		Notes          string     `json:"notes"`
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

	utility := models.Utility{
		UserID:         userID,
		PropertyID:     input.PropertyID,
		Type:           input.Type,
		Provider:       input.Provider,
		CustomerCode:   input.CustomerCode,
		ServiceCode:    input.ServiceCode,
		Address:        input.Address,
		StartDate:      input.StartDate,
		EndDate:        input.EndDate,
		IsActive:       true,
		PowerCapacity:  input.PowerCapacity,
		CustomerPortal: input.CustomerPortal,
		Notes:          input.Notes,
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
		Provider       string     `json:"provider"`
		CustomerCode   string     `json:"customer_code"`
		ServiceCode    string     `json:"service_code"`
		Address        string     `json:"address"`
		StartDate      *time.Time `json:"start_date"`
		EndDate        *time.Time `json:"end_date"`
		IsActive       *bool      `json:"is_active"`
		PowerCapacity  *float64   `json:"power_capacity"`
		CustomerPortal string     `json:"customer_portal"`
		Notes          string     `json:"notes"`
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
		utility.IsActive = *input.IsActive
	}
	if input.PowerCapacity != nil {
		utility.PowerCapacity = *input.PowerCapacity
	}
	if input.CustomerPortal != "" {
		utility.CustomerPortal = input.CustomerPortal
	}
	utility.Notes = input.Notes

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
		ValueF1     *float64  `json:"value_f1"`     // Electricity F1 (peak)
		ValueF2     *float64  `json:"value_f2"`     // Electricity F2 (mid)
		ValueF3     *float64  `json:"value_f3"`     // Electricity F3 (off-peak)
		Value       *float64  `json:"value"`        // Gas/Water single reading (mc/Smc)
		Source      string    `json:"source"`       // manual, submitted
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
		ConsumptionTotal    float64    `json:"consumption_total" binding:"required"`
		ConsumptionF1       *float64   `json:"consumption_f1"`
		ConsumptionF2       *float64   `json:"consumption_f2"`
		ConsumptionF3       *float64   `json:"consumption_f3"`
		AmountTotal         float64    `json:"amount_total" binding:"required"`
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
		UtilityID:           uint(utilityID),
		BillNumber:          input.BillNumber,
		IssueDate:           input.IssueDate,
		PeriodStart:         input.PeriodStart,
		PeriodEnd:           input.PeriodEnd,
		DueDate:             input.DueDate,
		ReadingStartDate:    input.ReadingStartDate,
		ReadingStartValue:   input.ReadingStartValue,
		ReadingEndDate:      input.ReadingEndDate,
		ReadingEndValue:     input.ReadingEndValue,
		ReadingType:         input.ReadingType,
		ProviderReadingDate: input.ProviderReadingDate,
		ProviderReadingF1:   input.ProviderReadingF1,
		ProviderReadingF2:   input.ProviderReadingF2,
		ProviderReadingF3:   input.ProviderReadingF3,
		ProviderReading:     input.ProviderReading,
		ConsumptionTotal:    input.ConsumptionTotal,
		ConsumptionF1:       input.ConsumptionF1,
		ConsumptionF2:       input.ConsumptionF2,
		ConsumptionF3:       input.ConsumptionF3,
		AmountTotal:         input.AmountTotal,
		AmountEnergy:        input.AmountEnergy,
		AmountFixed:         input.AmountFixed,
		AmountTaxes:         input.AmountTaxes,
		AmountVAT:           input.AmountVAT,
		IsPaid:              input.IsPaid,
		PaidDate:            input.PaidDate,
		PDFURL:              input.PDFURL,
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
		BillNumber       string     `json:"bill_number"`
		IssueDate        time.Time  `json:"issue_date"`
		PeriodStart      time.Time  `json:"period_start"`
		PeriodEnd        time.Time  `json:"period_end"`
		DueDate          time.Time  `json:"due_date"`
		ConsumptionTotal float64    `json:"consumption_total"`
		AmountTotal      float64    `json:"amount_total"`
		ReadingType      string     `json:"reading_type"`
		IsPaid           bool       `json:"is_paid"`
		PaidDate         *time.Time `json:"paid_date"`
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
	bill.ReadingType = input.ReadingType
	bill.IsPaid = input.IsPaid
	bill.PaidDate = input.PaidDate

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
	ReadingType         string     `json:"reading_type"`           // actual or estimated
	ProviderReadingDate *time.Time `json:"provider_reading_date"`
	UserReadingDate     *time.Time `json:"user_reading_date"`
	// For electricity (F1/F2/F3)
	ProviderF1    *float64 `json:"provider_f1,omitempty"`
	ProviderF2    *float64 `json:"provider_f2,omitempty"`
	ProviderF3    *float64 `json:"provider_f3,omitempty"`
	UserF1        *float64 `json:"user_f1,omitempty"`
	UserF2        *float64 `json:"user_f2,omitempty"`
	UserF3        *float64 `json:"user_f3,omitempty"`
	DifferenceF1  *float64 `json:"difference_f1,omitempty"`
	DifferenceF2  *float64 `json:"difference_f2,omitempty"`
	DifferenceF3  *float64 `json:"difference_f3,omitempty"`
	PercentageF1  *float64 `json:"percentage_f1,omitempty"`
	PercentageF2  *float64 `json:"percentage_f2,omitempty"`
	PercentageF3  *float64 `json:"percentage_f3,omitempty"`
	// For gas/water (single value)
	ProviderReading   *float64 `json:"provider_reading,omitempty"`
	UserReading       *float64 `json:"user_reading,omitempty"`
	Difference        *float64 `json:"difference,omitempty"`
	DifferencePercent *float64 `json:"difference_percent,omitempty"`
	// Status
	Status       string `json:"status"`        // ok, warning, alert
	AlertMessage string `json:"alert_message"` // Human readable message
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

	// Get all user readings
	var readings []models.MeterReading
	h.db.Where("utility_id = ?", utilityID).
		Order("reading_date DESC").
		Find(&readings)

	// Default threshold (5%)
	threshold := 5.0
	if t := c.Query("threshold"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil {
			threshold = parsed
		}
	}

	var comparisons []ReadingComparison

	for _, bill := range bills {
		// Skip bills without provider readings
		hasProviderReading := false
		if utility.Type == "electricity" {
			hasProviderReading = bill.ProviderReadingF1 != nil || bill.ProviderReadingF2 != nil || bill.ProviderReadingF3 != nil
		} else {
			hasProviderReading = bill.ProviderReading != nil
		}

		if !hasProviderReading {
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
		}

		// Find the closest user reading to the provider reading date
		var closestReading *models.MeterReading
		var minDiff time.Duration = time.Hour * 24 * 365 // 1 year max

		targetDate := bill.PeriodEnd
		if bill.ProviderReadingDate != nil {
			targetDate = *bill.ProviderReadingDate
		}

		for i := range readings {
			diff := readings[i].ReadingDate.Sub(targetDate)
			if diff < 0 {
				diff = -diff
			}
			// Only consider readings within 7 days
			if diff < minDiff && diff <= time.Hour*24*7 {
				minDiff = diff
				closestReading = &readings[i]
			}
		}

		if closestReading != nil {
			comparison.UserReadingDate = &closestReading.ReadingDate
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

				// Calculate differences for each band
				maxPercentage := 0.0

				if bill.ProviderReadingF1 != nil && closestReading.ValueF1 != nil {
					diff := *bill.ProviderReadingF1 - *closestReading.ValueF1
					comparison.DifferenceF1 = &diff
					if *closestReading.ValueF1 != 0 {
						pct := (diff / *closestReading.ValueF1) * 100
						comparison.PercentageF1 = &pct
						if pct > maxPercentage || -pct > maxPercentage {
							if pct < 0 {
								maxPercentage = -pct
							} else {
								maxPercentage = pct
							}
						}
					}
				}

				if bill.ProviderReadingF2 != nil && closestReading.ValueF2 != nil {
					diff := *bill.ProviderReadingF2 - *closestReading.ValueF2
					comparison.DifferenceF2 = &diff
					if *closestReading.ValueF2 != 0 {
						pct := (diff / *closestReading.ValueF2) * 100
						comparison.PercentageF2 = &pct
						if pct > maxPercentage || -pct > maxPercentage {
							if pct < 0 {
								maxPercentage = -pct
							} else {
								maxPercentage = pct
							}
						}
					}
				}

				if bill.ProviderReadingF3 != nil && closestReading.ValueF3 != nil {
					diff := *bill.ProviderReadingF3 - *closestReading.ValueF3
					comparison.DifferenceF3 = &diff
					if *closestReading.ValueF3 != 0 {
						pct := (diff / *closestReading.ValueF3) * 100
						comparison.PercentageF3 = &pct
						if pct > maxPercentage || -pct > maxPercentage {
							if pct < 0 {
								maxPercentage = -pct
							} else {
								maxPercentage = pct
							}
						}
					}
				}

				// Determine status
				if maxPercentage > threshold*2 {
					comparison.Status = "alert"
					comparison.AlertMessage = "Discrepanza significativa tra letture fornitore e autolettura"
				} else if maxPercentage > threshold {
					comparison.Status = "warning"
					comparison.AlertMessage = "Differenza rilevata tra letture fornitore e autolettura"
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

					if *closestReading.Value != 0 {
						pct := (diff / *closestReading.Value) * 100
						comparison.DifferencePercent = &pct

						// Determine status
						absPct := pct
						if absPct < 0 {
							absPct = -absPct
						}

						if absPct > threshold*2 {
							comparison.Status = "alert"
							comparison.AlertMessage = "Discrepanza significativa tra letture fornitore e autolettura"
						} else if absPct > threshold {
							comparison.Status = "warning"
							comparison.AlertMessage = "Differenza rilevata tra letture fornitore e autolettura"
						}
					}
				}
			} else {
				comparison.Status = "no_data"
				comparison.AlertMessage = "Nessuna autolettura disponibile per il confronto"
			}
		}

		comparisons = append(comparisons, comparison)
	}

	c.JSON(http.StatusOK, gin.H{
		"comparisons": comparisons,
		"threshold":   threshold,
		"utility_type": utility.Type,
	})
}
