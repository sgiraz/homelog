package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	db *gorm.DB
}

func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

type CreateProjectRequest struct {
	PropertyID       *uint   `json:"property_id"`
	Name             string  `json:"name" binding:"required"`
	Icon             string  `json:"icon"`
	Description      string  `json:"description"`
	Budget           float64 `json:"budget" binding:"required,gt=0"`
	StartDate        string  `json:"start_date" binding:"required"`
	EndDate          string  `json:"end_date" binding:"required"`
	Status           string  `json:"status"`
	SharedWithUserIDs []uint `json:"shared_with_user_ids"`
}

type ProjectStatsResponse struct {
	TotalBudget     float64 `json:"total_budget"`
	TotalSpent      float64 `json:"total_spent"`
	Remaining       float64 `json:"remaining"`
	PercentageSpent float64 `json:"percentage_spent"`
	ExpenseCount    int     `json:"expense_count"`
}

// List - GET /api/v1/projects?property_id=X&status=active
func (h *ProjectHandler) List(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	propertyID := c.Query("property_id")
	status := c.Query("status")

	// Visibility: owned by user OR shared with user
	query := h.db.Where(
		"user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?)",
		userID, userID,
	)

	if propertyID != "" {
		query = query.Where("property_id = ?", propertyID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var projects []models.Project
	if err := query.Preload("Expenses").
		Preload("SharedWith").
		Order("status ASC, start_date DESC").
		Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	type ProjectWithStats struct {
		models.Project
		Stats ProjectStatsResponse `json:"stats"`
	}

	projectsWithStats := make([]ProjectWithStats, len(projects))
	for i, proj := range projects {
		stats := calculateProjectStats(proj)
		projectsWithStats[i] = ProjectWithStats{
			Project: proj,
			Stats:   stats,
		}
	}

	log.Printf("User %d has %d projects", userID, len(projects))
	c.JSON(http.StatusOK, projectsWithStats)
}

// Create - POST /api/v1/projects
func (h *ProjectHandler) Create(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
		return
	}

	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	status := req.Status
	if status == "" {
		status = "planned"
	}

	validStatuses := []string{"planned", "active", "completed", "cancelled"}
	isValidStatus := false
	for _, s := range validStatuses {
		if status == s {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be: planned, active, completed, cancelled"})
		return
	}

	project := models.Project{
		UserID:      userID,
		PropertyID:  req.PropertyID,
		Name:        req.Name,
		Icon:        req.Icon,
		Description: req.Description,
		Budget:      req.Budget,
		StartDate:   startDate,
		EndDate:     endDate,
		Status:      status,
	}

	if err := h.db.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	// Share with specified users (only users different from creator)
	if len(req.SharedWithUserIDs) > 0 {
		var users []models.User
		h.db.Where("id IN ? AND id != ?", req.SharedWithUserIDs, userID).Find(&users)
		if len(users) > 0 {
			if err := h.db.Model(&project).Association("SharedWith").Append(&users); err != nil {
				log.Printf("⚠️  Failed to set shared_with for project %d: %v", project.ID, err)
			}
		}
	}

	// Reload with associations
	h.db.Preload("SharedWith").First(&project, project.ID)

	log.Printf("Project created: ID=%d, Name=%s, Budget=%.2f, SharedWith=%d users", project.ID, project.Name, project.Budget, len(project.SharedWith))
	c.JSON(http.StatusCreated, project)
}

// Get - GET /api/v1/projects/:id
func (h *ProjectHandler) Get(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	projectID := c.Param("id")

	var project models.Project
	if err := h.db.Where(
		"id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ?))",
		projectID, userID, userID,
	).
		Preload("Expenses").
		Preload("Expenses.Category").
		Preload("SharedWith").
		First(&project).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project"})
		}
		return
	}

	stats := calculateProjectStats(project)

	response := struct {
		models.Project
		Stats ProjectStatsResponse `json:"stats"`
	}{
		Project: project,
		Stats:   stats,
	}

	c.JSON(http.StatusOK, response)
}

// Update - PUT /api/v1/projects/:id (creator only)
func (h *ProjectHandler) Update(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	projectID := c.Param("id")

	// Only creator can edit
	var project models.Project
	if err := h.db.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or you are not the creator"})
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDate, _ := time.Parse("2006-01-02", req.EndDate)

	project.Name = req.Name
	project.Icon = req.Icon
	project.Description = req.Description
	project.Budget = req.Budget
	project.StartDate = startDate
	project.EndDate = endDate

	if req.Status != "" {
		project.Status = req.Status
	}

	if err := h.db.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	// Replace shared users (only users different from creator)
	var users []models.User
	if len(req.SharedWithUserIDs) > 0 {
		h.db.Where("id IN ? AND id != ?", req.SharedWithUserIDs, userID).Find(&users)
	}
	if err := h.db.Model(&project).Association("SharedWith").Replace(&users); err != nil {
		log.Printf("⚠️  Failed to update shared_with for project %d: %v", project.ID, err)
	}

	// Reload with associations
	h.db.Preload("SharedWith").First(&project, project.ID)

	log.Printf("Project updated: ID=%d, Name=%s", project.ID, project.Name)
	c.JSON(http.StatusOK, project)
}

// Delete - DELETE /api/v1/projects/:id (creator only)
func (h *ProjectHandler) Delete(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	projectID := c.Param("id")

	var project models.Project
	if err := h.db.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or you are not the creator"})
		return
	}

	var expenseCount int64
	h.db.Model(&models.Expense{}).Where("project_id = ?", projectID).Count(&expenseCount)

	if expenseCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Cannot delete project with associated expenses",
			"message": "Remove or reassign expenses first",
			"count":   expenseCount,
		})
		return
	}

	// Clear shared_with associations before deleting
	h.db.Model(&project).Association("SharedWith").Clear()

	if err := h.db.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	log.Printf("Project deleted: ID=%d", project.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}

func calculateProjectStats(project models.Project) ProjectStatsResponse {
	var totalSpent float64
	expenseCount := len(project.Expenses)

	for _, expense := range project.Expenses {
		totalSpent += expense.Amount
	}

	remaining := project.Budget - totalSpent
	percentageSpent := 0.0
	if project.Budget > 0 {
		percentageSpent = (totalSpent / project.Budget) * 100
	}

	return ProjectStatsResponse{
		TotalBudget:     project.Budget,
		TotalSpent:      totalSpent,
		Remaining:       remaining,
		PercentageSpent: percentageSpent,
		ExpenseCount:    expenseCount,
	}
}
