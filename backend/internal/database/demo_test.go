package database

import (
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/sgiraz/homelog/internal/models"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := Open(":memory:", &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
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

// demoCounts fingerprints the demo dataset by row counts.
func demoCounts(t *testing.T, db *gorm.DB) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	for _, table := range demoTables {
		var n int64
		if err := db.Table(table).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		out[table] = n
	}
	return out
}

// Any visitor can reset the demo while the hourly ticker may be resetting it
// too: concurrent resets must end in exactly one dataset.
func TestResetDemoData_ConcurrentResetsLeaveOneDataset(t *testing.T) {
	db, _ := openFile(t)
	if err := ResetDemoData(db); err != nil {
		t.Fatalf("first reset: %v", err)
	}
	want := demoCounts(t, db)

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- ResetDemoData(db)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent reset: %v", err)
		}
	}

	got := demoCounts(t, db)
	for table, n := range want {
		if got[table] != n {
			t.Errorf("%s: %d rows after concurrent resets, want %d", table, got[table], n)
		}
	}
	var user models.User
	db.Where("email = ?", DemoEmail).First(&user)
	if user.ID != 1 {
		t.Errorf("demo user ID = %d, want 1", user.ID)
	}
}

// A reset that fails half way leaves the previous dataset untouched.
func TestResetDemoData_FailedResetKeepsTheOldDataset(t *testing.T) {
	db, _ := openFile(t)
	if err := ResetDemoData(db); err != nil {
		t.Fatalf("first reset: %v", err)
	}
	want := demoCounts(t, db)

	if err := db.Exec("CREATE TRIGGER fail_seed BEFORE INSERT ON users BEGIN SELECT RAISE(ABORT, 'seed fails'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := ResetDemoData(db); err == nil {
		t.Fatal("reset succeeded although the seed cannot insert the demo user")
	}

	got := demoCounts(t, db)
	for table, n := range want {
		if got[table] != n {
			t.Errorf("%s: %d rows after a failed reset, want %d (the old dataset)", table, got[table], n)
		}
	}
}
