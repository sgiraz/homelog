package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

type DeviceHandler struct {
	db *gorm.DB
}

func NewDeviceHandler(db *gorm.DB) *DeviceHandler {
	return &DeviceHandler{db: db}
}

type CreateDeviceRequest struct {
	PropertyID    uint     `json:"property_id"`
	CategoryID    *uint    `json:"category_id"`
	Name          string   `json:"name"`
	Manufacturer  string   `json:"manufacturer"`
	Model         string   `json:"model"`
	SerialNumber  string   `json:"serial_number"`
	Status        string   `json:"status"`
	Location      string   `json:"location"`
	Notes         string   `json:"notes"`
	PurchaseDate  string   `json:"purchase_date"`
	PurchasePrice *float64 `json:"purchase_price"`
	WarrantyUntil string   `json:"warranty_until"`
	Hostname      string   `json:"hostname"`
	IPAddress     string   `json:"ip_address"`
	MACAddress    string   `json:"mac_address"`
	Firmware      string   `json:"firmware"`
	ManagementURL string   `json:"management_url"`
}
type UpdateDeviceRequest struct {
	CategoryID    *uint    `json:"category_id"`
	Name          string   `json:"name"`
	Manufacturer  string   `json:"manufacturer"`
	Model         string   `json:"model"`
	SerialNumber  string   `json:"serial_number"`
	Status        string   `json:"status"`
	Location      string   `json:"location"`
	Notes         string   `json:"notes"`
	PurchaseDate  string   `json:"purchase_date"`
	PurchasePrice *float64 `json:"purchase_price"`
	WarrantyUntil string   `json:"warranty_until"`
	Hostname      string   `json:"hostname"`
	IPAddress     string   `json:"ip_address"`
	MACAddress    string   `json:"mac_address"`
	Firmware      string   `json:"firmware"`
	ManagementURL string   `json:"management_url"`
}

// List - GET /api/v1/devices
func (h *DeviceHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	propertyID := c.Query("property_id")
	categoryID := c.Query("category_id")
	status := c.Query("status")
	q := c.Query("q")

	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	var devices []models.Device

	query := h.db.
		Preload("Category").
		Preload("Property").
		Where("property_id IN (?)",
			h.db.Model(&models.HouseholdMember{}).
				Select("property_id").
				Where("user_id = ?", userID),
		)

	if propertyID != "" {
		query = query.Where("property_id = ?", propertyID)
	}

	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"name LIKE ? OR manufacturer LIKE ? OR model LIKE ? OR serial_number LIKE ? OR hostname LIKE ?",
			like, like, like, like, like,
		)
	}

	var total int64
	countQuery := h.db.
		Model(&models.Device{}).
		Where("property_id IN (?)",
			h.db.Model(&models.HouseholdMember{}).
				Select("property_id").
				Where("user_id = ?", userID),
		)

	if propertyID != "" {
		countQuery = countQuery.Where("property_id = ?", propertyID)
	}

	if categoryID != "" {
		countQuery = countQuery.Where("category_id = ?", categoryID)
	}

	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}

	if q != "" {
		like := "%" + q + "%"
		countQuery = countQuery.Where(
			"name LIKE ? OR manufacturer LIKE ? OR model LIKE ? OR serial_number LIKE ? OR hostname LIKE ?",
			like, like, like, like, like,
		)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to count devices")
		return
	}

	if err := query.
		Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&devices).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch devices")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"limit":   limit,
		"offset":  offset,
		"total":   total,
	})
}

