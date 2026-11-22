package handlers

import (
	"sort"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// TestCascadeDeleteProperty_CoversSchema reads the foreign keys of the real
// schema and fails on any edge into a table cascadeDeleteProperty empties that
// is left to nobody: the schema either deletes or unlinks the child itself
// (CASCADE, SET NULL), or the cascade deletes the child first. Adding such a
// key means declaring its action or teaching the cascade, then listing it.
func TestCascadeDeleteProperty_CoversSchema(t *testing.T) {
	db := testutil.NewDB(t)

	tree := map[string]bool{
		"properties": true, "household_members": true, "utilities": true,
		"bills": true, "bill_installments": true, "meter_readings": true,
		"expenses": true, "expense_splits": true, "settlements": true, "projects": true,
		"households": true,
	}
	// Edges the database refuses to orphan: the cascade deletes these children
	// before their parent.
	deletedFirst := map[string]string{
		"bills.user_reading_id":                   "bills go before the utility takes its readings",
		"bills.utility_id":                        "bills go before their utility",
		"bill_templates.property_id":              "deleted with the property",
		"expense_splits.member_id":                "splits go with their expenses, before the members",
		"expenses.bill_id":                        "expenses go before the bills",
		"expenses.bill_installment_id":            "expenses go before the bills",
		"expenses.paid_by_member_id":              "expenses go before the members",
		"expenses.property_id":                    "deleted with the property",
		"expenses.household_id":                   "clearHouseholdLedger deletes them with the last home",
		"household_join_requests.household_id":    "deleteHousehold deletes them",
		"household_members.household_id":          "deleteHousehold deletes them",
		"household_settings.household_id":         "deleteHousehold deletes them",
		"properties.household_id":                 "the last home goes before its household",
		"notifications.property_id":               "deleted with the property",
		"projects.property_id":                    "deleted with the property",
		"settlement_allocations.expense_split_id": "deleteExpenses deletes them first",
		"settlements.from_member_id":              "settlements go first",
		"settlements.to_member_id":                "settlements go first",
		"settlements.household_id":                "clearHouseholdLedger deletes them first",
		"utilities.paid_by_member_id":             "utilities go before the members",
		"utilities.property_id":                   "deleted with the property",
	}

	var edges []struct {
		Child    string
		Column   string
		Parent   string
		OnDelete string
	}
	if err := db.Raw(`SELECT m.name AS child, f."from" AS "column", f."table" AS parent, f.on_delete
		FROM sqlite_master m, pragma_foreign_key_list(m.name) f WHERE m.type = 'table'`).Scan(&edges).Error; err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var missing []string
	for _, e := range edges {
		if !tree[e.Parent] {
			continue
		}
		edge := e.Child + "." + e.Column
		seen[edge] = true
		_, listed := deletedFirst[edge]
		declared := e.OnDelete == "CASCADE" || e.OnDelete == "SET NULL"
		switch {
		case declared && listed:
			t.Errorf("%s is ON DELETE %s: drop it from the list", edge, e.OnDelete)
		case !declared && !listed:
			missing = append(missing, edge+" -> "+e.Parent)
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("foreign key %s is left to nobody: declare its ON DELETE or delete it in cascadeDeleteProperty", m)
	}
	for edge := range deletedFirst {
		if !seen[edge] {
			t.Errorf("%s is listed but no longer exists in the schema", edge)
		}
	}
}

// Soft-deleted rows keep their foreign keys and settled splits keep pointing
// at their settlement: both used to make the delete fail.
func TestDeleteProperty_SoftDeletedAndSettledRows(t *testing.T) {
	f := setupPropertyFixture(t)
	f.memberLeaves(t)
	db := f.db
	now := time.Now()
	id := f.home.ID

	var mOwner models.HouseholdMember
	db.Where("household_id = ? AND user_id = ?", f.home.HouseholdID, f.owner.ID).First(&mOwner)
	var settlement models.Settlement
	db.Where("household_id = ?", f.home.HouseholdID).First(&settlement)
	var util models.Utility
	db.Where("property_id = ?", id).First(&util)
	var cat models.Category
	db.First(&cat)

	if err := db.Model(&models.ExpenseSplit{}).Where("member_id = ?", mOwner.ID).Update("settlement_id", settlement.ID).Error; err != nil {
		t.Fatal(err)
	}

	oldExpense := models.Expense{UserID: f.owner.ID, HouseholdID: f.home.HouseholdID, PropertyID: &id, CategoryID: cat.ID, Amount: 9, Date: now, Description: "Old", PaidByMemberID: mOwner.ID}
	mustCreate(t, db, &oldExpense)
	mustCreate(t, db, &models.ExpenseSplit{ExpenseID: oldExpense.ID, MemberID: mOwner.ID, Amount: 9, IsSettled: true, SettledAt: &now})

	oldBill := models.Bill{UtilityID: util.ID, BillNumber: "G-0", IssueDate: now, PeriodStart: now, PeriodEnd: now, DueDate: now, AmountTotal: 5}
	mustCreate(t, db, &oldBill)
	mustCreate(t, db, &models.BillInstallment{BillID: oldBill.ID, Number: 1, DueDate: now, Amount: 5})

	oldUtil := models.Utility{UserID: f.owner.ID, PropertyID: id, Type: "water", Provider: "Acme Water", IsActive: true}
	mustCreate(t, db, &oldUtil)
	oldUtilBill := models.Bill{UtilityID: oldUtil.ID, BillNumber: "W-1", IssueDate: now, PeriodStart: now, PeriodEnd: now, DueDate: now, AmountTotal: 7}
	mustCreate(t, db, &oldUtilBill)
	mustCreate(t, db, &models.BillInstallment{BillID: oldUtilBill.ID, Number: 1, DueDate: now, Amount: 7})

	oldProject := models.Project{UserID: f.owner.ID, PropertyID: &id, Name: "Garden", StartDate: now, EndDate: now}
	mustCreate(t, db, &oldProject)
	mustCreate(t, db, &models.ProjectMember{ProjectID: oldProject.ID, UserID: f.member.ID, Role: "member"})

	for _, row := range []any{&oldExpense, &oldBill, &oldUtil, &oldProject} {
		if err := db.Delete(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := db.Transaction(func(tx *gorm.DB) error { return cascadeDeleteProperty(tx, id) }); err != nil {
		t.Fatalf("delete: %v", err)
	}

	for _, c := range []struct {
		name  string
		model any
		query string
	}{
		{"utilities", &models.Utility{}, "property_id = ?"},
		{"expenses", &models.Expense{}, "property_id = ?"},
		{"projects", &models.Project{}, "property_id = ?"},
	} {
		if n := f.count(t, c.model, c.query, id); n != 0 {
			t.Errorf("%s: %d rows left for the deleted property, want 0", c.name, n)
		}
	}
	if n := f.count(t, &models.Settlement{}, "household_id = ?", f.home.HouseholdID); n != 0 {
		t.Errorf("settlements: %d rows left for the deleted household, want 0", n)
	}
	for _, c := range []struct {
		name  string
		model any
	}{
		{"bills", &models.Bill{}},
		{"bill_installments", &models.BillInstallment{}},
		{"expense_splits", &models.ExpenseSplit{}},
		{"project_members", &models.ProjectMember{}},
	} {
		if n := f.count(t, c.model, "1 = 1"); n != 0 {
			t.Errorf("%s: %d orphaned rows, want 0", c.name, n)
		}
	}
}
