package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UtilityHandler struct {
	db *gorm.DB
}

func NewUtilityHandler(db *gorm.DB) *UtilityHandler {
	return &UtilityHandler{db: db}
}

// Standard CRUD
func (h *UtilityHandler) List(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *UtilityHandler) Create(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *UtilityHandler) Get(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *UtilityHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *UtilityHandler) Delete(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// Meter readings
func (h *UtilityHandler) AddReading(c *gin.Context) {
	// TODO: POST /api/v1/utilities/:id/readings
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *UtilityHandler) GetReadings(c *gin.Context) {
	// TODO: GET /api/v1/utilities/:id/readings
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// Bills
func (h *UtilityHandler) AddBill(c *gin.Context) {
	// TODO: POST /api/v1/utilities/:id/bills
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *UtilityHandler) GetBills(c *gin.Context) {
	// TODO: GET /api/v1/utilities/:id/bills
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
