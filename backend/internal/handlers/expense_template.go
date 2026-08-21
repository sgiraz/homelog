package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

type ExpenseTemplateHandler struct {
	db *gorm.DB
}

func NewExpenseTemplateHandler(db *gorm.DB) *ExpenseTemplateHandler {
	return &ExpenseTemplateHandler{db: db}
}

type CreateExpenseTemplateRequest struct {
	Name          string  `json:"name" binding:"required"`
	Icon          string  `json:"icon"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Description   string  `json:"description"`
	CategoryID    uint    `json:"category_id" binding:"required"`
	SubcategoryID *uint   `json:"subcategory_id"`
	ProjectID     *uint   `json:"project_id"`
	SortOrder     int     `json:"sort_order"`
}

// List - GET /api/v1/expense-templates
func (h *ExpenseTemplateHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var templates []models.ExpenseTemplate
	if err := h.db.Where("user_id = ?", userID).
		Preload("Category").
		Preload("Subcategory").
		Preload("Project").
		Order("sort_order ASC, name ASC").
		Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// Create - POST /api/v1/expense-templates
func (h *ExpenseTemplateHandler) Create(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateExpenseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template := models.ExpenseTemplate{
		UserID:        userID,
		Name:          req.Name,
		Icon:          req.Icon,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Description:   req.Description,
		CategoryID:    req.CategoryID,
		SubcategoryID: req.SubcategoryID,
		ProjectID:     req.ProjectID,
		SortOrder:     req.SortOrder,
	}

	if err := h.db.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	h.db.Preload("Category").Preload("Subcategory").Preload("Project").First(&template, template.ID)

	log.Printf("Expense template created: ID=%d, Name=%s", template.ID, template.Name)
	c.JSON(http.StatusCreated, template)
}

// Update - PUT /api/v1/expense-templates/:id
func (h *ExpenseTemplateHandler) Update(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id := c.Param("id")

	var template models.ExpenseTemplate
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&template).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	var req CreateExpenseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template.Name = req.Name
	template.Icon = req.Icon
	template.Amount = req.Amount
	template.Currency = req.Currency
	template.Description = req.Description
	template.CategoryID = req.CategoryID
	template.SubcategoryID = req.SubcategoryID
	template.ProjectID = req.ProjectID
	template.SortOrder = req.SortOrder

	if err := h.db.Save(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	h.db.Preload("Category").Preload("Subcategory").Preload("Project").First(&template, template.ID)

	log.Printf("Expense template updated: ID=%d, Name=%s", template.ID, template.Name)
	c.JSON(http.StatusOK, template)
}

// Delete - DELETE /api/v1/expense-templates/:id
func (h *ExpenseTemplateHandler) Delete(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id := c.Param("id")

	var template models.ExpenseTemplate
	if err := h.db.Where("id = ? AND user_id = ?", id, userID).First(&template).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	if err := h.db.Delete(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}

	log.Printf("Expense template deleted: ID=%d", template.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Template deleted"})
}
