package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/handlers"
	"github.com/sgiraz/homelog/internal/middleware"
)

var startTime = time.Now()

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
			"version": "1.0.0",
			"uptime":  time.Since(startTime).String(),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
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
			// Properties
			properties := protected.Group("/properties")
			{
				propHandler := handlers.NewPropertyHandler(db)
				properties.GET("", propHandler.List)
				properties.POST("", middleware.AdminRequired(), propHandler.Create)
				properties.GET("/:id", propHandler.Get)
				properties.PUT("/:id", middleware.AdminRequired(), propHandler.Update)
				properties.DELETE("/:id", middleware.AdminRequired(), propHandler.Delete)
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
			protected.GET("/communications", utilHandler.GetAllCommunications)
			protected.GET("/communications/unread-count", utilHandler.GetUnreadCount)

			// Utilities
			utilities := protected.Group("/utilities")
			{
				pdfHandler := handlers.NewPDFHandler(db)

				utilities.GET("", utilHandler.List)
				utilities.POST("", middleware.AdminRequired(), utilHandler.Create)
				utilities.GET("/:id", utilHandler.Get)
				utilities.PUT("/:id", middleware.AdminRequired(), utilHandler.Update)
				utilities.DELETE("/:id", middleware.AdminRequired(), utilHandler.Delete)

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
				utilities.POST("/contract/upload", middleware.AdminRequired(), pdfHandler.UploadContractPDF)
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

			// User settings
			settingsHandler := handlers.NewSettingsHandler(db)
			settings := protected.Group("/settings")
			{
				settings.GET("", settingsHandler.Get)
				settings.PUT("", settingsHandler.Update)
				settings.PUT("/password", handlers.NewAuthHandler(db).ChangePassword)
				settings.POST("/avatar", settingsHandler.UploadAvatar)
				settings.DELETE("/avatar", settingsHandler.DeleteAvatar)
			}

			// Balance (per property) - nested under properties
			balanceHandler := handlers.NewBalanceHandler(db)
			properties.GET("/:id/balance", balanceHandler.GetBalance)
			properties.GET("/:id/balance/details", balanceHandler.GetBalanceDetails)

			// Household settings (per property) - nested under properties
			properties.GET("/:id/settings", settingsHandler.GetHouseholdSettings)
			properties.PUT("/:id/settings", middleware.AdminRequired(), settingsHandler.UpdateHouseholdSettings)

			// Household members (per property) - nested under properties
			memberHandler := handlers.NewMemberHandler(db)
			properties.GET("/:id/members", memberHandler.List)
			properties.POST("/:id/members", middleware.AdminRequired(), memberHandler.Create)

			// Individual member operations
			members := protected.Group("/members")
			{
				members.GET("/:id", memberHandler.Get)
				members.PUT("/:id", middleware.AdminRequired(), memberHandler.Update)
				members.DELETE("/:id", middleware.AdminRequired(), memberHandler.Delete)
			}

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

	// Serve uploaded files
	router.Static("/uploads", "./data/uploads")
	router.Static("/avatars", "./data/avatars")

	// Serve embedded frontend (SPA + static assets)
	serveFrontend(router)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🏠 HomeLog API starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
