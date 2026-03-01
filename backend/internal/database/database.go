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
		&models.BillTemplate{},
		&models.ContractTemplate{},
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
		{UserID: nil, Name: "Abbigliamento e Cura Personale", Icon: "👕", Color: "#F97316", IsDefault: true},
		{UserID: nil, Name: "Istruzione", Icon: "🎓", Color: "#6366F1", IsDefault: true},
		{UserID: nil, Name: "Tecnologia", Icon: "📱", Color: "#0EA5E9", IsDefault: true},
		{UserID: nil, Name: "Regali", Icon: "🎁", Color: "#EF4444", IsDefault: true},
	}

	// Create categories with subcategories
	for _, cat := range defaultCategories {
		if err := db.Create(&cat).Error; err != nil {
			return err
		}

		subs := defaultSubcategoriesFor(cat.Name, cat.ID)
		if len(subs) > 0 {
			if err := db.Create(&subs).Error; err != nil {
				return err
			}
		}
	}

	log.Println("✅ Default categories seeded")
	return nil
}

// defaultSubcategoriesFor returns the default subcategories for a given category name.
func defaultSubcategoriesFor(catName string, catID uint) []models.Subcategory {
	switch catName {
	case "Casa":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Utenze"},
			{CategoryID: catID, Name: "Manutenzione ordinaria"},
			{CategoryID: catID, Name: "Lavori straordinari"},
			{CategoryID: catID, Name: "Arredamento"},
			{CategoryID: catID, Name: "Elettrodomestici"},
		}
	case "Alimentari e Ristorazione":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Spesa supermercato"},
			{CategoryID: catID, Name: "Ristoranti/Pizzerie"},
			{CategoryID: catID, Name: "Bar/Caffè"},
			{CategoryID: catID, Name: "Delivery"},
		}
	case "Trasporti":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Carburante"},
			{CategoryID: catID, Name: "Manutenzione auto"},
			{CategoryID: catID, Name: "Assicurazione"},
			{CategoryID: catID, Name: "Parcheggi/Pedaggi"},
			{CategoryID: catID, Name: "Trasporti pubblici"},
		}
	case "Salute":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Farmaci"},
			{CategoryID: catID, Name: "Visite mediche"},
			{CategoryID: catID, Name: "Assicurazioni sanitarie"},
		}
	case "Intrattenimento":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Cinema/Teatro"},
			{CategoryID: catID, Name: "Streaming/Abbonamenti"},
			{CategoryID: catID, Name: "Libri/Riviste"},
			{CategoryID: catID, Name: "Viaggi/Vacanze"},
		}
	case "Famiglia":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Abbigliamento bambini"},
			{CategoryID: catID, Name: "Scuola/Asilo"},
			{CategoryID: catID, Name: "Giochi/Attrezzature"},
		}
	case "Abbigliamento e Cura Personale":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Abbigliamento adulti"},
			{CategoryID: catID, Name: "Parrucchiere/Barbiere"},
			{CategoryID: catID, Name: "Prodotti cura personale"},
		}
	case "Istruzione":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Corsi/Formazione"},
			{CategoryID: catID, Name: "Libri di testo"},
			{CategoryID: catID, Name: "Materiale scolastico"},
		}
	case "Tecnologia":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Dispositivi"},
			{CategoryID: catID, Name: "Abbonamenti software"},
			{CategoryID: catID, Name: "Riparazioni"},
		}
	case "Regali":
		return []models.Subcategory{
			{CategoryID: catID, Name: "Compleanni"},
			{CategoryID: catID, Name: "Natale"},
			{CategoryID: catID, Name: "Anniversari"},
			{CategoryID: catID, Name: "San Valentino"},
			{CategoryID: catID, Name: "Lauree/Diplomi"},
			{CategoryID: catID, Name: "Nascite/Battesimi"},
		}
	}
	return nil
}

// MigrateDefaultSubcategories adds missing subcategories to existing default categories.
// Safe to call at every startup — it only inserts what's missing.
func MigrateDefaultSubcategories(db *gorm.DB) error {
	var categories []models.Category
	if err := db.Where("is_default = true").Preload("Subcategories").Find(&categories).Error; err != nil {
		return err
	}

	for _, cat := range categories {
		expected := defaultSubcategoriesFor(cat.Name, cat.ID)
		if len(expected) == 0 {
			continue
		}

		existingNames := make(map[string]bool)
		for _, s := range cat.Subcategories {
			existingNames[s.Name] = true
		}

		for _, sub := range expected {
			if !existingNames[sub.Name] {
				if err := db.Create(&sub).Error; err != nil {
					return err
				}
			}
		}
	}
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
