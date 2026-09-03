package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/database"
)

// DemoHandler exposes demo-only operations. Its routes are registered only when
// DEMO_MODE is enabled (see main.go).
type DemoHandler struct {
	db *gorm.DB
}

func NewDemoHandler(db *gorm.DB) *DemoHandler {
	return &DemoHandler{db: db}
}

// Reset rebuilds the demo dataset on demand, backing the "Reset demo data"
// button in the UI. Available to any authenticated visitor; in demo mode the
// only data that exists is throwaway, so there is nothing to protect.
func (h *DemoHandler) Reset(c *gin.Context) {
	if !database.IsDemoMode() {
		apierr.Fail(c, http.StatusForbidden, "demo_mode_disabled", "Demo mode is not enabled on this instance")
		return
	}
	if err := database.ResetDemoData(h.db); err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to reset demo data")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Demo data reset"})
}
