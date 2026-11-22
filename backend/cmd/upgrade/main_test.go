package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sgiraz/homelog/internal/database"
	"github.com/sgiraz/homelog/internal/handlers"
	"github.com/sgiraz/homelog/internal/models"
)

// beforeHouseholdsDB returns an empty database with the schema of the last
// release before households (testdata/schema-before-households.sql, dumped
// from it): members, settlements and split settings still hang off a home.
func beforeHouseholdsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "homelog.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, db)
	schema, err := os.ReadFile("testdata/schema-before-households.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range strings.Split(strings.ReplaceAll(string(schema), "\r\n", "\n"), ";\n") {
		if stmt = strings.TrimSpace(stmt); stmt != "" {
			mustExec(t, db, stmt)
		}
	}
	// The models' search hooks write here while the tests seed rows; the
	// server rebuilds the index on its next start anyway.
	mustExec(t, db, `CREATE VIRTUAL TABLE search_index USING fts5(entity_type UNINDEXED, entity_id UNINDEXED,
		property_id UNINDEXED, household_id UNINDEXED, user_id UNINDEXED, title, body)`)
	return db
}

// legacyDB returns a database shaped like 2026.09.11 left it: templates owned
// by their author, estimates kept apart, a default template per service.
func legacyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := beforeHouseholdsDB(t)
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

// insertHome adds a home the way the schema before households stores it.
func insertHome(t *testing.T, db *gorm.DB, owner uint, name string, current bool) uint {
	t.Helper()
	now := time.Now()
	mustExec(t, db, `INSERT INTO properties (created_at, updated_at, user_id, name, start_date, is_current, residents)
		VALUES (?, ?, ?, ?, ?, ?, 1)`, now, now, owner, name, now, current)
	var id uint
	db.Raw(`SELECT MAX(id) FROM properties`).Scan(&id)
	return id
}

