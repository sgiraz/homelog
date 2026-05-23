package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Demo mode not enabled"})
		return
	}
	if err := database.ResetDemoData(h.db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset demo data"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Demo data reset"})
}
