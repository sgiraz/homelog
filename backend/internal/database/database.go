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

// InitDatabase initializes the SQLite database connection
func InitDatabase() (*gorm.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/homelog.db"
	}

	// Derive data directory from DB_PATH for consistency across dev/prod
	dataDir := filepath.Dir(dbPath)
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

	// Data migration: set default role for existing project members
	db.Exec(`UPDATE project_members SET role = 'member' WHERE role IS NULL OR role = ''`)

	// Data migration: mark existing metered services. Guarded, because this runs
	// on every start: unguarded it would discard an explicit is_metered=false,
	// which a metered service is allowed to carry (a flat-rate water contract).
	runOnce(db, "2026-08-utility-metered-classification", func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE utilities SET is_metered = 1 WHERE type IN ('electricity', 'gas', 'water')`).Error
	})

	// Not guarded, unlike the backfill above: for a fixed-cost type is_metered=0
	// is an invariant rather than a user choice — there is no meter on a mortgage
	// or a TARI bill, so re-asserting it every boot cannot discard anything the
	// user picked. It self-heals rows written by a build that still took
	// is_metered from the client (see handlers.UtilityHandler.Create).
	db.Exec(`UPDATE utilities SET is_metered = 0 WHERE type IN ? AND is_metered = 1`, models.FixedCostTypes())
	// Default billing frequency for existing utilities
	db.Exec(`UPDATE utilities SET billing_interval = 1 WHERE billing_interval IS NULL OR billing_interval = 0`)
	db.Exec(`UPDATE utilities SET billing_unit = 'month' WHERE billing_unit IS NULL OR billing_unit = ''`)

	// Data migration: backfill SettledAmount for splits settled before the
	// partial-settlement ledger existed (SettledAmount added as a new column,
	// defaults to 0 for all pre-existing rows).
	db.Exec(`UPDATE expense_splits SET settled_amount = amount WHERE is_settled = 1 AND settled_amount = 0`)
	// Backfill settlement_allocations for legacy fully-settled splits so a
	// future partial reversal of an old settlement has ledger detail to work
	// with instead of an orphaned SettlementID. Idempotent: skips splits that
	// already have an allocation row.
	db.Exec(`INSERT INTO settlement_allocations (settlement_id, expense_split_id, amount, created_at)
		SELECT settlement_id, id, amount, COALESCE(settled_at, CURRENT_TIMESTAMP)
		FROM expense_splits
		WHERE is_settled = 1 AND settlement_id IS NOT NULL
		AND id NOT IN (SELECT expense_split_id FROM settlement_allocations)`)

	// Data migration: ensure property owners have admin role on their HouseholdMember
	// This backfills existing databases where HouseholdMember.Role was not set to "admin"
	db.Exec(`UPDATE household_members SET role = 'admin'
		WHERE user_id IS NOT NULL AND role != 'admin'
		AND EXISTS (
			SELECT 1 FROM properties WHERE properties.id = household_members.property_id
			AND properties.user_id = household_members.user_id
		)`)
	// Ensure non-owner members have at least 'member' role
	db.Exec(`UPDATE household_members SET role = 'member' WHERE (role IS NULL OR role = '') AND user_id IS NOT NULL`)

	// Data migration: mark onboarding as completed for existing users who already have a property
	db.Exec(`UPDATE user_settings SET onboarding_completed = 1
		WHERE onboarding_completed = 0
		AND user_id IN (
			SELECT DISTINCT user_id FROM household_members WHERE user_id IS NOT NULL
		)`)

	// Data migration: backfill 1 installment per existing bill (one-time).
	// After this migration, every Bill has at least one BillInstallment mirroring its
	// due_date/amount/is_paid, so the payment flow can treat all bills uniformly.
	var biCount int64
	db.Model(&models.BillInstallment{}).Count(&biCount)
	if biCount == 0 {
		db.Exec(`INSERT INTO bill_installments
			(created_at, updated_at, bill_id, number, due_date, amount, is_paid, paid_at, expense_id)
			SELECT b.created_at, b.updated_at, b.id, 1, b.due_date, b.amount_total, b.is_paid, b.paid_date,
			       (SELECT id FROM expenses WHERE expenses.bill_id = b.id AND expenses.deleted_at IS NULL LIMIT 1)
			FROM bills b
			WHERE b.deleted_at IS NULL`)
	}

	// Data migration: fix payer splits for bill-originated expenses.
	// Earlier versions of autoCreateExpenseFromBill created the payer's own split with
	// is_settled=false, which blocked the "fully settled" status after a Settlement run
	// (the payer split is excluded from Salda queries by design).
	db.Exec(`UPDATE expense_splits
		SET is_settled = 1, settled_at = COALESCE(settled_at, CURRENT_TIMESTAMP)
		WHERE is_settled = 0
		AND expense_id IN (
			SELECT id FROM expenses WHERE bill_id IS NOT NULL
		)
		AND member_id IN (
			SELECT paid_by_member_id FROM expenses
			WHERE expenses.id = expense_splits.expense_id
			AND expenses.paid_by_member_id IS NOT NULL
		)`)

	// Backfill price changes from existing bills of fixed-cost services (one-time)
	var pcCount int64
	db.Model(&models.PriceChange{}).Count(&pcCount)
	if pcCount == 0 {
		backfillPriceChanges(db)
	}

	// Global search: FTS5 virtual table + one-time backfill from existing rows.
	// GORM hooks (see models) keep it in sync on subsequent writes.
	if err := initSearchIndex(db); err != nil {
		return fmt.Errorf("failed to init search index: %w", err)
	}
	backfillSearchIndex(db)

	log.Println("✅ Database migrations completed")
	return nil
}

// runOnce runs fn the first time it is called for key, then records key in
// applied_migrations so later boots skip it. Data fixes that overwrite a value
// the user can set belong here: replayed on every start they silently revert
// the user's choice. Failures are logged, not fatal — the key stays unrecorded
// so the next start retries.
func runOnce(db *gorm.DB, key string, fn func(tx *gorm.DB) error) {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS applied_migrations (
		key TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error; err != nil {
		log.Printf("⚠️  migration %s skipped: %v", key, err)
		return
	}

	var applied int64
	if err := db.Raw(`SELECT COUNT(*) FROM applied_migrations WHERE key = ?`, key).Scan(&applied).Error; err != nil {
		log.Printf("⚠️  migration %s skipped: %v", key, err)
		return
	}
	if applied > 0 {
		return
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := fn(tx); err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO applied_migrations (key) VALUES (?)`, key).Error
	}); err != nil {
		log.Printf("⚠️  migration %s failed: %v", key, err)
		return
	}
	log.Printf("✅ migration %s applied", key)
}

