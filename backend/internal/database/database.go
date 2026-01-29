package database

import (
	"fmt"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDatabase initializes the SQLite database connection
func InitDatabase() (*gorm.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/homelog.db"
	}

	// Create data directory if it doesn't exist
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	// Create uploads directory
	uploadsDir := "./data/uploads"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadsDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create uploads directory: %w", err)
		}
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

// AutoMigrate runs automatic migrations for all models
func AutoMigrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")

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
		&models.UtilityRate{},
		&models.Project{},
		&models.UserSettings{},
		&models.HouseholdSettings{},
		&models.ExpenseSplit{},
		&models.Settlement{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("✅ Database migrations completed")
	return nil
}

// SeedDefaultCategories seeds global default categories (called once for first user)
func SeedDefaultCategories(db *gorm.DB) error {
	// Check if categories already exist
	var count int64
	db.Model(&models.Category{}).Count(&count)
	if count > 0 {
		return nil // Categories already exist
	}

	log.Println("🔄 Seeding default categories...")

	// Default categories (global, no user_id)
	defaultCategories := []models.Category{
		{UserID: nil, Name: "Casa", Icon: "🏠", Color: "#3B82F6", IsDefault: true},
		{UserID: nil, Name: "Alimentari e Ristorazione", Icon: "🍕", Color: "#10B981", IsDefault: true},
		{UserID: nil, Name: "Trasporti", Icon: "🚗", Color: "#F59E0B", IsDefault: true},
		{UserID: nil, Name: "Salute", Icon: "🏥", Color: "#8B5CF6", IsDefault: true},
		{UserID: nil, Name: "Intrattenimento", Icon: "🎬", Color: "#EC4899", IsDefault: true},
		{UserID: nil, Name: "Famiglia", Icon: "👶", Color: "#14B8A6", IsDefault: true},
		{UserID: nil, Name: "Abbigliamento", Icon: "👕", Color: "#F97316", IsDefault: true},
		{UserID: nil, Name: "Istruzione", Icon: "🎓", Color: "#6366F1", IsDefault: true},
		{UserID: nil, Name: "Tecnologia", Icon: "📱", Color: "#0EA5E9", IsDefault: true},
	}

	// Create categories with subcategories
	for _, cat := range defaultCategories {
		if err := db.Create(&cat).Error; err != nil {
			return err
		}

		// Add subcategories based on category
		var subcategories []models.Subcategory
		switch cat.Name {
		case "Casa":
			subcategories = []models.Subcategory{
				{CategoryID: cat.ID, Name: "Utenze"},
				{CategoryID: cat.ID, Name: "Manutenzione ordinaria"},
				{CategoryID: cat.ID, Name: "Lavori straordinari"},
				{CategoryID: cat.ID, Name: "Arredamento"},
				{CategoryID: cat.ID, Name: "Elettrodomestici"},
			}
		case "Alimentari e Ristorazione":
			subcategories = []models.Subcategory{
				{CategoryID: cat.ID, Name: "Spesa supermercato"},
				{CategoryID: cat.ID, Name: "Ristoranti/Pizzerie"},
				{CategoryID: cat.ID, Name: "Bar/Caffè"},
				{CategoryID: cat.ID, Name: "Delivery"},
			}
		case "Trasporti":
			subcategories = []models.Subcategory{
				{CategoryID: cat.ID, Name: "Carburante"},
				{CategoryID: cat.ID, Name: "Manutenzione auto"},
				{CategoryID: cat.ID, Name: "Assicurazione"},
				{CategoryID: cat.ID, Name: "Parcheggi/Pedaggi"},
				{CategoryID: cat.ID, Name: "Trasporti pubblici"},
			}
		}

		if len(subcategories) > 0 {
			if err := db.Create(&subcategories).Error; err != nil {
				return err
			}
		}
	}

	log.Println("✅ Default categories seeded")
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
		Language:                  "it",
		Currency:                  "EUR",
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