// insertMember adds a member of a home, the way the schema before households stores it.
func insertMember(t *testing.T, db *gorm.DB, propertyID uint, userID *uint, name, role string) uint {
	t.Helper()
	now := time.Now()
	mustExec(t, db, `INSERT INTO household_members (created_at, updated_at, property_id, user_id, name, role, is_virtual)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, now, now, propertyID, userID, name, role, userID == nil)
	var id uint
	db.Raw(`SELECT MAX(id) FROM household_members`).Scan(&id)
	return id
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
	cabin := models.Property{ID: insertHome(t, db, author.ID, "Cabin", false)}
	home := models.Property{ID: insertHome(t, db, author.ID, "Home", true)}
	flat := models.Property{ID: insertHome(t, db, lodger.ID, "Flat", false)}
	insertMember(t, db, cabin.ID, &author.ID, "Author", "admin")
	insertMember(t, db, home.ID, &author.ID, "Author", "admin")
	insertMember(t, db, home.ID, &partner.ID, "Partner", "member")
	insertMember(t, db, flat.ID, &lodger.ID, "Lodger", "admin")

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
	db, err := database.Open(filepath.Join(t.TempDir(), "homelog.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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

// Every home becomes a household of its own: members, settlements, split
// settings and join requests follow it, balances stay as they were, and only
// Home expenses keep their home.
func TestUpgradeGroupsHomesIntoHouseholds(t *testing.T) {
	db := beforeHouseholdsDB(t)
	now := time.Now()
	alice := &models.User{Email: "alice@example.com", PasswordHash: "x", Name: "Alice", IsActive: true}
	bob := &models.User{Email: "bob@example.com", PasswordHash: "x", Name: "Bob", IsActive: true}
	mustCreate(t, db, alice, bob)
	mustExec(t, db, `INSERT INTO categories (id, created_at, updated_at, slug, name, is_default) VALUES (1, ?, ?, 'home', 'Home', true), (2, ?, ?, 'food_dining', 'Food', true)`, now, now, now, now)

	flat := insertHome(t, db, alice.ID, "Flat", true)
	lake := insertHome(t, db, alice.ID, "Lake house", false)
	aliceFlat := insertMember(t, db, flat, &alice.ID, "Alice", "admin")
	bobFlat := insertMember(t, db, flat, &bob.ID, "Bob", "member")
	aliceLake := insertMember(t, db, lake, &alice.ID, "Alice", "admin")
	mustExec(t, db, `INSERT INTO household_settings (created_at, updated_at, property_id, split_mode, default_split_type) VALUES (?, ?, ?, true, 'equal'), (?, ?, ?, false, 'equal')`, now, now, flat, now, now, lake)

	expense := func(userID uint, propertyID *uint, categoryID, payer uint, amount float64, split bool) uint {
		t.Helper()
		mustExec(t, db, `INSERT INTO expenses (created_at, updated_at, user_id, property_id, category_id, amount, date, description, paid_by_member_id, is_split)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'x', ?, ?)`, now, now, userID, propertyID, categoryID, amount, now, payer, split)
		var id uint
		db.Raw(`SELECT MAX(id) FROM expenses`).Scan(&id)
		return id
	}
	plumber := expense(alice.ID, &flat, 1, aliceFlat, 100, true)
	groceries := expense(bob.ID, &flat, 2, bobFlat, 40, true)
	homeless := expense(bob.ID, nil, 2, bobFlat, 12, false)
	boat := expense(alice.ID, &lake, 2, aliceLake, 30, false)
	for _, s := range []struct {
		expense, member uint
		amount          float64
		settled         bool
	}{{plumber, aliceFlat, 50, true}, {plumber, bobFlat, 50, false}, {groceries, bobFlat, 20, true}, {groceries, aliceFlat, 20, false}} {
		mustExec(t, db, `INSERT INTO expense_splits (created_at, updated_at, expense_id, member_id, amount, settled_amount, is_settled) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			now, now, s.expense, s.member, s.amount, map[bool]float64{true: s.amount}[s.settled], s.settled)
	}
	mustExec(t, db, `INSERT INTO settlements (created_at, updated_at, property_id, from_member_id, to_member_id, amount, date) VALUES (?, ?, ?, ?, ?, 5, ?)`, now, now, flat, bobFlat, aliceFlat, now)
	mustExec(t, db, `INSERT INTO property_join_requests (created_at, updated_at, user_id, property_id, status) VALUES (?, ?, ?, ?, 'pending')`, now, now, bob.ID, lake)

	if err := upgrade(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if steps := database.PendingUpgrades(db); len(steps) != 0 {
		t.Fatalf("still pending after the upgrade: %v", steps)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("server migration after the upgrade: %v", err)
	}

	var households []models.Household
	db.Order("id").Find(&households)
	if len(households) != 2 || households[0].ID != flat || households[0].Name != "Flat" || households[1].ID != lake || households[1].Name != "Lake house" {
		t.Fatalf("households = %+v, want one per home, with its id and name", households)
	}
	var homes []models.Property
	db.Order("id").Find(&homes)
	for _, h := range homes {
		if h.HouseholdID != h.ID {
			t.Errorf("home %d is in household %d", h.ID, h.HouseholdID)
		}
		// The lake house was not Alice's primary home, but it is the only
		// home of its household now.
		if !h.IsCurrent {
			t.Errorf("home %d is not the primary home of its household", h.ID)
		}
	}
	var members []models.HouseholdMember
	db.Order("id").Find(&members)
	if want := []uint{flat, flat, lake}; len(members) != 3 || members[0].HouseholdID != want[0] || members[1].HouseholdID != want[1] || members[2].HouseholdID != want[2] {
		t.Errorf("members = %+v, want households %v", members, want)
	}

	check := func(id, household uint, property *uint) {
		t.Helper()
		var e models.Expense
		if err := db.First(&e, id).Error; err != nil {
			t.Fatal(err)
		}
		if e.HouseholdID != household || (e.PropertyID == nil) != (property == nil) || (property != nil && *e.PropertyID != *property) {
			t.Errorf("expense %d: household %d, home %v; want %d, %v", id, e.HouseholdID, e.PropertyID, household, property)
		}
	}
	check(plumber, flat, &flat)
	check(groceries, flat, nil)
	check(homeless, flat, nil)
	check(boat, lake, nil)

	var settlement models.Settlement
	db.First(&settlement)
	if settlement.HouseholdID != flat {
		t.Errorf("settlement in household %d, want %d", settlement.HouseholdID, flat)
	}
	var settings []models.HouseholdSettings
	db.Order("household_id").Find(&settings)
	if len(settings) != 2 || settings[0].HouseholdID != flat || !settings[0].SplitMode || settings[1].HouseholdID != lake {
		t.Errorf("household settings = %+v", settings)
	}
	var request models.HouseholdJoinRequest
	if err := db.First(&request).Error; err != nil || request.HouseholdID != lake || request.Status != "pending" {
		t.Errorf("join request = %+v (%v), want the pending one for the lake house", request, err)
	}
	if db.Migrator().HasTable("property_join_requests") {
		t.Error("property_join_requests survived")
	}
	for _, table := range []string{"household_members", "settlements", "household_settings"} {
		if db.Migrator().HasColumn(table, "property_id") {
			t.Errorf("%s.property_id survived", table)
		}
	}

	balance, err := handlers.CalculateBalance(aliceFlat, bobFlat, flat, db)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(balance-30) > 1e-9 {
		t.Errorf("balance Alice-Bob = %.2f, want 30 (Bob owes 50, Alice owes 20)", balance)
	}

	if err := upgrade(db); err != nil {
		t.Fatalf("second run: %v", err)
	}
}
