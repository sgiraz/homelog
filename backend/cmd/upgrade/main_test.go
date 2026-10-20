package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/models"
)

// legacyDB returns a database shaped like 2026.09.11 left it: templates owned
// by their author, estimates kept apart, a default template per service.
func legacyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "homelog.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, db)
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DROP TABLE bill_templates`,
		"CREATE TABLE `bill_templates` (`id` integer PRIMARY KEY AUTOINCREMENT,`created_at` datetime,`updated_at` datetime," +
			"`user_id` integer NOT NULL,`name` text NOT NULL,`provider` text NOT NULL,`utility_type` text NOT NULL," +
			"`is_default` numeric NOT NULL DEFAULT false,`extraction_rules` text," +
			"CONSTRAINT `fk_bill_templates_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`))",
		`ALTER TABLE bills ADD COLUMN estimated_date datetime`,
		`ALTER TABLE bills ADD COLUMN estimated_reading real`,
		`ALTER TABLE bills ADD COLUMN estimated_consumption real`,
		`ALTER TABLE utilities ADD COLUMN default_bill_template_id integer`,
	} {
		mustExec(t, db, stmt)
	}
	return db
}

// closeOnCleanup closes db before t.TempDir is removed: Windows cannot delete
// an open database file.
func closeOnCleanup(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func mustCreate(t *testing.T, db *gorm.DB, rows ...any) {
	t.Helper()
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create %T: %v", row, err)
		}
	}
}

func TestUpgradeFromTheLastReleaseBeforeTemplatesPerProperty(t *testing.T) {
	db := legacyDB(t)
	now := time.Now()
	author := &models.User{Email: "author@example.com", PasswordHash: "x", Name: "Author", IsActive: true}
	partner := &models.User{Email: "partner@example.com", PasswordHash: "x", Name: "Partner", IsActive: true}
	lodger := &models.User{Email: "lodger@example.com", PasswordHash: "x", Name: "Lodger", IsActive: true}
	loner := &models.User{Email: "loner@example.com", PasswordHash: "x", Name: "Loner", IsActive: true}
	mustCreate(t, db, author, partner, lodger, loner)
	home := models.Property{UserID: author.ID, Name: "Home", StartDate: now, IsCurrent: true}
	cabin := models.Property{UserID: author.ID, Name: "Cabin", StartDate: now}
	flat := models.Property{UserID: lodger.ID, Name: "Flat", StartDate: now}
	mustCreate(t, db, &cabin, &home, &flat)
	mustCreate(t, db,
		&models.HouseholdMember{PropertyID: cabin.ID, UserID: &author.ID, Name: "Author"},
		&models.HouseholdMember{PropertyID: home.ID, UserID: &author.ID, Name: "Author"},
		&models.HouseholdMember{PropertyID: home.ID, UserID: &partner.ID, Name: "Partner"},
		&models.HouseholdMember{PropertyID: flat.ID, UserID: &lodger.ID, Name: "Lodger"},
	)

	insert := func(userID uint, provider, utilityType string, isDefault bool, rules string) uint {
		t.Helper()
		mustExec(t, db, `INSERT INTO bill_templates (created_at, updated_at, user_id, name, provider, utility_type, is_default, extraction_rules)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, now, now, userID, provider, provider, utilityType, isDefault, rules)
		var id uint
		db.Raw(`SELECT MAX(id) FROM bill_templates`).Scan(&id)
		return id
	}
	byDefault := insert(author.ID, "Aqua", "water", false, `[]`)
	byProvider := insert(author.ID, "  VOLTA ", "electricity", false, `[]`)
	unmatched := insert(author.ID, "Nobody", "gas", false, `[]`)
	firstHousehold := insert(lodger.ID, "Acme", "water", false, `[]`)
	orphan := insert(loner.ID, "Acme", "water", false, `[]`)
	shared := insert(author.ID, "Fibernet", "internet", false, `[]`)
	olderDefault := insert(author.ID, "Flame", "gas", true, `[]`)
	newerDefault := insert(partner.ID, "flame", "gas", true,
		`[{"field":"provider_reading","anchor_text":"Current"},{"field":"estimated_reading","anchor_text":"Estimate"},{"field":"estimated_date"}]`)

	cabinWater := models.Utility{UserID: author.ID, PropertyID: cabin.ID, Type: "water", Provider: "Other"}
	mustCreate(t, db, &cabinWater,
		&models.Utility{UserID: author.ID, PropertyID: cabin.ID, Type: "electricity", Provider: "Volta"},
		&models.Utility{UserID: author.ID, PropertyID: home.ID, Type: "internet", Provider: "Fibernet"},
		&models.Utility{UserID: author.ID, PropertyID: cabin.ID, Type: "internet", Provider: "Fibernet"},
	)
	mustExec(t, db, `UPDATE utilities SET default_bill_template_id = ? WHERE id = ?`, byDefault, cabinWater.ID)
	mustExec(t, db, `UPDATE utilities SET default_bill_template_id = ? WHERE type = 'internet'`, shared)

	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	value := func(v float64) *float64 { return &v }
	estimated := models.Bill{UtilityID: cabinWater.ID, BillNumber: "E", IssueDate: day(6, 20), PeriodStart: day(1, 4), PeriodEnd: day(6, 16),
		DueDate: day(7, 10), ProviderReading: value(119), ReadingType: "actual", ConsumptionTotal: 53}
	actual := models.Bill{UtilityID: cabinWater.ID, BillNumber: "A", IssueDate: day(10, 9), PeriodStart: day(5, 29), PeriodEnd: day(10, 7),
		DueDate: day(11, 1), ProviderReading: value(145), ProviderReadingDate: &[]time.Time{day(10, 1)}[0], ReadingType: "actual"}
	mustCreate(t, db, &estimated, &actual)
	mustExec(t, db, `UPDATE bills SET estimated_date = ?, estimated_reading = 122, estimated_consumption = 3 WHERE id = ?`, day(6, 16), estimated.ID)

	if err := upgrade(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if steps := database.PendingUpgrades(db); len(steps) != 0 {
		t.Fatalf("still pending after the upgrade: %v", steps)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("server migration after the upgrade: %v", err)
	}

	propertyOf := func(id uint) uint {
		t.Helper()
		var tpl models.BillTemplate
		if err := db.First(&tpl, id).Error; err != nil {
			t.Fatalf("template %d: %v", id, err)
		}
		return tpl.PropertyID
	}
	for name, c := range map[string]struct{ id, want uint }{
		"default of a service":     {byDefault, cabin.ID},
		"provider match":           {byProvider, cabin.ID},
		"current property":         {unmatched, home.ID},
		"first household":          {firstHousehold, flat.ID},
		"default of first service": {shared, home.ID},
	} {
		if got := propertyOf(c.id); got != c.want {
			t.Errorf("%s: property %d, want %d", name, got, c.want)
		}
	}
	var count int64
	db.Model(&models.BillTemplate{}).Where("id = ?", orphan).Count(&count)
	if count != 0 {
		t.Error("a template no household can reach survived")
	}
	db.Model(&models.BillTemplate{}).Where("property_id = ? AND provider = ?", cabin.ID, "Fibernet").Count(&count)
	if count != 1 {
		t.Errorf("the cabin's internet service should get its own copy of the shared template, found %d", count)
	}
	var older, newer models.BillTemplate
	db.First(&older, olderDefault)
	db.First(&newer, newerDefault)
	if older.IsDefault || !newer.IsDefault {
		t.Errorf("one default per property+provider+type: older=%v newer=%v", older.IsDefault, newer.IsDefault)
	}
	if want := `[{"anchor_text":"Estimate","field":"provider_reading"},{"field":"provider_reading_date"}]`; newer.ExtractionRules != want {
		t.Errorf("template rules = %s, want the estimate rules as the closing reading: %s", newer.ExtractionRules, want)
	}

	var bills []models.Bill
	db.Order("id").Find(&bills)
	if e := bills[0]; *e.ProviderReading != 122 || e.ReadingType != "estimated" || !e.ProviderReadingDate.Equal(day(6, 16)) || e.ConsumptionTotal != 53 {
		t.Errorf("estimated bill = reading %v on %v (%s), consumption %v; want 122 on 16 Jun (estimated), 53",
			*e.ProviderReading, e.ProviderReadingDate, e.ReadingType, e.ConsumptionTotal)
	}
	if a := bills[1]; *a.ProviderReading != 145 || a.ReadingType != "actual" || !a.ProviderReadingDate.Equal(day(10, 1)) {
		t.Errorf("a bill without an estimate must be left alone: %+v", a)
	}

	db.Model(&models.BillTemplate{}).Count(&count)
	before := count
	if err := upgrade(db); err != nil {
		t.Fatalf("second run: %v", err)
	}
	db.Model(&models.BillTemplate{}).Count(&count)
	if count != before {
		t.Errorf("a second run changed the template count: %d -> %d", before, count)
	}
}

// A current database has nothing to convert, and the tool leaves it alone.
func TestUpgradeLeavesACurrentDatabaseAlone(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "homelog.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, db)
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if steps := database.PendingUpgrades(db); len(steps) != 0 {
		t.Fatalf("a current database reports %v", steps)
	}
	if err := upgrade(db); err != nil {
		t.Fatal(err)
	}
}
