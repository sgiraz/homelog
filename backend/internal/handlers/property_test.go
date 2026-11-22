package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

type propertyFixture struct {
	router    *gin.Engine
	db        *gorm.DB
	owner     *models.User
	member    *models.User
	neighbour *models.User
	home      models.Property // principal, shared with member, full of data
	other     models.Property // the neighbour's home, older, primary of another household
	mOwner    models.HouseholdMember
	mMember   models.HouseholdMember
	mOther    models.HouseholdMember
	ownerTok  string
	memTok    string
	neighTok  string
}

func setupPropertyFixture(t *testing.T) *propertyFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	now := time.Now()

	owner := &models.User{Email: "owner@example.com", PasswordHash: "x", Name: "Owner", Role: "admin", IsActive: true}
	member := &models.User{Email: "member@example.com", PasswordHash: "x", Name: "Member", Role: "user", IsActive: true}
	neighbour := &models.User{Email: "neighbour@example.com", PasswordHash: "x", Name: "Neighbour", Role: "user", IsActive: true}
	mustCreate(t, db, owner, member, neighbour)

	other := models.Property{Household: &models.Household{Name: "Seaside flat"}, UserID: neighbour.ID, Name: "Seaside flat", Type: "rented", StartDate: now, IsCurrent: true, Residents: 1}
	mustCreate(t, db, &other)
	mOther := models.HouseholdMember{HouseholdID: other.HouseholdID, UserID: &neighbour.ID, Name: "Neighbour", Role: "admin"}
	mustCreate(t, db, &mOther)

	home := models.Property{Household: &models.Household{Name: "Main house"}, UserID: owner.ID, Name: "Main house", Type: "owned", StartDate: now, IsCurrent: true, Residents: 2}
	mustCreate(t, db, &home)
	mOwner := models.HouseholdMember{HouseholdID: home.HouseholdID, UserID: &owner.ID, Name: "Owner", Role: "admin"}
	mMember := models.HouseholdMember{HouseholdID: home.HouseholdID, UserID: &member.ID, Name: "Member", Role: "member"}
	mustCreate(t, db, &mOwner, &mMember, &models.HouseholdSettings{HouseholdID: home.HouseholdID})

	cat := models.Category{Name: "Groceries"}
	mustCreate(t, db, &cat)

	reading := 100.0
	util := models.Utility{UserID: owner.ID, PropertyID: home.ID, Type: "gas", Provider: "Acme Gas", IsMetered: true, IsActive: true}
	mustCreate(t, db, &util)
	bill := models.Bill{UtilityID: util.ID, BillNumber: "G-1", IssueDate: now, PeriodStart: now, PeriodEnd: now, DueDate: now, AmountTotal: 50}
	mustCreate(t, db, &bill, &models.MeterReading{UtilityID: util.ID, ReadingDate: now, Value: &reading})
	mustCreate(t, db, &models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: now, Amount: 50})
	project := models.Project{UserID: owner.ID, PropertyID: &home.ID, Name: "Kitchen", Budget: 1000, StartDate: now, EndDate: now}
	mustCreate(t, db, &project)
	mustCreate(t, db, &models.ProjectMember{ProjectID: project.ID, UserID: member.ID, Role: "member"})
	exp := models.Expense{UserID: owner.ID, HouseholdID: home.HouseholdID, PropertyID: &home.ID, ProjectID: &project.ID, CategoryID: cat.ID, Amount: 40, Date: now, Description: "Tiles", PaidByMemberID: mOwner.ID}
	mustCreate(t, db, &exp)
	memberSplit := models.ExpenseSplit{ExpenseID: exp.ID, MemberID: mMember.ID, Amount: 20}
	settlement := models.Settlement{HouseholdID: home.HouseholdID, FromMemberID: mMember.ID, ToMemberID: mOwner.ID, Amount: 5, Date: now, TargetExpenseID: &exp.ID}
	mustCreate(t, db,
		&models.ExpenseSplit{ExpenseID: exp.ID, MemberID: mOwner.ID, Amount: 20, IsSettled: true, SettledAt: &now},
		&memberSplit,
		&settlement,
		&models.Notification{UserID: member.ID, Type: "expense_shared", Title: "Shared", PropertyID: &home.ID},
	)
	mustCreate(t, db, &models.SettlementAllocation{SettlementID: settlement.ID, ExpenseSplitID: memberSplit.ID, Amount: 5})

	// A record on the other home that must survive.
	mustCreate(t, db, &models.Expense{UserID: neighbour.ID, HouseholdID: other.HouseholdID, PropertyID: &other.ID, CategoryID: cat.ID, Amount: 10, Date: now, Description: "Rent", PaidByMemberID: mOther.ID})

	gin.SetMode(gin.TestMode)
	h := NewPropertyHandler(db)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.GET("/properties", h.List)
	p.POST("/properties", h.Create)
	p.PUT("/properties/:id", h.Update)
	p.DELETE("/properties/:id", h.Delete)

	return &propertyFixture{
		router: r, db: db, owner: owner, member: member, neighbour: neighbour, home: home, other: other,
		mOwner: mOwner, mMember: mMember, mOther: mOther,
		ownerTok: testutil.SignToken(t, owner), memTok: testutil.SignToken(t, member), neighTok: testutil.SignToken(t, neighbour),
	}
}

