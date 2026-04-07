package handlers

import (
	"log"
	"net/http"
	"strconv"
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

type MemberRoleInput struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role"` // "owner" or "member", defaults to "member"
}

type CreateProjectRequest struct {
	PropertyID        *uint             `json:"property_id"`
	Name              string            `json:"name" binding:"required"`
	Icon              string            `json:"icon"`
	Description       string            `json:"description"`
	Budget            float64           `json:"budget" binding:"required,gt=0"`
	StartDate         string            `json:"start_date" binding:"required"`
	EndDate           string            `json:"end_date" binding:"required"`
	Status            string            `json:"status"`
	SharedWithUserIDs []uint            `json:"shared_with_user_ids"` // backward compat
	Members           []MemberRoleInput `json:"members,omitempty"`   // new: with roles
}

type ProjectStatsResponse struct {
	TotalBudget     float64 `json:"total_budget"`
	TotalSpent      float64 `json:"total_spent"`
	Remaining       float64 `json:"remaining"`
	PercentageSpent float64 `json:"percentage_spent"`
	ExpenseCount    int     `json:"expense_count"`
}

type ProjectMemberResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"` // creator, owner, member
}

// canManageProject checks if the user is the creator or has "owner" role
func (h *ProjectHandler) canManageProject(projectID string, userID uint) (models.Project, bool) {
	var project models.Project
	id, err := strconv.ParseUint(projectID, 10, 32)
	if err != nil {
		return project, false
	}
	if err := h.db.Where(
		"id = ? AND (user_id = ? OR id IN (SELECT project_id FROM project_members WHERE user_id = ? AND role = 'owner'))",
		id, userID, userID,
	).First(&project).Error; err != nil {
		return project, false
	}
	return project, true
}

// buildMembersResponse builds the members list including the creator
func (h *ProjectHandler) buildMembersResponse(project models.Project) []ProjectMemberResponse {
	var members []ProjectMemberResponse

	// Add creator
	var creator models.User
	if err := h.db.First(&creator, project.UserID).Error; err == nil {
		members = append(members, ProjectMemberResponse{
			ID:   creator.ID,
			Name: creator.Name,
			Role: "creator",
		})
	}

	// Add shared members with their roles
	var projectMembers []models.ProjectMember
	h.db.Where("project_id = ?", project.ID).Preload("User").Find(&projectMembers)
	for _, pm := range projectMembers {
		members = append(members, ProjectMemberResponse{
			ID:   pm.User.ID,
			Name: pm.User.Name,
			Role: pm.Role,
		})
	}

	return members
}

// setProjectMembers replaces all shared members with the given list
func (h *ProjectHandler) setProjectMembers(project *models.Project, req CreateProjectRequest, creatorID uint) {
	// Delete existing members
	h.db.Where("project_id = ?", project.ID).Delete(&models.ProjectMember{})

	if len(req.Members) > 0 {
		// New format: members with roles
		for _, m := range req.Members {
			if m.UserID == creatorID {
				continue
			}
			role := m.Role
			if role != "owner" {
				role = "member"
			}
			pm := models.ProjectMember{ProjectID: project.ID, UserID: m.UserID, Role: role}
			h.db.Create(&pm)
		}
	} else if len(req.SharedWithUserIDs) > 0 {
		// Backward compat: all as "member"
		var users []models.User
		h.db.Where("id IN ? AND id != ?", req.SharedWithUserIDs, creatorID).Find(&users)
		for _, u := range users {
			pm := models.ProjectMember{ProjectID: project.ID, UserID: u.ID, Role: "member"}
			h.db.Create(&pm)
		}
	}
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
		Preload("Members").
		Preload("Members.User").
		Order("status ASC, start_date DESC").
		Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	type ProjectWithStats struct {
		models.Project
		Stats   ProjectStatsResponse    `json:"stats"`
		Members []ProjectMemberResponse `json:"members"`
	}

	projectsWithStats := make([]ProjectWithStats, len(projects))
	for i, proj := range projects {
		stats := calculateProjectStats(proj)
		projectsWithStats[i] = ProjectWithStats{
			Project: proj,
			Stats:   stats,
			Members: h.buildMembersResponse(proj),
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

	// Add shared members with roles
	h.setProjectMembers(&project, req, userID)

	// Reload with associations
	h.db.Preload("SharedWith").Preload("Members").Preload("Members.User").First(&project, project.ID)

	log.Printf("Project created: ID=%d, Name=%s, Budget=%.2f, Members=%d", project.ID, project.Name, project.Budget, len(project.Members))
	c.JSON(http.StatusCreated, struct {
		models.Project
		Members []ProjectMemberResponse `json:"members"`
	}{
		Project: project,
		Members: h.buildMembersResponse(project),
	})
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
		Preload("Members").
		Preload("Members.User").
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
		Stats   ProjectStatsResponse    `json:"stats"`
		Members []ProjectMemberResponse `json:"members"`
	}{
		Project: project,
		Stats:   stats,
		Members: h.buildMembersResponse(project),
	}

	c.JSON(http.StatusOK, response)
}

// Update - PUT /api/v1/projects/:id (creator or owner role)
func (h *ProjectHandler) Update(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	projectID := c.Param("id")

	project, canManage := h.canManageProject(projectID, userID)
	if !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only project owners can edit"})
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

	// Replace members with roles
	h.setProjectMembers(&project, req, project.UserID)

	// Reload with associations
	h.db.Preload("SharedWith").Preload("Members").Preload("Members.User").First(&project, project.ID)

	log.Printf("Project updated: ID=%d, Name=%s", project.ID, project.Name)
	c.JSON(http.StatusOK, struct {
		models.Project
		Members []ProjectMemberResponse `json:"members"`
	}{
		Project: project,
		Members: h.buildMembersResponse(project),
	})
}

// Delete - DELETE /api/v1/projects/:id (creator or owner role)
func (h *ProjectHandler) Delete(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	projectID := c.Param("id")

	project, canManage := h.canManageProject(projectID, userID)
	if !canManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only project owners can delete"})
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

	// Clear members before deleting
	h.db.Where("project_id = ?", project.ID).Delete(&models.ProjectMember{})
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