// Create - POST /api/v1/devices
func (h *DeviceHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid device data")
		return
	}

	device := models.Device{
		PropertyID:    req.PropertyID,
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Manufacturer:  req.Manufacturer,
		Model:         req.Model,
		SerialNumber:  req.SerialNumber,
		Status:        req.Status,
		Location:      req.Location,
		Notes:         req.Notes,
		PurchasePrice: req.PurchasePrice,
		Hostname:      req.Hostname,
		IPAddress:     req.IPAddress,
		MACAddress:    req.MACAddress,
		Firmware:      req.Firmware,
		ManagementURL: req.ManagementURL,
	}

	if req.PurchaseDate != "" {
		purchaseDate, err := time.Parse("2006-01-02", req.PurchaseDate)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid purchase date")
			return
		}
		device.PurchaseDate = &purchaseDate
	}

	if req.WarrantyUntil != "" {
		warrantyUntil, err := time.Parse("2006-01-02", req.WarrantyUntil)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid warranty date")
			return
		}
		device.WarrantyUntil = &warrantyUntil
	}

	if device.Name == "" {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Device name is required")
		return
	}

	if device.PropertyID == 0 {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Property is required")
		return
	}

	if device.CategoryID != nil {
		var category models.DeviceCategory
		if err := h.db.
			Where("id = ? AND (user_id IS NULL OR user_id = ?)", *device.CategoryID, userID).
			First(&category).Error; err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid device category")
			return
		}
	}

	if !requirePropertyMember(c, h.db, userID, device.PropertyID) {
		return
	}

	device.UserID = userID

	if device.Status == "" {
		device.Status = "active"
	}

	if err := h.db.Create(&device).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create device")
		return
	}

	if err := h.db.
		Preload("Category").
		Preload("Property").
		First(&device, device.ID).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch created device")
		return
	}

	c.JSON(http.StatusCreated, device)
}

// Get - GET /api/v1/devices/:id
func (h *DeviceHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var device models.Device
	if err := h.db.
		Preload("Category").
		Preload("Property").
		Preload("Expenses").
		First(&device, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusNotFound, "invalid_request", "Device not found")
			return
		}

		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch device")
		return
	}

	if !requirePropertyMember(c, h.db, userID, device.PropertyID) {
		return
	}

	c.JSON(http.StatusOK, device)
}

// Update - PUT /api/v1/devices/:id
func (h *DeviceHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var device models.Device
	if err := h.db.First(&device, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Device not found")
			return
		}

		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch device")
		return
	}

	if !requirePropertyMember(c, h.db, userID, device.PropertyID) {
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid device data")
		return
	}

	if req.Name == "" {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Device name is required")
		return
	}

	if req.CategoryID != nil {
		var category models.DeviceCategory
		if err := h.db.
			Where("id = ? AND (user_id IS NULL OR user_id = ?)", *req.CategoryID, userID).
			First(&category).Error; err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid device category")
			return
		}
	}

	input := models.Device{
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Manufacturer:  req.Manufacturer,
		Model:         req.Model,
		SerialNumber:  req.SerialNumber,
		Status:        req.Status,
		Location:      req.Location,
		Notes:         req.Notes,
		PurchasePrice: req.PurchasePrice,
		Hostname:      req.Hostname,
		IPAddress:     req.IPAddress,
		MACAddress:    req.MACAddress,
		Firmware:      req.Firmware,
		ManagementURL: req.ManagementURL,
	}

	input.ID = device.ID
	input.UserID = device.UserID
	input.PropertyID = device.PropertyID

	if input.Status == "" {
		input.Status = device.Status
	}

	if req.PurchaseDate == "" {
		input.PurchaseDate = nil
	} else {
		purchaseDate, err := time.Parse("2006-01-02", req.PurchaseDate)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid purchase date")
			return
		}
		input.PurchaseDate = &purchaseDate
	}

	if req.WarrantyUntil == "" {
		input.WarrantyUntil = nil
	} else {
		warrantyUntil, err := time.Parse("2006-01-02", req.WarrantyUntil)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid warranty date")
			return
		}
		input.WarrantyUntil = &warrantyUntil
	}

	if err := h.db.Model(&device).Updates(&input).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update device")
		return
	}

	if err := h.db.
		Preload("Category").
		Preload("Property").
		First(&device, device.ID).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch updated device")
		return
	}

	c.JSON(http.StatusOK, device)
}

// Delete - DELETE /api/v1/devices/:id
func (h *DeviceHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var device models.Device
	if err := h.db.First(&device, c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Device not found")
			return
		}

		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch device")
		return
	}

	if !requirePropertyMember(c, h.db, userID, device.PropertyID) {
		return
	}

	if err := h.db.Delete(&device).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete device")
		return
	}

	c.Status(http.StatusNoContent)
}
