package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

type propertyFixture struct {
	router   *gin.Engine
	db       *gorm.DB
	owner    *models.User
	member   *models.User
	home     models.Property // principal, shared with member, full of data
	other    models.Property // second home of owner, older
	ownerTok string
	memTok   string
}

func setupPropertyFixture(t *testing.T) *propertyFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	now := time.Now()

	owner := &models.User{Email: "owner@example.com", PasswordHash: "x", Name: "Owner", Role: "admin", IsActive: true}
	member := &models.User{Email: "member@example.com", PasswordHash: "x", Name: "Member", Role: "user", IsActive: true}
	mustCreate(t, db, owner, member)

	other := models.Property{UserID: owner.ID, Name: "Seaside flat", Type: "rented", StartDate: now, Residents: 1}
	mustCreate(t, db, &other)
	mOther := models.HouseholdMember{PropertyID: other.ID, UserID: &owner.ID, Name: "Owner", Role: "admin"}
	mustCreate(t, db, &mOther)

	home := models.Property{UserID: owner.ID, Name: "Main house", Type: "owned", StartDate: now, IsCurrent: true, Residents: 2}
	mustCreate(t, db, &home)
	mOwner := models.HouseholdMember{PropertyID: home.ID, UserID: &owner.ID, Name: "Owner", Role: "admin"}
	mMember := models.HouseholdMember{PropertyID: home.ID, UserID: &member.ID, Name: "Member", Role: "member"}
	mustCreate(t, db, &mOwner, &mMember, &models.HouseholdSettings{PropertyID: home.ID})

	cat := models.Category{Name: "Groceries"}
	mustCreate(t, db, &cat)

	reading := 100.0
	util := models.Utility{UserID: owner.ID, PropertyID: home.ID, Type: "gas", Provider: "Acme Gas", IsMetered: true, IsActive: true}
	mustCreate(t, db, &util)
	mustCreate(t, db,
		&models.Bill{UtilityID: util.ID, BillNumber: "G-1", IssueDate: now, PeriodStart: now, PeriodEnd: now, DueDate: now, AmountTotal: 50},
		&models.MeterReading{UtilityID: util.ID, ReadingDate: now, Value: &reading},
	)
	project := models.Project{UserID: owner.ID, PropertyID: &home.ID, Name: "Kitchen", Budget: 1000, StartDate: now, EndDate: now}
	mustCreate(t, db, &project)
	mustCreate(t, db, &models.ProjectMember{ProjectID: project.ID, UserID: member.ID, Role: "member"})
	exp := models.Expense{UserID: owner.ID, PropertyID: &home.ID, ProjectID: &project.ID, CategoryID: cat.ID, Amount: 40, Date: now, Description: "Tiles", PaidByMemberID: mOwner.ID}
	mustCreate(t, db, &exp)
	mustCreate(t, db,
		&models.ExpenseSplit{ExpenseID: exp.ID, MemberID: mOwner.ID, Amount: 20, IsSettled: true, SettledAt: &now},
		&models.ExpenseSplit{ExpenseID: exp.ID, MemberID: mMember.ID, Amount: 20},
		&models.Settlement{PropertyID: home.ID, FromMemberID: mMember.ID, ToMemberID: mOwner.ID, Amount: 5, Date: now},
		&models.Notification{UserID: member.ID, Type: "expense_shared", Title: "Shared", PropertyID: &home.ID},
	)

	// A record on the other home that must survive.
	mustCreate(t, db, &models.Expense{UserID: owner.ID, PropertyID: &other.ID, CategoryID: cat.ID, Amount: 10, Date: now, Description: "Rent", PaidByMemberID: mOther.ID})

	gin.SetMode(gin.TestMode)
	h := NewPropertyHandler(db)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.GET("/properties", h.List)
	p.PUT("/properties/:id", h.Update)
	p.DELETE("/properties/:id", h.Delete)

	return &propertyFixture{
		router: r, db: db, owner: owner, member: member, home: home, other: other,
		ownerTok: testutil.SignToken(t, owner), memTok: testutil.SignToken(t, member),
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

func TestDeleteProperty_CascadesAndPromotesChosenPrincipal(t *testing.T) {
	f := setupPropertyFixture(t)
	id := f.home.ID

	var before int64
	if err := f.db.Raw("SELECT COUNT(*) FROM search_index WHERE property_id = ?", id).Scan(&before).Error; err != nil || before == 0 {
		t.Fatalf("precondition: search_index should hold the home's rows (n=%d, err=%v)", before, err)
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
		{"properties", &models.Property{}, "id = ?"},
		{"household_members", &models.HouseholdMember{}, "property_id = ?"},
		{"household_settings", &models.HouseholdSettings{}, "property_id = ?"},
		{"utilities", &models.Utility{}, "property_id = ?"},
		{"expenses", &models.Expense{}, "property_id = ?"},
		{"projects", &models.Project{}, "property_id = ?"},
		{"settlements", &models.Settlement{}, "property_id = ?"},
		{"notifications", &models.Notification{}, "property_id = ?"},
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
		t.Error("the chosen home should have been promoted to principal")
	}
}

func TestDeleteProperty_LastOneIsRefused(t *testing.T) {
	f := setupPropertyFixture(t)
	if rec := doJSON(t, f.router, http.MethodDelete, "/properties/"+itoa(f.other.ID), f.ownerTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("delete first home: status %d, body %s", rec.Code, rec.Body.String())
	}

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
	path := "/properties/" + itoa(f.other.ID)

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
	f.db.First(&got, f.other.ID)
	if got.Name != "Lake house" || got.Address != "1 Shore Rd" || got.Type != "owned" || got.Residents != 3 || !got.IsCurrent {
		t.Errorf("updated property = %+v", got)
	}
	var home models.Property
	f.db.First(&home, f.home.ID)
	if home.IsCurrent {
		t.Error("the previous principal should have been unset")
	}

	if rec := doJSON(t, f.router, http.MethodPut, path, f.memTok, map[string]any{"name": "Mine"}); rec.Code != http.StatusNotFound {
		t.Errorf("non-owner update: status %d, want 404", rec.Code)
	}
}

func TestDeleteProperty_PrincipalNeedsAValidSuccessor(t *testing.T) {
	f := setupPropertyFixture(t)
	foreign := models.Property{UserID: f.member.ID, Name: "Member's flat", Type: "rented", StartDate: time.Now(), Residents: 1}
	mustCreate(t, f.db, &foreign)
	base := "/properties/" + itoa(f.home.ID)

	for name, c := range map[string]struct {
		query, code string
	}{
		"missing":      {"", "new_principal_required"},
		"itself":       {"?new_principal_id=" + itoa(f.home.ID), "new_principal_invalid"},
		"not owned":    {"?new_principal_id=" + itoa(foreign.ID), "new_principal_invalid"},
		"not a number": {"?new_principal_id=abc", "new_principal_invalid"},
	} {
		rec := doJSON(t, f.router, http.MethodDelete, base+c.query, f.ownerTok, nil)
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != c.code {
			t.Errorf("%s: status %d, body %s; want 400 %s", name, rec.Code, rec.Body.String(), c.code)
		}
	}
	if n := f.count(t, &models.Property{}, "id = ?", f.home.ID); n != 1 {
		t.Error("a refused delete must leave the primary home in place")
	}
}
