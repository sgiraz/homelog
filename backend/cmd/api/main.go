package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/handlers"
	"github.com/sgiraz/homelog/internal/middleware"
)

// appVersion is set at build time via -ldflags "-X main.appVersion=..."
var appVersion = "dev"

var startTime = time.Now()

// Cached GitHub release check
var (
	cachedLatest    string
	cachedLatestURL string
	cachedAt        time.Time
	cacheMu         sync.Mutex
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Validate required environment variables
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("❌ JWT_SECRET environment variable is required. Set it in .env or as a system variable.")
	}

	// Initialize database
	db, err := database.InitDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Auto-migrate models
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed/migrate default categories and their subcategories
	if err := database.SeedDefaultCategories(db); err != nil {
		log.Printf("Warning: failed to seed default categories: %v", err)
	}
	if err := database.MigrateDefaultSubcategories(db); err != nil {
		log.Printf("Warning: failed to migrate default subcategories: %v", err)
	}

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.Default()

	// Apply middleware
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimiter())
	router.Use(middleware.Logger())
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// Security headers (replaces nginx headers)
	router.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "homelog-api",
			"version": appVersion,
			"uptime":  time.Since(startTime).String(),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Version check (public, no auth required)
		v1.GET("/version", func(c *gin.Context) {
			latest, latestURL := getLatestRelease()
			updateAvailable := false
			if latest != "" && appVersion != "dev" && isVersionNewer(latest, appVersion) {
				updateAvailable = true
			}
			resp := gin.H{
				"current":          appVersion,
				"update_available": updateAvailable,
			}
			if latest != "" {
				resp["latest"] = latest
				resp["latest_url"] = latestURL
			}
			c.JSON(200, resp)
		})

		// Public routes
		auth := v1.Group("/auth")
		{
			authHandler := handlers.NewAuthHandler(db)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(middleware.AuthRequired())
		{
			// Exchange rates
			exchangeHandler := handlers.NewExchangeHandler()
			protected.GET("/exchange-rate", exchangeHandler.GetRate)

			// Properties
			properties := protected.Group("/properties")
			{
				propHandler := handlers.NewPropertyHandler(db)
				properties.GET("", propHandler.List)
				properties.POST("", propHandler.Create)
				properties.GET("/:id", propHandler.Get)
				properties.PUT("/:id", propHandler.Update)
				properties.DELETE("/:id", propHandler.Delete)
			}

			// Categories
			categories := protected.Group("/categories")
			{
				catHandler := handlers.NewCategoryHandler(db)
				categories.GET("", catHandler.List)
				categories.POST("", catHandler.Create)
				categories.GET("/:id", catHandler.Get)
				categories.PUT("/:id", catHandler.Update)
				categories.DELETE("/:id", catHandler.Delete)
				categories.POST("/:id/subcategories", catHandler.CreateSubcategory)
				categories.DELETE("/:id/subcategories/:subId", catHandler.DeleteSubcategory)
			}

			// Expenses
			expenses := protected.Group("/expenses")
			{
				expHandler := handlers.NewExpenseHandler(db)
				expenses.GET("", expHandler.List)
				expenses.POST("", expHandler.Create)
				expenses.GET("/:id", expHandler.Get)
				expenses.PUT("/:id", expHandler.Update)
				expenses.DELETE("/:id", expHandler.Delete)
				expenses.GET("/stats", expHandler.GetStats)
			}

			// Centralized communications (across all utilities)
			utilHandler := handlers.NewUtilityHandler(db)

			// Domiciliation sweep: auto-pay due installments for domiciled services.
			// Runs once at startup, then every hour in the background.
			go func() {
				utilHandler.RunDomiciliationSweep()
				ticker := time.NewTicker(1 * time.Hour)
				defer ticker.Stop()
				for range ticker.C {
					utilHandler.RunDomiciliationSweep()
				}
			}()
			protected.GET("/communications", utilHandler.GetAllCommunications)
			protected.GET("/communications/unread-count", utilHandler.GetUnreadCount)
			protected.DELETE("/communications/read", utilHandler.DeleteReadCommunications)

			// Generic notifications (join requests, shared expenses, etc.)
			notifHandler := handlers.NewNotificationHandler(db)
			notifGroup := protected.Group("/notifications")
			{
				notifGroup.GET("", notifHandler.List)
				notifGroup.GET("/unread-count", notifHandler.GetUnreadCount)
				notifGroup.PATCH("/:id/read", notifHandler.MarkRead)
				notifGroup.POST("/read-all", notifHandler.MarkAllRead)
				notifGroup.DELETE("/:id", notifHandler.Delete)
				notifGroup.DELETE("/read", notifHandler.DeleteAllRead)
			}

			// Utilities
			utilities := protected.Group("/utilities")
			{
				pdfHandler := handlers.NewPDFHandler(db)

				utilities.GET("", utilHandler.List)
				utilities.POST("", utilHandler.Create)
				utilities.GET("/:id", utilHandler.Get)
				utilities.PUT("/:id", utilHandler.Update)
				utilities.DELETE("/:id", utilHandler.Delete)

				// Meter readings
				utilities.POST("/:id/readings", utilHandler.AddReading)
				utilities.GET("/:id/readings", utilHandler.GetReadings)
				utilities.PUT("/:id/readings/:readingId", utilHandler.UpdateReading)
				utilities.DELETE("/:id/readings/:readingId", utilHandler.DeleteReading)

				// Bills
				utilities.POST("/:id/bills", utilHandler.AddBill)
				utilities.GET("/:id/bills", utilHandler.GetBills)
				utilities.PUT("/:id/bills/:billId", utilHandler.UpdateBill)
				utilities.PUT("/:id/bills/:billId/full", utilHandler.UpdateBillFull)
				utilities.DELETE("/:id/bills/:billId", utilHandler.DeleteBill)
				utilities.PATCH("/:id/bills/:billId/installments/:instId", utilHandler.UpdateBillInstallment)

				// PDF upload for bills
				utilities.POST("/:id/bills/upload", pdfHandler.UploadBillPDF)

				// Reading comparison (autolettura vs lettura fornitore)
				utilities.GET("/:id/compare-readings", utilHandler.CompareReadings)

				// Communications
				utilities.GET("/:id/communications", utilHandler.GetCommunications)
				utilities.POST("/:id/communications", utilHandler.AddCommunication)
				utilities.PUT("/:id/communications/:commId/read", utilHandler.MarkCommunicationRead)
				utilities.DELETE("/:id/communications/:commId", utilHandler.DeleteCommunication)

				// Contract upload (for creating new utilities)
				utilities.POST("/contract/upload", pdfHandler.UploadContractPDF)
			}

			// Bill extraction templates
			templates := protected.Group("/templates")
			{
				pdfHandler := handlers.NewPDFHandler(db)
				templates.GET("/bills", pdfHandler.ListBillTemplates)
				templates.POST("/bills", pdfHandler.CreateBillTemplate)
				templates.PUT("/bills/:id", pdfHandler.UpdateBillTemplate)
				templates.DELETE("/bills/:id", pdfHandler.DeleteBillTemplate)
			}

			// PDF text extraction helper
			pdf := protected.Group("/pdf")
			{
				pdfHandler := handlers.NewPDFHandler(db)
				pdf.POST("/extract-text", pdfHandler.GetPDFRawText)
				pdf.POST("/analyze", pdfHandler.AnalyzePDFForTemplate)
				pdf.DELETE("/cleanup/:timestamp", pdfHandler.CleanupTemplateImages)
			}

			// Projects
			projects := protected.Group("/projects")
			{
				projHandler := handlers.NewProjectHandler(db)
				projects.GET("", projHandler.List)
				projects.POST("", projHandler.Create)
				projects.GET("/:id", projHandler.Get)
				projects.PUT("/:id", projHandler.Update)
				projects.DELETE("/:id", projHandler.Delete)
			}

			// Expense templates
			expTplHandler := handlers.NewExpenseTemplateHandler(db)
			expTemplates := protected.Group("/expense-templates")
			{
				expTemplates.GET("", expTplHandler.List)
				expTemplates.POST("", expTplHandler.Create)
				expTemplates.PUT("/:id", expTplHandler.Update)
				expTemplates.DELETE("/:id", expTplHandler.Delete)
			}

			// User settings
			settingsHandler := handlers.NewSettingsHandler(db)
			settings := protected.Group("/settings")
			{
				settings.GET("", settingsHandler.Get)
				settings.PUT("", settingsHandler.Update)
				settings.PUT("/password", handlers.NewAuthHandler(db).ChangePassword)
				settings.POST("/avatar", settingsHandler.UploadAvatar)
				settings.DELETE("/avatar", settingsHandler.DeleteAvatar)
				settings.GET("/account/delete-check", settingsHandler.DeleteAccountCheck)
				settings.POST("/account/promote-admin", settingsHandler.PromoteAdmin)
				settings.DELETE("/account", settingsHandler.DeleteAccount)
			}

			// Balance (per property) - nested under properties
			balanceHandler := handlers.NewBalanceHandler(db)
			properties.GET("/:id/balance", balanceHandler.GetBalance)
			properties.GET("/:id/balance/details", balanceHandler.GetBalanceDetails)

			// Household settings (per property) - nested under properties
			properties.GET("/:id/settings", settingsHandler.GetHouseholdSettings)
			properties.PUT("/:id/settings", settingsHandler.UpdateHouseholdSettings)

			// Household members (per property) - nested under properties
			memberHandler := handlers.NewMemberHandler(db)
			properties.GET("/:id/members", memberHandler.List)
			properties.POST("/:id/members", memberHandler.Create)

			// Individual member operations
			members := protected.Group("/members")
			{
				members.GET("/:id", memberHandler.Get)
				members.PUT("/:id", memberHandler.Update)
				members.DELETE("/:id", memberHandler.Delete)
			}

			// Join Requests
			joinRequestHandler := handlers.NewJoinRequestHandler(db)
			protected.POST("/join-requests", joinRequestHandler.Create)
			protected.GET("/join-requests", joinRequestHandler.List)
			protected.PATCH("/join-requests/:id", joinRequestHandler.Resolve)
			protected.GET("/properties/joinable", joinRequestHandler.ListJoinable)

			// Settlements
			settlements := protected.Group("/settlements")
			{
				settlementHandler := handlers.NewSettlementHandler(db)
				settlements.GET("", settlementHandler.List)
				settlements.POST("", settlementHandler.Create)
				settlements.GET("/:id", settlementHandler.Get)
				settlements.DELETE("/:id", settlementHandler.Delete)
			}

			// Admin operations (require admin role)
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminRequired())
			{
				adminHandler := handlers.NewAdminHandler(db)
				admin.DELETE("/users/:id", adminHandler.DeleteUser)
				admin.PUT("/users/:id/role", adminHandler.ToggleAdmin)
			}

			// Export / Import
			exportHandler := handlers.NewExportHandler(db)
			protected.GET("/export/all", exportHandler.ExportAll)
			protected.GET("/export/expenses", exportHandler.ExportExpenses)
			protected.GET("/export/utilities", exportHandler.ExportUtilities)
			protected.GET("/export/projects", exportHandler.ExportProjects)
			protected.POST("/import", exportHandler.ImportData)
		}
	}

	// Serve uploaded files — derive paths from DB_PATH for dev/prod consistency
	baseDataDir := "./data"
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		baseDataDir = filepath.Dir(dbPath)
	}
	router.Static("/uploads", filepath.Join(baseDataDir, "uploads"))
	router.Static("/avatars", filepath.Join(baseDataDir, "avatars"))

	// Serve embedded frontend (SPA + static assets)
	serveFrontend(router)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🏠 HomeLog API v%s starting on port %s...", appVersion, port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func normalizeVersionTag(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "refs/tags/")
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

func parseCalVer(v string) ([4]int, bool) {
	var out [4]int
	parts := strings.Split(normalizeVersionTag(v), ".")
	if len(parts) != 3 && len(parts) != 4 {
		return out, false
	}
	for i, p := range parts {
		if p == "" {
			return out, false
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// isVersionNewer returns true only when latest is strictly newer than current.
func isVersionNewer(latest, current string) bool {
	l, lok := parseCalVer(latest)
	c, cok := parseCalVer(current)
	if !lok || !cok {
		// Unsupported format: don't report a newer version to avoid false positives.
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// getLatestRelease fetches the latest release tag from GitHub, cached for 1 hour.
func getLatestRelease() (tag string, url string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if cachedLatest != "" && time.Since(cachedAt) < time.Hour {
		return cachedLatest, cachedLatestURL
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/sgiraz/homelog/releases/latest")
	if err != nil {
		log.Printf("⚠️  Failed to check GitHub releases: %v", err)
		return cachedLatest, cachedLatestURL
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return cachedLatest, cachedLatestURL
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return cachedLatest, cachedLatestURL
	}

	cachedLatest = release.TagName
	cachedLatestURL = release.HTMLURL
	cachedAt = time.Now()
	return cachedLatest, cachedLatestURL
}