// memberLeaves turns the member's account-linked row virtual, as deleting
// their account does: the history stays, the home is no longer shared.
func (f *propertyFixture) memberLeaves(t *testing.T) {
	t.Helper()
	if err := f.db.Model(&models.HouseholdMember{}).Where("id = ?", f.mMember.ID).
		Updates(map[string]any{"is_virtual": true, "user_id": nil}).Error; err != nil {
		t.Fatal(err)
	}
}

// dropHomeExpenses empties the primary home of its Home expenses, which would
// otherwise keep it from being deleted while its household has other homes.
func (f *propertyFixture) dropHomeExpenses(t *testing.T) {
	t.Helper()
	var ids []uint
	if err := f.db.Model(&models.Expense{}).Where("property_id = ?", f.home.ID).Pluck("id", &ids).Error; err != nil {
		t.Fatal(err)
	}
	if err := deleteExpenses(f.db, ids); err != nil {
		t.Fatal(err)
	}
}

func (f *propertyFixture) count(t *testing.T, model any, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := f.db.Unscoped().Model(model).Where(query, args...).Count(&n).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	return n
}

func TestDeleteProperty_CascadesItsHousehold(t *testing.T) {
	f := setupPropertyFixture(t)
	f.memberLeaves(t)
	id := f.home.ID
	mustCreate(t, f.db, &models.UserSettings{UserID: f.owner.ID,
		DefaultSplitWithMemberIDs: "[" + itoa(f.mMember.ID) + "," + itoa(f.mOther.ID) + "]"})

	var before int64
	if err := f.db.Raw("SELECT COUNT(*) FROM search_index WHERE property_id = ?", id).Scan(&before).Error; err != nil || before == 0 {
		t.Fatalf("precondition: search_index should hold the home's rows (n=%d, err=%v)", before, err)
	}

	// An account's only household is reached through deleting the account:
	// the API refuses to delete a user's last home.
	if err := f.db.Transaction(func(tx *gorm.DB) error { return cascadeDeleteProperty(tx, id) }); err != nil {
		t.Fatalf("delete: %v", err)
	}

	for _, c := range []struct {
		name  string
		model any
		query string
	}{
		{"properties", &models.Property{}, "id = ?"},
		{"utilities", &models.Utility{}, "property_id = ?"},
		{"expenses", &models.Expense{}, "property_id = ?"},
		{"projects", &models.Project{}, "property_id = ?"},
		{"notifications", &models.Notification{}, "property_id = ?"},
	} {
		if n := f.count(t, c.model, c.query, id); n != 0 {
			t.Errorf("%s: %d rows left for the deleted property, want 0", c.name, n)
		}
	}
	// Its household had no other home: it goes too, with everything it held.
	for _, c := range []struct {
		name  string
		model any
		query string
	}{
		{"households", &models.Household{}, "id = ?"},
		{"household_members", &models.HouseholdMember{}, "household_id = ?"},
		{"household_settings", &models.HouseholdSettings{}, "household_id = ?"},
		{"settlements", &models.Settlement{}, "household_id = ?"},
		{"expenses", &models.Expense{}, "household_id = ?"},
	} {
		if n := f.count(t, c.model, c.query, f.home.HouseholdID); n != 0 {
			t.Errorf("%s: %d rows left for the deleted household, want 0", c.name, n)
		}
	}
	for _, c := range []struct {
		name  string
		model any
	}{
		{"bills", &models.Bill{}},
		{"bill_installments", &models.BillInstallment{}},
		{"settlement_allocations", &models.SettlementAllocation{}},
		{"meter_readings", &models.MeterReading{}},
		{"expense_splits", &models.ExpenseSplit{}},
		{"project_members", &models.ProjectMember{}},
	} {
		if n := f.count(t, c.model, "1 = 1"); n != 0 {
			t.Errorf("%s: %d orphaned rows, want 0", c.name, n)
		}
	}
	var indexed int64
	f.db.Raw("SELECT COUNT(*) FROM search_index WHERE property_id = ?", id).Scan(&indexed)
	if indexed != 0 {
		t.Errorf("search_index: %d rows left for the deleted property, want 0", indexed)
	}

	if n := f.count(t, &models.Expense{}, "property_id = ?", f.other.ID); n != 1 {
		t.Errorf("other home's expenses: %d, want 1 (untouched)", n)
	}
	var other models.Property
	f.db.First(&other, f.other.ID)
	if !other.IsCurrent {
		t.Error("another household's primary home must stay primary")
	}

	var settings models.UserSettings
	f.db.Where("user_id = ?", f.owner.ID).First(&settings)
	if want := "[" + itoa(f.mOther.ID) + "]"; settings.DefaultSplitWithMemberIDs != want {
		t.Errorf("default split = %s, want %s (deleted members removed)", settings.DefaultSplitWithMemberIDs, want)
	}
}

