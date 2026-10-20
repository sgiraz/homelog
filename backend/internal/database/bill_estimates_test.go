package database

import (
	"testing"
	"time"

	"github.com/sgiraz/homelog/internal/models"
)

// A bill used to keep its estimate apart from the provider reading. The move
// makes the estimate the closing reading and drops the old columns.
func TestMoveBillEstimatesToClosingReading(t *testing.T) {
	db := newTestDB(t)
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	mustExec(`ALTER TABLE bills ADD COLUMN estimated_date datetime`)
	mustExec(`ALTER TABLE bills ADD COLUMN estimated_reading real`)
	mustExec(`ALTER TABLE bills ADD COLUMN estimated_consumption real`)

	user := &models.User{Email: "u@example.com", PasswordHash: "x", Name: "U", IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	home := &models.Property{UserID: user.ID, Name: "Home", StartDate: time.Now()}
	if err := db.Create(home).Error; err != nil {
		t.Fatal(err)
	}
	water := &models.Utility{UserID: user.ID, PropertyID: home.ID, Type: "water", Provider: "Etra"}
	if err := db.Create(water).Error; err != nil {
		t.Fatal(err)
	}
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	reading := func(v float64) *float64 { return &v }
	estimated := models.Bill{UtilityID: water.ID, BillNumber: "E", IssueDate: day(6, 20), PeriodStart: day(1, 4), PeriodEnd: day(6, 16),
		DueDate: day(7, 10), ProviderReading: reading(119), ReadingType: "actual", ConsumptionTotal: 53}
	actual := models.Bill{UtilityID: water.ID, BillNumber: "A", IssueDate: day(10, 9), PeriodStart: day(5, 29), PeriodEnd: day(10, 7),
		DueDate: day(11, 1), ProviderReading: reading(145), ProviderReadingDate: &[]time.Time{day(10, 1)}[0], ReadingType: "actual"}
	for _, b := range []*models.Bill{&estimated, &actual} {
		if err := db.Create(b).Error; err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`UPDATE bills SET estimated_date = ?, estimated_reading = 122, estimated_consumption = 3 WHERE id = ?`, day(6, 16), estimated.ID)
	tpl := models.BillTemplate{PropertyID: home.ID, Name: "Etra", Provider: "Etra", UtilityType: "water", ExtractionRules: `[` +
		`{"field":"amount_total","anchor_text":"Totale"},{"field":"provider_reading","anchor_text":"attuale"},` +
		`{"field":"provider_reading_date"},{"field":"estimated_reading","anchor_text":"Stimata"},{"field":"estimated_date"}]`}
	if err := db.Create(&tpl).Error; err != nil {
		t.Fatal(err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var got []models.Bill
	db.Order("id").Find(&got)
	if e := got[0]; *e.ProviderReading != 122 || e.ReadingType != "estimated" || !e.ProviderReadingDate.Equal(day(6, 16)) || e.ConsumptionTotal != 53 {
		t.Errorf("estimated bill = reading %v on %v (%s), consumption %v; want 122 on 16 Jun (estimated), 53",
			*e.ProviderReading, e.ProviderReadingDate, e.ReadingType, e.ConsumptionTotal)
	}
	if a := got[1]; *a.ProviderReading != 145 || a.ReadingType != "actual" || !a.ProviderReadingDate.Equal(day(10, 1)) {
		t.Errorf("a bill without an estimate must be left alone: %+v", a)
	}
	db.First(&tpl, tpl.ID)
	if want := `[{"anchor_text":"Totale","field":"amount_total"},{"anchor_text":"Stimata","field":"provider_reading"},{"field":"provider_reading_date"}]`; tpl.ExtractionRules != want {
		t.Errorf("template rules = %s, want the estimate rules as the closing reading: %s", tpl.ExtractionRules, want)
	}
	for _, col := range []string{"estimated_date", "estimated_reading", "estimated_consumption"} {
		if db.Migrator().HasColumn(&models.Bill{}, col) {
			t.Errorf("column %s still exists", col)
		}
	}
	if err := AutoMigrate(db); err != nil {
		t.Errorf("second boot: %v", err)
	}
}
