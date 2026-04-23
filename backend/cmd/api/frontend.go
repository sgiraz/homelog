package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:static
var staticFiles embed.FS

// serveFrontend configures the router to serve the embedded Vue SPA.
// Hashed assets get aggressive caching; all other non-API routes fall back to index.html.
func serveFrontend(router *gin.Engine) {
	distFS, _ := fs.Sub(staticFiles, "static")

	fileServer := http.FileServer(http.FS(distFS))

	// Hashed assets: aggressive cache (Vite adds content hashes to filenames)
	router.GET("/assets/*filepath", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// SPA catch-all: serve index.html for non-API, non-file routes
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API and upload routes return 404 JSON
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/uploads/") {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}

		// Try serving the actual static file first (e.g. favicon.ico, manifest.json)
		f, err := fs.Stat(distFS, strings.TrimPrefix(path, "/"))
		if err == nil && !f.IsDir() {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Fallback: serve index.html for SPA client-side routing
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.Status(500)
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})
}
