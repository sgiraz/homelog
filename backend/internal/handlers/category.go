package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	db *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
	return &CategoryHandler{db: db}
}

// List returns all default categories plus the user's personal categories.
// GET /api/v1/categories
func (h *CategoryHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var categories []models.Category
	if err := h.db.
		Where("is_default = true OR user_id = ?", userID).
		Preload("Subcategories").
		Order("is_default DESC, name ASC").
		Find(&categories).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch categories")
		return
	}

	c.JSON(http.StatusOK, categories)
}

// Create creates a new category.
// Admin users can create default categories (is_default = true).
// Regular users can only create personal categories.
// POST /api/v1/categories
func (h *CategoryHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var input struct {
		Name      string `json:"name" binding:"required"`
		Icon      string `json:"icon"`
		Color     string `json:"color"`
		IsDefault bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Only admins can create default (global) categories
	if input.IsDefault {
		role, _ := c.Get("user_role")
		if role != "admin" {
			apierr.Fail(c, http.StatusForbidden, "category_default_create_admin_only", "Only admins can create built-in categories")
			return
		}
	}

	cat := models.Category{
		Name:      input.Name,
		Icon:      input.Icon,
		Color:     input.Color,
		IsDefault: input.IsDefault,
	}
	if !input.IsDefault {
		cat.UserID = &userID
	}

	if err := h.db.Create(&cat).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create category")
		return
	}

	h.db.Preload("Subcategories").First(&cat, cat.ID)
	c.JSON(http.StatusCreated, cat)
}

// Get returns a single category.
// GET /api/v1/categories/:id
func (h *CategoryHandler) Get(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_category_id", "Invalid category id")
		return
	}

	var cat models.Category
	if err := h.db.
		Where("id = ? AND (is_default = true OR user_id = ?)", id, userID).
		Preload("Subcategories").
		First(&cat).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "category_not_found", "Category not found")
		return
	}

	c.JSON(http.StatusOK, cat)
}

// Update updates a category.
// Admins can update default categories. Users can only update their own personal categories.
// PUT /api/v1/categories/:id
func (h *CategoryHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_category_id", "Invalid category id")
		return
	}

	var cat models.Category
	if err := h.db.First(&cat, id).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "category_not_found", "Category not found")
		return
	}

	// Permission check
	role, _ := c.Get("user_role")
	if cat.IsDefault && role != "admin" {
		apierr.Fail(c, http.StatusForbidden, "category_default_modify_admin_only", "Only admins can edit built-in categories")
		return
	}
	if !cat.IsDefault && (cat.UserID == nil || *cat.UserID != userID) {
		apierr.Fail(c, http.StatusForbidden, "category_not_owned_modify", "You can only edit your own categories")
		return
	}

	var input struct {
		Name  *string `json:"name"`
		Icon  *string `json:"icon"`
		Color *string `json:"color"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	updates := map[string]any{}
	if input.Name != nil {
		updates["name"] = *input.Name
		// Renaming a built-in category drops its slug: the admin's own label
		// must win over the translated one, in every language. Without this the
		// UI would keep rendering t("categories.<slug>") and silently discard
		// the rename.
		if cat.Slug != "" && *input.Name != cat.Name {
			updates["slug"] = ""
		}
	}
	if input.Icon != nil {
		updates["icon"] = *input.Icon
	}
	if input.Color != nil {
		updates["color"] = *input.Color
	}

	if len(updates) > 0 {
		if err := h.db.Model(&cat).Updates(updates).Error; err != nil {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update category")
			return
		}
	}

	h.db.Preload("Subcategories").First(&cat, cat.ID)
	c.JSON(http.StatusOK, cat)
}

// Delete deletes a category.
// Admins can delete default categories. Users can only delete their own personal categories.
// DELETE /api/v1/categories/:id
func (h *CategoryHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_category_id", "Invalid category id")
		return
	}

	var cat models.Category
	if err := h.db.First(&cat, id).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "category_not_found", "Category not found")
		return
	}

	// Permission check
	role, _ := c.Get("user_role")
	if cat.IsDefault && role != "admin" {
		apierr.Fail(c, http.StatusForbidden, "category_default_delete_admin_only", "Only admins can delete built-in categories")
		return
	}
	if !cat.IsDefault && (cat.UserID == nil || *cat.UserID != userID) {
		apierr.Fail(c, http.StatusForbidden, "category_not_owned_delete", "You can only delete your own categories")
		return
	}

	// Block deletion if any expenses reference this category
	var expenseCount int64
	h.db.Model(&models.Expense{}).Where("category_id = ?", id).Count(&expenseCount)
	if expenseCount > 0 {
		apierr.Fail(c, http.StatusConflict, "category_in_use", "This category is used by existing expenses and cannot be deleted")
		return
	}

	// Delete subcategories first
	h.db.Where("category_id = ?", id).Delete(&models.Subcategory{})

	if err := h.db.Delete(&cat).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete category")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted"})
}

// CreateSubcategory adds a subcategory to an existing category.
// POST /api/v1/categories/:id/subcategories
func (h *CategoryHandler) CreateSubcategory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	catID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_category_id", "Invalid category id")
		return
	}

	var cat models.Category
	if err := h.db.First(&cat, catID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "category_not_found", "Category not found")
		return
	}

	// Permission check: only admin for default, owner for personal
	role, _ := c.Get("user_role")
	if cat.IsDefault && role != "admin" {
		apierr.Fail(c, http.StatusForbidden, "subcategory_default_create_admin_only", "Only admins can add subcategories to built-in categories")
		return
	}
	if !cat.IsDefault && (cat.UserID == nil || *cat.UserID != userID) {
		apierr.Fail(c, http.StatusForbidden, "not_authorized", "You are not authorized to do this")
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	sub := models.Subcategory{
		CategoryID: uint(catID),
		Name:       input.Name,
	}
	if err := h.db.Create(&sub).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create subcategory")
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// DeleteSubcategory removes a subcategory.
// DELETE /api/v1/categories/:id/subcategories/:subId
func (h *CategoryHandler) DeleteSubcategory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	catID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_category_id", "Invalid category id")
		return
	}
	subID, err := strconv.ParseUint(c.Param("subId"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_subcategory_id", "Invalid subcategory id")
		return
	}

	var cat models.Category
	if err := h.db.First(&cat, catID).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "category_not_found", "Category not found")
		return
	}

	// Permission check
	role, _ := c.Get("user_role")
	if cat.IsDefault && role != "admin" {
		apierr.Fail(c, http.StatusForbidden, "subcategory_default_delete_admin_only", "Only admins can delete subcategories of built-in categories")
		return
	}
	if !cat.IsDefault && (cat.UserID == nil || *cat.UserID != userID) {
		apierr.Fail(c, http.StatusForbidden, "not_authorized", "You are not authorized to do this")
		return
	}

	result := h.db.Where("id = ? AND category_id = ?", subID, catID).Delete(&models.Subcategory{})
	if result.Error != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to delete subcategory")
		return
	}
	if result.RowsAffected == 0 {
		apierr.Fail(c, http.StatusNotFound, "subcategory_not_found", "Subcategory not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Subcategory deleted"})
}
