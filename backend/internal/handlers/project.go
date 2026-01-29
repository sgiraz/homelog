package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	db *gorm.DB
}

func NewProjectHandler(db *gorm.DB) *ProjectHandler {
	return &ProjectHandler{db: db}
}

func (h *ProjectHandler) List(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *ProjectHandler) Create(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *ProjectHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}
