package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Path is the SQLite file the app uses: DB_PATH, or ./data/homelog.db.
func Path() string {
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		return dbPath
	}
	return "./data/homelog.db"
}

// DataDir is the directory next to the database, where uploads and avatars
// live. Every path the app writes to or serves files from must start here:
// the PDF handler used to hardcode ./data/uploads while the router served
// DataDir()/uploads, which only coincide when DB_PATH sits in ./data — with
// any other DB_PATH, template pages and bill attachments answered 404.
func DataDir() string {
	return filepath.Dir(Path())
}

// InitDatabase initializes the SQLite database connection
func InitDatabase() (*gorm.DB, error) {
	dbPath := Path()
	dataDir := DataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create uploads directory
	uploadsDir := filepath.Join(dataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create uploads directory: %w", err)
	}

	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	if os.Getenv("GIN_MODE") == "release" {
		config.Logger = logger.Default.LogMode(logger.Error)
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(dbPath), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable WAL mode for better concurrency
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA synchronous=NORMAL;")
	db.Exec("PRAGMA foreign_keys=ON;")

	log.Println("✅ Database connected successfully")
	return db, nil
}

// PendingUpgrades names the conversions a database from HomeLog 2026.09.11 or
// earlier still needs before this version can start on it. cmd/upgrade
// applies them; the server only refuses to start.
func PendingUpgrades(db *gorm.DB) []string {
	m := db.Migrator()
	var steps []string
	if m.HasTable("bill_templates") && !m.HasColumn("bill_templates", "property_id") {
		steps = append(steps, "bill templates still belong to their author")
	}
	if m.HasColumn("bills", "estimated_reading") {
		steps = append(steps, "bills still keep a separate estimate")
	}
	if m.HasColumn("utilities", "default_bill_template_id") {
		steps = append(steps, "services still have a default template")
	}
	return steps
}

// AutoMigrate runs automatic migrations for all models
func AutoMigrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")

	// Disable FK checks during migration — GORM may need to recreate tables
	// to add new foreign keys, which fails if other tables reference them.
	db.Exec("PRAGMA foreign_keys=OFF;")
	defer db.Exec("PRAGMA foreign_keys=ON;")

	// Register custom join table for Project <-> User with role column
	if err := db.SetupJoinTable(&models.Project{}, "SharedWith", &models.ProjectMember{}); err != nil {
		return fmt.Errorf("failed to setup project_members join table: %w", err)
	}

	err := db.AutoMigrate(
		&models.User{},
		&models.Property{},
		&models.HouseholdMember{},
		&models.Category{},
		&models.Subcategory{},
		&models.Expense{},
		&models.Utility{},
		&models.MeterReading{},
		&models.Bill{},
		&models.BillInstallment{},
		&models.UtilityRate{},
		&models.PriceChange{},
		&models.ServiceCommunication{},
		&models.Project{},
		&models.UserSettings{},
		&models.HouseholdSettings{},
		&models.ExpenseSplit{},
		&models.Settlement{},
		&models.SettlementAllocation{},
		&models.BillTemplate{},
		&models.ContractTemplate{},
		&models.ExpenseTemplate{},
		&models.PropertyJoinRequest{},
		&models.Notification{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	db.Exec(`DROP TABLE IF EXISTS applied_migrations`)

	// Global search: FTS5 virtual table + one-time backfill from existing rows.
	// GORM hooks (see models) keep it in sync on subsequent writes.
	if err := initSearchIndex(db); err != nil {
		return fmt.Errorf("failed to init search index: %w", err)
	}
	backfillSearchIndex(db)

	log.Println("✅ Database migrations completed")
	return nil
}

// SeedDefaultData seeds user settings for new users (categories are now global)
func SeedDefaultData(db *gorm.DB, userID uint) error {
	// Check if global categories exist, if not create them
	var count int64
	db.Model(&models.Category{}).Count(&count)
	if count == 0 {
		if err := SeedDefaultCategories(db); err != nil {
			return err
		}
	}

	// Check if user settings already exist
	var settingsCount int64
	db.Model(&models.UserSettings{}).Where("user_id = ?", userID).Count(&settingsCount)
	if settingsCount > 0 {
		return nil // User settings already exist
	}

	// Create default user settings
	settings := models.UserSettings{
		UserID:                    userID,
		Language:                  models.DefaultLanguage,
		Currency:                  models.DefaultCurrency,
		Theme:                     "auto",
		DateFormat:                "DD/MM/YYYY",
		DefaultSplitWithMemberIDs: "",
		EmailNotifications:        true,
		InAppNotifications:        true,
		BillDueAlertDays:          3,
		ReadingReminderDays:       7,
		AnomalyThreshold:          5.0,
	}

	return db.Create(&settings).Error
}
