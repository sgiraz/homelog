package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// HouseholdHandler serves the households a user belongs to.
type HouseholdHandler struct {
	db *gorm.DB
}

// NewHouseholdHandler creates a new household handler
func NewHouseholdHandler(db *gorm.DB) *HouseholdHandler {
	return &HouseholdHandler{db: db}
}

// HouseholdSummary is one household of the current user.
type HouseholdSummary struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"is_admin"`
	// PrimaryHomeID is the household's primary home, the default for its Home
	// expenses; 0 only while the household is being created.
	PrimaryHomeID uint `json:"primary_home_id"`
}

// List - GET /api/v1/households
// The households the user is a member of, oldest first.
func (h *HouseholdHandler) List(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var rows []struct {
		ID   uint
		Name string
		Role string
	}
	if err := h.db.Model(&models.Household{}).
		Select("households.id, households.name, household_members.role").
		Joins("JOIN household_members ON household_members.household_id = households.id").
		Where("household_members.user_id = ?", userID).
		Order("households.id").
		Scan(&rows).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch households")
		return
	}

	ids := make([]uint, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	var primaries []models.Property
	if len(ids) > 0 {
		if err := h.db.Select("id", "household_id").
			Where("household_id IN ? AND is_current = ?", ids, true).
			Find(&primaries).Error; err != nil {
			apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch households")
			return
		}
	}
	primaryOf := make(map[uint]uint, len(primaries))
	for _, p := range primaries {
		primaryOf[p.HouseholdID] = p.ID
	}

	out := make([]HouseholdSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, HouseholdSummary{
			ID:            r.ID,
			Name:          r.Name,
			IsAdmin:       r.Role == "admin",
			PrimaryHomeID: primaryOf[r.ID],
		})
	}
	c.JSON(http.StatusOK, out)
}
