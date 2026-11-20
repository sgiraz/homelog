package database

import (
	"bytes"
	"log"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
)

// downgrade rewrites a table as an install from before the declared actions
// has it: every ON DELETE dropped, and the constraints named in drop gone.
func downgrade(t *testing.T, db *gorm.DB, table string, drop ...string) {
	t.Helper()
	err := WithoutForeignKeys(db, func(conn *gorm.DB) error {
		var create string
		var indexes []string
		conn.Raw(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&create)
		conn.Raw(`SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND sql IS NOT NULL`, table).Scan(&indexes)
		create = regexp.MustCompile(` ON DELETE (CASCADE|SET NULL)`).ReplaceAllString(create, "")
		for _, name := range drop {
			create = regexp.MustCompile(",CONSTRAINT `"+name+"` FOREIGN KEY [^,)]*\\)[^,)]*\\)").ReplaceAllString(create, "")
		}
		create = strings.Replace(create, "CREATE TABLE `"+table+"`", "CREATE TABLE `"+table+"_old`", 1)
		steps := append([]string{
			create,
			"INSERT INTO `" + table + "_old` SELECT * FROM `" + table + "`",
			"DROP TABLE `" + table + "`",
			"ALTER TABLE `" + table + "_old` RENAME TO `" + table + "`",
		}, indexes...)
		for _, s := range steps {
			if err := conn.Exec(s).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("downgrade %s: %v", table, err)
	}
}

func create(t *testing.T, db *gorm.DB, rows ...any) {
	t.Helper()
	for _, r := range rows {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("create %T: %v", r, err)
		}
	}
}

func indexNames(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	db.Raw(`SELECT name FROM sqlite_master WHERE type = 'index' ORDER BY name`).Scan(&names)
	return names
}

// An existing install gets the foreign keys the models declare on its next
// start, with its rows, indexes and unknown columns intact.
func TestAutoMigrate_AlignsTheForeignKeysOfExistingTables(t *testing.T) {
	db, _ := openFile(t)
	wantIndexes := indexNames(t, db)

	downgrade(t, db, "expenses", "fk_expenses_bill", "fk_expenses_bill_installment")
	downgrade(t, db, "bill_installments", "fk_bill_installments_expense")
	if err := db.Exec("ALTER TABLE expenses ADD COLUMN legacy_note text").Error; err != nil {
		t.Fatal(err)
	}
	var keys []foreignKey
	if keys, _ = foreignKeys(db, "expenses"); len(keys) == 0 || keys[0].OnDelete != "NO ACTION" {
		t.Fatalf("downgrade did not take: %+v", keys)
	}

	now := time.Now()
	user := models.User{Email: "a@example.com", PasswordHash: "x", Name: "A", Role: "admin", IsActive: true}
	create(t, db, &user)
	home := models.Property{Household: &models.Household{Name: "Home"}, UserID: user.ID, Name: "Home", StartDate: now}
	create(t, db, &home)
	member := models.HouseholdMember{HouseholdID: home.HouseholdID, UserID: &user.ID, Name: "A"}
	category := models.Category{Name: "Home"}
	create(t, db, &member, &category)
	utility := models.Utility{UserID: user.ID, PropertyID: home.ID, Type: "gas", Provider: "Gas Co", IsMetered: true}
	create(t, db, &utility)
	bill := models.Bill{UtilityID: utility.ID, BillNumber: "B-1", IssueDate: now, PeriodStart: now, PeriodEnd: now, DueDate: now, AmountTotal: 50}
	create(t, db, &bill)
	expense := models.Expense{UserID: user.ID, HouseholdID: home.HouseholdID, PropertyID: &home.ID, CategoryID: category.ID, BillID: &bill.ID, Amount: 50, Date: now, PaidByMemberID: member.ID}
	create(t, db, &expense)
	installment := models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: now, Amount: 50, ExpenseID: &expense.ID}
	create(t, db, &installment)
	// Project 999 never existed: a link the schema would have cleared.
	if err := WithoutForeignKeys(db, func(conn *gorm.DB) error {
		return conn.Exec("UPDATE expenses SET project_id = 999, legacy_note = 'kept' WHERE id = ?", expense.ID).Error
	}); err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(prev) })
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	if !strings.Contains(logs.String(), "Rebuilding expenses") || !strings.Contains(logs.String(), "Rebuilding bill_installments") {
		t.Errorf("expected both tables rebuilt; log: %s", logs.String())
	}

	ref, closeRef, err := referenceSchema()
	if err != nil {
		t.Fatal(err)
	}
	defer closeRef()
	for _, table := range []string{"expenses", "bill_installments"} {
		want, _ := foreignKeys(ref, table)
		have, _ := foreignKeys(db, table)
		if !reflect.DeepEqual(want, have) {
			t.Errorf("%s foreign keys:\n have %+v\n want %+v", table, have, want)
		}
	}
	if got := indexNames(t, db); !reflect.DeepEqual(got, wantIndexes) {
		t.Errorf("indexes after the rebuild:\n have %v\n want %v", got, wantIndexes)
	}
	var row struct {
		ProjectID  *uint
		BillID     *uint
		LegacyNote string
	}
	db.Raw("SELECT project_id, bill_id, legacy_note FROM expenses WHERE id = ?", expense.ID).Scan(&row)
	if row.ProjectID != nil || row.BillID == nil || *row.BillID != bill.ID || row.LegacyNote != "kept" {
		t.Errorf("expense after the rebuild: project %v, bill %v, legacy %q; want no project, bill 1, legacy kept",
			row.ProjectID, row.BillID, row.LegacyNote)
	}
	var violations int64
	db.Raw("SELECT count(*) FROM pragma_foreign_key_check").Scan(&violations)
	if violations != 0 {
		t.Errorf("%d rows violate foreign keys after the rebuild", violations)
	}

	// The declared actions now apply: the installment goes with its bill once
	// the expense paying it is gone.
	if err := db.Exec("DELETE FROM expenses WHERE id = ?", expense.ID).Error; err != nil {
		t.Fatal(err)
	}
	var expenseID *uint
	db.Raw("SELECT expense_id FROM bill_installments WHERE id = ?", installment.ID).Scan(&expenseID)
	if expenseID != nil {
		t.Errorf("installment still points at deleted expense %d", *expenseID)
	}
	if err := db.Exec("DELETE FROM bills WHERE id = ?", bill.ID).Error; err != nil {
		t.Fatal(err)
	}
	var installments int64
	db.Raw("SELECT count(*) FROM bill_installments").Scan(&installments)
	if installments != 0 {
		t.Error("the installment outlived its bill")
	}

	// A second start finds nothing to do.
	logs.Reset()
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), "Rebuilding") {
		t.Errorf("second start rebuilt again; log: %s", logs.String())
	}
}

// A database created by the current models has nothing to align.
func TestAutoMigrate_NewDatabaseMatchesTheModels(t *testing.T) {
	db, _ := openFile(t)
	ref, closeRef, err := referenceSchema()
	if err != nil {
		t.Fatal(err)
	}
	defer closeRef()
	var tables []string
	ref.Raw(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables)
	for _, table := range tables {
		want, _ := foreignKeys(ref, table)
		have, _ := foreignKeys(db, table)
		if !reflect.DeepEqual(want, have) {
			t.Errorf("%s: have %+v, want %+v", table, have, want)
		}
	}
}