func TestDeleteProperty_SharedHomeIsRefused(t *testing.T) {
	f := setupPropertyFixture(t)

	isShared := func() map[uint]bool {
		t.Helper()
		rec := doJSON(t, f.router, http.MethodGet, "/properties", f.ownerTok, nil)
		var props []models.Property
		if err := json.Unmarshal(rec.Body.Bytes(), &props); err != nil {
			t.Fatalf("decode properties: %v", err)
		}
		out := map[uint]bool{}
		for _, p := range props {
			out[p.ID] = p.IsShared
		}
		return out
	}
	if got := isShared(); !got[f.home.ID] || got[f.other.ID] {
		t.Errorf("is_shared = %v, want home shared and other not", got)
	}

	rec := doJSON(t, f.router, http.MethodDelete, "/properties/"+itoa(f.home.ID)+"?new_principal_id="+itoa(f.other.ID), f.ownerTok, nil)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "property_has_members" {
		t.Fatalf("delete shared home: status %d, body %s; want 409 property_has_members", rec.Code, rec.Body.String())
	}
	if n := f.count(t, &models.Expense{}, "property_id = ?", f.home.ID); n == 0 {
		t.Error("a refused delete must not touch the property's data")
	}

	// Virtual members have no account to lose the home from.
	f.memberLeaves(t)
	if got := isShared(); got[f.home.ID] {
		t.Error("a home with only virtual members besides the owner is not shared")
	}
}

