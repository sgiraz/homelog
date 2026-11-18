package handlers

import (
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// TestCascadeDeleteProperty_CoversSchema reads the foreign keys of the real
// schema and fails on any edge into a table cascadeDeleteProperty empties that
// the cascade does not handle. Adding such a key means teaching the cascade
// about it, then listing it here.
func TestCascadeDeleteProperty_CoversSchema(t *testing.T) {
	db := testutil.NewDB(t)

	tree := map[string]bool{
		"properties": true, "household_members": true, "utilities": true,
		"bills": true, "meter_readings": true, "expenses": true,
		"expense_splits": true, "settlements": true, "projects": true,
	}
	handled := map[string]string{
		"bill_installments.bill_id":               "deleted with its bill",
		"bills.utility_id":                        "deleted with its utility",
		"bills.user_reading_id":                   "bills go before readings",
		"expense_splits.expense_id":               "deleted with its expense",
		"expense_splits.member_id":                "deleted with its expense",
		"expense_splits.settlement_id":            "unlinked",
		"expense_templates.project_id":            "unlinked",
		"expenses.project_id":                     "unlinked",
		"expenses.paid_by_member_id":              "deleted with the property",
		"expenses.property_id":                    "deleted with the property",
		"household_members.property_id":           "deleted with the property",
		"household_settings.property_id":          "deleted with the property",
		"meter_readings.utility_id":               "deleted with its utility",
		"notifications.property_id":               "deleted with the property",
		"price_changes.utility_id":                "deleted with its utility",
		"project_members.project_id":              "deleted with its project",
		"projects.property_id":                    "deleted with the property",
		"property_join_requests.property_id":      "deleted with the property",
		"service_communications.utility_id":       "deleted with its utility",
		"settlement_allocations.settlement_id":    "deleted with its settlement",
		"settlement_allocations.expense_split_id": "deleted with its split",
		"settlements.from_member_id":              "deleted with the property",
		"settlements.to_member_id":                "deleted with the property",
		"settlements.property_id":                 "deleted with the property",
		"settlements.target_expense_id":           "unlinked",
		"utilities.paid_by_member_id":             "deleted with the property",
		"utilities.property_id":                   "deleted with the property",
		"utility_rates.utility_id":                "deleted with its utility",
	}

	var tables []string
	if err := db.Raw("SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'").Scan(&tables).Error; err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var missing []string
	for _, table := range tables {
		var keys []struct {
			Table string `gorm:"column:table"`
			From  string `gorm:"column:from"`
		}
		if err := db.Raw("SELECT \"table\", \"from\" FROM pragma_foreign_key_list(?)", table).Scan(&keys).Error; err != nil {
			t.Fatal(err)
		}
		for _, k := range keys {
			if !tree[k.Table] {
				continue
			}
			edge := table + "." + k.From
			seen[edge] = true
			if _, ok := handled[edge]; !ok {
				missing = append(missing, edge+" -> "+k.Table)
			}
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		t.Errorf("foreign key %s is not handled by cascadeDeleteProperty", m)
	}
	for edge := range handled {
		if !seen[edge] {
			t.Errorf("%s is listed as handled but no longer exists in the schema", edge)
		}
	}
}

// Soft-deleted rows keep their foreign keys and settled splits keep pointing
// at their settlement: both used to make the delete fail.
func TestDeleteProperty_SoftDeletedAndSettledRows(t *testing.T) {
	f := setupPropertyFixture(t)
	db := f.db
	now := time.Now()
	id := f.home.ID

	var mOwner models.HouseholdMember
	db.Where("property_id = ? AND user_id = ?", id, f.owner.ID).First(&mOwner)
	var settlement models.Settlement
	db.Where("property_id = ?", id).First(&settlement)
	var util models.Utility
	db.Where("property_id = ?", id).First(&util)
	var cat models.Category
	db.First(&cat)

	if err := db.Model(&models.ExpenseSplit{}).Where("member_id = ?", mOwner.ID).Update("settlement_id", settlement.ID).Error; err != nil {
		t.Fatal(err)
	}

	oldExpense := models.Expense{UserID: f.owner.ID, PropertyID: &id, CategoryID: cat.ID, Amount: 9, Date: now, Description: "Old", PaidByMemberID: mOwner.ID}
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

	rec := doJSON(t, f.router, http.MethodDelete, "/properties/"+itoa(id)+"?new_principal_id="+itoa(f.other.ID), f.ownerTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: status %d, body %s", rec.Code, rec.Body.String())
	}

	for _, c := range []struct {
		name  string
		model any
		query string
	}{
		{"utilities", &models.Utility{}, "property_id = ?"},
		{"expenses", &models.Expense{}, "property_id = ?"},
		{"projects", &models.Project{}, "property_id = ?"},
		{"settlements", &models.Settlement{}, "property_id = ?"},
	} {
		if n := f.count(t, c.model, c.query, id); n != 0 {
			t.Errorf("%s: %d rows left for the deleted property, want 0", c.name, n)
		}
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
