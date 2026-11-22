package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sgiraz/homelog/internal/models"
)

// Payer and split members must belong to the expense's household: anything
// else would hand a member of another household a share of this expense.
func TestCreateExpense_MembersMustBelongToTheHousehold(t *testing.T) {
	f := setupMoneyFixture(t)

	elsewhere := models.Property{Household: &models.Household{Name: "Other home"}, UserID: f.bob.ID, Name: "Other home", Type: "rented", StartDate: time.Now(), Residents: 1}
	mustCreate(t, f.db, &elsewhere)
	stranger := models.HouseholdMember{HouseholdID: elsewhere.HouseholdID, Name: "Dave", Role: "member", IsVirtual: true}
	mustCreate(t, f.db, &stranger)

	propID := f.prop.ID
	for name, c := range map[string]struct {
		propertyID *uint
		payer      uint
		splitWith  []uint
		code       string
	}{
		"split with a member of another home": {&propID, f.mAlice.ID, []uint{f.mBob.ID, stranger.ID}, "split_members_not_in_household"},
		"payer from another home":             {&propID, stranger.ID, []uint{f.mBob.ID}, "payer_not_in_household"},
		"payer also in the split":             {&propID, f.mAlice.ID, []uint{f.mAlice.ID, f.mBob.ID}, "split_members_invalid"},
		"same member twice":                   {&propID, f.mAlice.ID, []uint{f.mBob.ID, f.mBob.ID}, "split_members_invalid"},
		"neither household nor home":          {nil, f.mAlice.ID, nil, "household_id_required"},
	} {
		body := map[string]any{
			"amount":                30,
			"category_id":           f.homeCatID,
			"date":                  "2026-05-01",
			"description":           "Groceries",
			"paid_by_member_id":     c.payer,
			"is_split":              len(c.splitWith) > 0,
			"split_with_member_ids": c.splitWith,
		}
		if c.propertyID != nil {
			body["property_id"] = *c.propertyID
		}
		rec := doJSON(t, f.router, http.MethodPost, "/expenses", f.aliceTok, body)
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != c.code {
			t.Errorf("%s: status %d, body %s; want 400 %s", name, rec.Code, rec.Body.String(), c.code)
		}
	}

	var n int64
	f.db.Model(&models.Expense{}).Count(&n)
	if n != 0 {
		t.Errorf("%d expenses created by refused requests, want 0", n)
	}
}

// The property is fixed at creation: moving an expense would leave its payer
// and splits in the old home.
func TestUpdateExpense_IgnoresPropertyChange(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 40, []uint{f.mBob.ID})

	elsewhere := models.Property{HouseholdID: f.prop.HouseholdID, UserID: f.alice.ID, Name: "Other home", Type: "rented", StartDate: time.Now(), Residents: 1}
	mustCreate(t, f.db, &elsewhere)

	rec := doJSON(t, f.router, http.MethodPut, "/expenses/"+itoa(id), f.aliceTok, map[string]any{
		"description": "Renamed",
		"property_id": elsewhere.ID,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: status %d, body %s", rec.Code, rec.Body.String())
	}
	var got models.Expense
	f.db.First(&got, id)
	if got.PropertyID == nil || *got.PropertyID != f.prop.ID || got.Description != "Renamed" {
		t.Errorf("expense after update: property %v, description %q; want property %d, description Renamed",
			got.PropertyID, got.Description, f.prop.ID)
	}
}

// Only the Home category carries a home: any other expense belongs to the
// household alone, a Home one without a home gets the household's primary
// home, and changing the category moves the home along.
func TestExpenseHome_FollowsTheCategory(t *testing.T) {
	f := setupMoneyFixture(t)
	food := models.Category{Slug: "food_dining", Name: "Food", IsDefault: true}
	mustCreate(t, f.db, &food)

	create := func(body map[string]any) models.Expense {
		t.Helper()
		body["amount"], body["date"], body["description"] = 10, "2026-05-01", "x"
		rec := doJSON(t, f.router, http.MethodPost, "/expenses", f.aliceTok, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %v: status %d, body %s", body, rec.Code, rec.Body.String())
		}
		var e models.Expense
		if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		f.db.First(&e, e.ID)
		return e
	}

	groceries := create(map[string]any{"category_id": food.ID, "property_id": f.prop.ID})
	if groceries.HouseholdID != f.prop.HouseholdID || groceries.PropertyID != nil {
		t.Errorf("food expense: household %d, home %v; want %d and no home", groceries.HouseholdID, groceries.PropertyID, f.prop.HouseholdID)
	}

	plumber := create(map[string]any{"category_id": f.homeCatID, "household_id": f.prop.HouseholdID})
	if plumber.PropertyID == nil || *plumber.PropertyID != f.prop.ID {
		t.Errorf("Home expense without a home: home %v, want the primary %d", plumber.PropertyID, f.prop.ID)
	}

	rec := doJSON(t, f.router, http.MethodPut, "/expenses/"+itoa(plumber.ID), f.aliceTok, map[string]any{"category_id": food.ID})
	if rec.Code != http.StatusOK {
		t.Fatalf("recategorise: status %d, body %s", rec.Code, rec.Body.String())
	}
	f.db.First(&plumber, plumber.ID)
	if plumber.PropertyID != nil {
		t.Errorf("after moving to Food the expense still has home %d", *plumber.PropertyID)
	}

	rec = doJSON(t, f.router, http.MethodPut, "/expenses/"+itoa(groceries.ID), f.aliceTok, map[string]any{"category_id": f.homeCatID})
	if rec.Code != http.StatusOK {
		t.Fatalf("recategorise: status %d, body %s", rec.Code, rec.Body.String())
	}
	f.db.First(&groceries, groceries.ID)
	if groceries.PropertyID == nil || *groceries.PropertyID != f.prop.ID {
		t.Errorf("after moving to Home: home %v, want the primary %d", groceries.PropertyID, f.prop.ID)
	}

	elsewhere := models.Property{Household: &models.Household{Name: "Other"}, UserID: f.bob.ID, Name: "Other", StartDate: time.Now()}
	mustCreate(t, f.db, &elsewhere)
	rec = doJSON(t, f.router, http.MethodPost, "/expenses", f.aliceTok, map[string]any{
		"amount": 10, "date": "2026-05-01", "category_id": f.homeCatID,
		"household_id": f.prop.HouseholdID, "property_id": elsewhere.ID,
	})
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "home_not_in_household" {
		t.Errorf("home of another household: status %d, body %s; want 400 home_not_in_household", rec.Code, rec.Body.String())
	}
}
