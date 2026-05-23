package database

import (
	"testing"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sgiraz/homelog/internal/models"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	db.Exec("PRAGMA foreign_keys=ON;")
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestResetDemoData_SeedsExpectedDataset(t *testing.T) {
	db := newTestDB(t)

	if err := ResetDemoData(db); err != nil {
		t.Fatalf("ResetDemoData: %v", err)
	}

	// Demo user must be the first row (ID 1) so existing demo JWTs stay valid.
	var user models.User
	if err := db.Where("email = ?", DemoEmail).First(&user).Error; err != nil {
		t.Fatalf("demo user not found: %v", err)
	}
	if user.ID != 1 {
		t.Errorf("demo user ID = %d, want 1", user.ID)
	}
	if user.Role != "admin" {
		t.Errorf("demo user role = %q, want admin", user.Role)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(DemoPassword)); err != nil {
		t.Errorf("demo password does not verify: %v", err)
	}

	// Onboarding must be marked complete or the demo lands on the wizard.
	var settings models.UserSettings
	if err := db.Where("user_id = ?", user.ID).First(&settings).Error; err != nil {
		t.Fatalf("user settings not found: %v", err)
	}
	if !settings.OnboardingCompleted {
		t.Error("demo user settings: OnboardingCompleted = false, want true")
	}

	// Spot-check that each major entity is populated.
	type check struct {
		model any
		min   int64
	}
	for name, c := range map[string]check{
		"properties":   {&models.Property{}, 1},
		"members":      {&models.HouseholdMember{}, 3},
		"utilities":    {&models.Utility{}, 4},
		"readings":     {&models.MeterReading{}, 6},
		"bills":        {&models.Bill{}, 4},
		"installments": {&models.BillInstallment{}, 4},
		"expenses":     {&models.Expense{}, 8},
		"projects":     {&models.Project{}, 2},
		"settlements":  {&models.Settlement{}, 1},
	} {
		var n int64
		db.Model(c.model).Count(&n)
		if n < c.min {
			t.Errorf("%s count = %d, want >= %d", name, n, c.min)
		}
	}

	// FTS index must be repopulated by the GORM AfterSave hooks fired on Create.
	var fts int64
	db.Raw("SELECT count(*) FROM search_index").Scan(&fts)
	if fts == 0 {
		t.Error("search_index is empty after seed; FTS hooks did not fire")
	}
}

// TestResetDemoData_PayerSplitsSettled guards the expense.go invariant: the
// payer's own split must be settled (they owe nothing to themselves).
func TestResetDemoData_PayerSplitsSettled(t *testing.T) {
	db := newTestDB(t)
	if err := ResetDemoData(db); err != nil {
		t.Fatalf("ResetDemoData: %v", err)
	}

	var expenses []models.Expense
	db.Where("is_split = ?", true).Find(&expenses)
	if len(expenses) == 0 {
		t.Fatal("expected at least one split expense in the demo dataset")
	}
	for _, e := range expenses {
		var payerSplit models.ExpenseSplit
		err := db.Where("expense_id = ? AND member_id = ?", e.ID, e.PaidByMemberID).First(&payerSplit).Error
		if err != nil {
			t.Errorf("expense %d: payer split missing: %v", e.ID, err)
			continue
		}
		if !payerSplit.IsSettled {
			t.Errorf("expense %d: payer split IsSettled = false, want true", e.ID)
		}
	}
}

// TestResetDemoData_Idempotent ensures repeated resets produce a stable dataset
// (same demo user ID, no row accumulation).
func TestResetDemoData_Idempotent(t *testing.T) {
	db := newTestDB(t)

	if err := ResetDemoData(db); err != nil {
		t.Fatalf("first reset: %v", err)
	}
	var firstExpenses int64
	db.Model(&models.Expense{}).Count(&firstExpenses)

	if err := ResetDemoData(db); err != nil {
		t.Fatalf("second reset: %v", err)
	}

	var user models.User
	db.Where("email = ?", DemoEmail).First(&user)
	if user.ID != 1 {
		t.Errorf("after second reset demo user ID = %d, want 1", user.ID)
	}

	var secondExpenses int64
	db.Model(&models.Expense{}).Count(&secondExpenses)
	if firstExpenses != secondExpenses {
		t.Errorf("expense count drifted across resets: %d then %d", firstExpenses, secondExpenses)
	}

	var users int64
	db.Model(&models.User{}).Count(&users)
	if users != 1 {
		t.Errorf("user count = %d after two resets, want 1", users)
	}
}
