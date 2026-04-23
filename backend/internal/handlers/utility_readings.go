package handlers

import (
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)


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

// UpdateReading updates a meter reading
func (h *UtilityHandler) UpdateReading(c *gin.Context) {
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

	// Enforce route contract: the reading must belong to the utility in the URL.
	if reading.UtilityID != uint(utilityID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reading not found"})
		return
	}

	// Verify access through utility
	var utility models.Utility
	if err := h.db.First(&utility, reading.UtilityID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	if !slices.Contains(memberPropertyIDs, utility.PropertyID) {
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

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid utility ID"})
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

	// Enforce route contract: the reading must belong to the utility in the URL.
	if reading.UtilityID != uint(utilityID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reading not found"})
		return
	}

	// Verify access through utility
	var utility models.Utility
	if err := h.db.First(&utility, reading.UtilityID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	if !slices.Contains(memberPropertyIDs, utility.PropertyID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this reading"})
		return
	}

	if err := h.db.Delete(&reading).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reading"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reading deleted successfully"})
}
