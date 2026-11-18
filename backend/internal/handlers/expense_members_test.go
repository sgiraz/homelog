package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/sgiraz/homelog/internal/models"
)

// Payer and split members must belong to the expense's property: anything
// else would hand a member of another home a share of this expense.
func TestCreateExpense_MembersMustBelongToTheProperty(t *testing.T) {
	f := setupMoneyFixture(t)

	elsewhere := models.Property{UserID: f.bob.ID, Name: "Other home", Type: "rented", StartDate: time.Now(), Residents: 1}
	mustCreate(t, f.db, &elsewhere)
	stranger := models.HouseholdMember{PropertyID: elsewhere.ID, Name: "Dave", Role: "member", IsVirtual: true}
	mustCreate(t, f.db, &stranger)

	propID := f.prop.ID
	for name, c := range map[string]struct {
		propertyID *uint
		payer      uint
		splitWith  []uint
		code       string
	}{
		"split with a member of another home": {&propID, f.mAlice.ID, []uint{f.mBob.ID, stranger.ID}, "split_members_not_in_property"},
		"payer from another home":             {&propID, stranger.ID, []uint{f.mBob.ID}, "payer_not_in_property"},
		"payer also in the split":             {&propID, f.mAlice.ID, []uint{f.mAlice.ID, f.mBob.ID}, "split_members_invalid"},
		"same member twice":                   {&propID, f.mAlice.ID, []uint{f.mBob.ID, f.mBob.ID}, "split_members_invalid"},
		"payer without a property":            {nil, f.mAlice.ID, nil, "payer_not_in_property"},
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

	elsewhere := models.Property{UserID: f.alice.ID, Name: "Other home", Type: "rented", StartDate: time.Now(), Residents: 1}
	mustCreate(t, f.db, &elsewhere)
	mustCreate(t, f.db, &models.HouseholdMember{PropertyID: elsewhere.ID, UserID: &f.alice.ID, Name: "Alice", Role: "admin"})

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