func TestDeleteProperty_LastOneIsRefused(t *testing.T) {
	f := setupPropertyFixture(t)
	f.memberLeaves(t)

	rec := doJSON(t, f.router, http.MethodDelete, "/properties/"+itoa(f.home.ID), f.ownerTok, nil)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "last_property" {
		t.Fatalf("delete last home: status %d, body %s; want 409 last_property", rec.Code, rec.Body.String())
	}
	if n := f.count(t, &models.Property{}, "id = ?", f.home.ID); n != 1 {
		t.Error("the last home must survive a refused delete")
	}
}

func TestDeleteProperty_MemberCannotDelete(t *testing.T) {
	f := setupPropertyFixture(t)
	rec := doJSON(t, f.router, http.MethodDelete, "/properties/"+itoa(f.home.ID), f.memTok, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("member delete: status %d, want 404", rec.Code)
	}
	if n := f.count(t, &models.Utility{}, "property_id = ?", f.home.ID); n != 1 {
		t.Error("a refused delete must not touch the property's data")
	}
}

func TestUpdateProperty(t *testing.T) {
	f := setupPropertyFixture(t)
	second := models.Property{HouseholdID: f.home.HouseholdID, UserID: f.owner.ID, Name: "Cabin", Type: "rented", StartDate: time.Now(), Residents: 1}
	mustCreate(t, f.db, &second)
	path := "/properties/" + itoa(second.ID)

	for name, body := range map[string]map[string]any{
		"blank name":     {"name": "   "},
		"zero residents": {"residents": 0},
		"unknown type":   {"type": "castle"},
	} {
		rec := doJSON(t, f.router, http.MethodPut, path, f.ownerTok, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, rec.Code)
		}
	}

	rec := doJSON(t, f.router, http.MethodPut, path, f.ownerTok, map[string]any{
		"name": " Lake house ", "address": "1 Shore Rd", "type": "owned", "residents": 3, "is_current": true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: status %d, body %s", rec.Code, rec.Body.String())
	}
	var got models.Property
	f.db.First(&got, second.ID)
	if got.Name != "Lake house" || got.Address != "1 Shore Rd" || got.Type != "owned" || got.Residents != 3 || !got.IsCurrent {
		t.Errorf("updated property = %+v", got)
	}
	// Primary is per household: the old one steps down, another household's
	// stays.
	var home, other models.Property
	f.db.First(&home, f.home.ID)
	f.db.First(&other, f.other.ID)
	if home.IsCurrent || !other.IsCurrent {
		t.Errorf("home primary %v (want false), other household's primary %v (want true)", home.IsCurrent, other.IsCurrent)
	}

	if rec := doJSON(t, f.router, http.MethodPut, path, f.memTok, map[string]any{"name": "Mine"}); rec.Code != http.StatusNotFound {
		t.Errorf("non-owner update: status %d, want 404", rec.Code)
	}
}

func TestDeleteProperty_PrincipalNeedsAValidSuccessor(t *testing.T) {
	f := setupPropertyFixture(t)
	f.memberLeaves(t)
	f.dropHomeExpenses(t)
	sibling := models.Property{HouseholdID: f.home.HouseholdID, UserID: f.owner.ID, Name: "New house", Type: "owned", StartDate: time.Now(), Residents: 2}
	mustCreate(t, f.db, &sibling)
	base := "/properties/" + itoa(f.home.ID)

	for name, c := range map[string]struct {
		query, code string
	}{
		"missing":         {"", "new_principal_required"},
		"itself":          {"?new_principal_id=" + itoa(f.home.ID), "new_principal_invalid"},
		"other household": {"?new_principal_id=" + itoa(f.other.ID), "new_principal_invalid"},
		"not a number":    {"?new_principal_id=abc", "new_principal_invalid"},
	} {
		rec := doJSON(t, f.router, http.MethodDelete, base+c.query, f.ownerTok, nil)
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != c.code {
			t.Errorf("%s: status %d, body %s; want 400 %s", name, rec.Code, rec.Body.String(), c.code)
		}
	}
	if n := f.count(t, &models.Property{}, "id = ?", f.home.ID); n != 1 {
		t.Fatal("a refused delete must leave the primary home in place")
	}

	rec := doJSON(t, f.router, http.MethodDelete, base+"?new_principal_id="+itoa(sibling.ID), f.ownerTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete with sibling successor: status %d, body %s", rec.Code, rec.Body.String())
	}
	f.db.First(&sibling, sibling.ID)
	if !sibling.IsCurrent {
		t.Error("the chosen home should have been promoted to primary")
	}
	// The household keeps its ledger: it still has a home.
	if n := f.count(t, &models.HouseholdMember{}, "household_id = ?", f.home.HouseholdID); n == 0 {
		t.Error("deleting one of two homes must keep the household's members")
	}
}

