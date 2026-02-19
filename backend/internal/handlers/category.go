package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	db *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
	return &CategoryHandler{db: db}
}

// List - GET /api/v1/categories
func (h *CategoryHandler) List(c *gin.Context) {
	var categories []models.Category

	// Fetch all default categories + user's custom categories
	// For now, just default categories (UserID = nil)
	if err := h.db.Where("is_default = ?", true).
		Preload("Subcategories").
		Order("name ASC").
		Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// Create - POST /api/v1/categories
func (h *CategoryHandler) Create(c *gin.Context) {
	// TODO: Create category
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// Get - GET /api/v1/categories/:id
func (h *CategoryHandler) Get(c *gin.Context) {
	// TODO: Get single category
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// Update - PUT /api/v1/categories/:id
func (h *CategoryHandler) Update(c *gin.Context) {
	// TODO: Update category
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// Delete - DELETE /api/v1/categories/:id
func (h *CategoryHandler) Delete(c *gin.Context) {
	// TODO: Delete category
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
