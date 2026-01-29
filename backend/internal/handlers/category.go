package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	// TODO: List categories with subcategories
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
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