// An account belongs to one household, so a new home joins the user's. A
// household always has exactly one primary home, and promoting one only
// demotes the homes of that same household.
func TestCreateProperty_InAnExistingHousehold(t *testing.T) {
	f := setupPropertyFixture(t)

	rec := doJSON(t, f.router, http.MethodPost, "/properties", f.ownerTok, map[string]any{
		"name": "New house", "type": "owned",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create in household: status %d, body %s", rec.Code, rec.Body.String())
	}
	var created models.Property
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.HouseholdID != f.home.HouseholdID || created.IsCurrent {
		t.Errorf("created = household %d primary %v, want household %d, not primary", created.HouseholdID, created.IsCurrent, f.home.HouseholdID)
	}

	// Promote it: the old primary of the same household steps down, the
	// other household's primary does not.
	if rec := doJSON(t, f.router, http.MethodPut, "/properties/"+itoa(created.ID), f.ownerTok, map[string]any{"is_current": true}); rec.Code != http.StatusOK {
		t.Fatalf("promote: status %d, body %s", rec.Code, rec.Body.String())
	}
	var home, other models.Property
	f.db.First(&home, f.home.ID)
	f.db.First(&other, f.other.ID)
	if home.IsCurrent || !other.IsCurrent {
		t.Errorf("after promotion: home primary %v (want false), other primary %v (want true)", home.IsCurrent, other.IsCurrent)
	}

	// Only the household's admins may add a home to it, and a member can't
	// start another household instead.
	if rec := doJSON(t, f.router, http.MethodPost, "/properties", f.memTok, map[string]any{
		"name": "Sneaky", "type": "owned",
	}); rec.Code != http.StatusForbidden {
		t.Errorf("member adds a home: status %d, want 403", rec.Code)
	}
	if n := f.count(t, &models.HouseholdMember{}, "user_id = ?", f.member.ID); n != 1 {
		t.Errorf("member is in %d households, want 1", n)
	}
}

// Someone in no household starts one with their first home, and a request
// to join another one has nothing left to ask.
func TestCreateProperty_NewHouseholdStartsWithItsPrimaryHome(t *testing.T) {
	f := setupPropertyFixture(t)
	loner := &models.User{Email: "loner@example.com", PasswordHash: "x", Name: "Loner", Role: "user", IsActive: true}
	mustCreate(t, f.db, loner)
	mustCreate(t, f.db, &models.HouseholdJoinRequest{UserID: loner.ID, HouseholdID: f.home.HouseholdID, Status: "pending"})
	rec := doJSON(t, f.router, http.MethodPost, "/properties", testutil.SignToken(t, loner), map[string]any{"name": "Cabin", "type": "owned"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body.String())
	}
	var created models.Property
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.HouseholdID == f.home.HouseholdID || created.HouseholdID == f.other.HouseholdID || !created.IsCurrent {
		t.Errorf("created = household %d primary %v, want a new household with this as its primary", created.HouseholdID, created.IsCurrent)
	}
	var home models.Property
	f.db.First(&home, f.home.ID)
	if !home.IsCurrent {
		t.Error("a new household must not demote another household's primary home")
	}
	var admin models.HouseholdMember
	if err := f.db.Where("household_id = ? AND user_id = ?", created.HouseholdID, loner.ID).First(&admin).Error; err != nil || admin.Role != "admin" {
		t.Errorf("creator membership = %+v (%v), want admin of the new household", admin, err)
	}
	if n := f.count(t, &models.HouseholdJoinRequest{}, "user_id = ?", loner.ID); n != 0 {
		t.Errorf("%d join requests left, want the pending one gone", n)
	}
}

// Archiving keeps a home's history out of everyday use. The primary home is
// always one in use, so it can neither be archived nor be an archived home.
func TestUpdateProperty_Archive(t *testing.T) {
	f := setupPropertyFixture(t)
	f.memberLeaves(t)
	f.dropHomeExpenses(t)
	sibling := models.Property{HouseholdID: f.home.HouseholdID, UserID: f.owner.ID, Name: "Old flat", Type: "rented", StartDate: time.Now(), Residents: 1}
	mustCreate(t, f.db, &sibling)
	put := func(id uint, body map[string]any) *httptest.ResponseRecorder {
		t.Helper()
		return doJSON(t, f.router, http.MethodPut, "/properties/"+itoa(id), f.ownerTok, body)
	}

	if rec := put(f.home.ID, map[string]any{"archived": true}); rec.Code != http.StatusConflict || errorCode(t, rec) != "primary_home_archived" {
		t.Errorf("archive the primary home: status %d, body %s; want 409 primary_home_archived", rec.Code, rec.Body.String())
	}

	if rec := put(sibling.ID, map[string]any{"archived": true}); rec.Code != http.StatusOK {
		t.Fatalf("archive: status %d, body %s", rec.Code, rec.Body.String())
	}
	f.db.First(&sibling, sibling.ID)
	if sibling.EndDate == nil {
		t.Fatal("an archived home has an end date")
	}

	// Listed last, after the homes in use.
	rec := doJSON(t, f.router, http.MethodGet, "/properties", f.ownerTok, nil)
	var props []models.Property
	if err := json.Unmarshal(rec.Body.Bytes(), &props); err != nil {
		t.Fatal(err)
	}
	if len(props) != 2 || props[1].ID != sibling.ID {
		t.Errorf("order = %v, want the archived home last", props)
	}

	if rec := put(sibling.ID, map[string]any{"is_current": true}); rec.Code != http.StatusConflict || errorCode(t, rec) != "archived_home_not_primary" {
		t.Errorf("promote an archived home: status %d, body %s; want 409 archived_home_not_primary", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, f.router, http.MethodDelete, "/properties/"+itoa(f.home.ID)+"?new_principal_id="+itoa(sibling.ID), f.ownerTok, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("archived home as successor: status %d, want 400", rec.Code)
	}

	// Restored and promoted in one go: allowed.
	if rec := put(sibling.ID, map[string]any{"archived": false, "is_current": true}); rec.Code != http.StatusOK {
		t.Fatalf("restore and promote: status %d, body %s", rec.Code, rec.Body.String())
	}
	var restored models.Property // fresh: GORM does not reset a set pointer on NULL
	f.db.First(&restored, sibling.ID)
	if restored.EndDate != nil || !restored.IsCurrent {
		t.Errorf("restored home: end date %v, primary %v; want none and true", restored.EndDate, restored.IsCurrent)
	}
}

// Home expenses are part of the household's ledger: while the household has
// other homes, a home holding any can be archived, not deleted, and the
// balance never moves. Soft-deleted expenses no longer count.
func TestDeleteProperty_HomeWithExpensesIsArchivedInstead(t *testing.T) {
	f := setupPropertyFixture(t)
	f.memberLeaves(t)
	now := time.Now()
	var cat models.Category
	f.db.First(&cat)
	newHome := func(name string) models.Property {
		p := models.Property{HouseholdID: f.home.HouseholdID, UserID: f.owner.ID, Name: name, Type: "owned", StartDate: now, Residents: 1}
		mustCreate(t, f.db, &p)
		return p
	}
	withExpenses, empty, onlyDeleted := newHome("Old flat"), newHome("Garage"), newHome("Cabin")

	exp := models.Expense{UserID: f.owner.ID, HouseholdID: f.home.HouseholdID, PropertyID: &withExpenses.ID, CategoryID: cat.ID, Amount: 30, Date: now, Description: "Boiler", PaidByMemberID: f.mOwner.ID}
	mustCreate(t, f.db, &exp)
	mustCreate(t, f.db,
		&models.ExpenseSplit{ExpenseID: exp.ID, MemberID: f.mOwner.ID, Amount: 15, IsSettled: true, SettledAt: &now},
		&models.ExpenseSplit{ExpenseID: exp.ID, MemberID: f.mMember.ID, Amount: 15},
	)
	gone := models.Expense{UserID: f.owner.ID, HouseholdID: f.home.HouseholdID, PropertyID: &onlyDeleted.ID, CategoryID: cat.ID, Amount: 5, Date: now, Description: "Old", PaidByMemberID: f.mOwner.ID}
	mustCreate(t, f.db, &gone)
	if err := f.db.Delete(&gone).Error; err != nil {
		t.Fatal(err)
	}

	rec := doJSON(t, f.router, http.MethodGet, "/properties", f.ownerTok, nil)
	var props []models.Property
	if err := json.Unmarshal(rec.Body.Bytes(), &props); err != nil {
		t.Fatal(err)
	}
	flags := map[uint]bool{}
	for _, p := range props {
		flags[p.ID] = p.HasExpenses
	}
	for id, want := range map[uint]bool{f.home.ID: true, withExpenses.ID: true, empty.ID: false, onlyDeleted.ID: false} {
		if flags[id] != want {
			t.Errorf("home %d has_expenses = %v, want %v", id, flags[id], want)
		}
	}

	del := func(id uint, query string) *httptest.ResponseRecorder {
		t.Helper()
		return doJSON(t, f.router, http.MethodDelete, "/properties/"+itoa(id)+query, f.ownerTok, nil)
	}
	if rec := del(withExpenses.ID, ""); rec.Code != http.StatusConflict || errorCode(t, rec) != "home_has_expenses" {
		t.Errorf("delete a home with expenses: status %d, body %s; want 409 home_has_expenses", rec.Code, rec.Body.String())
	}
	// Archiving does not make it deletable: archived is where it stays.
	if rec := doJSON(t, f.router, http.MethodPut, "/properties/"+itoa(withExpenses.ID), f.ownerTok, map[string]any{"archived": true}); rec.Code != http.StatusOK {
		t.Fatalf("archive: status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec := del(withExpenses.ID, ""); rec.Code != http.StatusConflict || errorCode(t, rec) != "home_has_expenses" {
		t.Errorf("delete an archived home with expenses: status %d; want 409 home_has_expenses", rec.Code)
	}
	// The primary home is refused for the same reason, successor or not.
	if rec := del(f.home.ID, "?new_principal_id="+itoa(empty.ID)); rec.Code != http.StatusConflict || errorCode(t, rec) != "home_has_expenses" {
		t.Errorf("delete the primary home with expenses: status %d; want 409 home_has_expenses", rec.Code)
	}
	if n := f.count(t, &models.Expense{}, "household_id = ?", f.home.HouseholdID); n != 3 {
		t.Errorf("household expenses: %d, want 3 (none touched)", n)
	}
	if n := f.count(t, &models.ExpenseSplit{}, "expense_id = ?", exp.ID); n != 2 {
		t.Errorf("splits of the refused home's expense: %d, want 2", n)
	}

	for _, p := range []models.Property{empty, onlyDeleted} {
		if rec := del(p.ID, ""); rec.Code != http.StatusOK {
			t.Errorf("delete %s: status %d, body %s; want 200", p.Name, rec.Code, rec.Body.String())
		}
	}
	if n := f.count(t, &models.Expense{}, "id = ?", gone.ID); n != 0 {
		t.Error("a soft-deleted expense goes with its home")
	}
}