// backfillPriceChanges scans existing bills of fixed-cost services and creates
// PriceChange records for any consecutive bills with different amounts.
func backfillPriceChanges(db *gorm.DB) {
	var utilities []models.Utility
	db.Where("is_metered = ?", false).Find(&utilities)

	for _, u := range utilities {
		var bills []models.Bill
		db.Where("utility_id = ?", u.ID).Order("period_start ASC").Find(&bills)

		for i := 1; i < len(bills); i++ {
			if bills[i].AmountTotal != bills[i-1].AmountTotal {
				// Check if this exact change already exists (idempotent)
				var existing models.PriceChange
				if db.Where("utility_id = ? AND source_bill_id = ?", u.ID, bills[i].ID).
					First(&existing).Error == nil {
					continue
				}
				change := models.PriceChange{
					UtilityID:     u.ID,
					EffectiveDate: bills[i].PeriodStart,
					OldAmount:     bills[i-1].AmountTotal,
					NewAmount:     bills[i].AmountTotal,
					SourceBillID:  &bills[i].ID,
				}
				db.Create(&change)
			}
		}

		// Update recurring_amount to the latest bill's amount
		if len(bills) > 0 {
			db.Model(&u).Update("recurring_amount", bills[len(bills)-1].AmountTotal)
		}
	}

	log.Println("✅ Price changes backfill completed")
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
			{CategoryID: catID, Name: "Salute"},
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
