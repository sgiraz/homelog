package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

type joinFixture struct {
	router    *gin.Engine
	db        *gorm.DB
	home      models.Property
	stranger  *models.User
	ownerTok  string
	memberTok string
	strTok    string
}

// A household with an admin, a plain member and one home; a stranger outside it.
func setupJoinFixture(t *testing.T) *joinFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)

	owner := &models.User{Email: "owner@example.com", PasswordHash: "x", Name: "Owner", Role: "admin", IsActive: true}
	member := &models.User{Email: "member@example.com", PasswordHash: "x", Name: "Member", Role: "user", IsActive: true}
	stranger := &models.User{Email: "stranger@example.com", PasswordHash: "x", Name: "Stranger", Role: "user", IsActive: true}
	mustCreate(t, db, owner, member, stranger)

	home := models.Property{Household: &models.Household{Name: "Main house"}, UserID: owner.ID, Name: "Main house", Address: "1 Oak Road", Type: "owned", StartDate: time.Now(), IsCurrent: true, Residents: 2}
	mustCreate(t, db, &home)
	mustCreate(t, db,
		&models.HouseholdMember{HouseholdID: home.HouseholdID, UserID: &owner.ID, Name: "Owner", Role: "admin"},
		&models.HouseholdMember{HouseholdID: home.HouseholdID, UserID: &member.ID, Name: "Member", Role: "member"},
	)

	gin.SetMode(gin.TestMode)
	h := NewJoinRequestHandler(db)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.GET("/households/joinable", h.ListJoinable)
	p.POST("/join-requests", h.Create)
	p.GET("/join-requests", h.List)
	p.PATCH("/join-requests/:id", h.Resolve)

	return &joinFixture{
		router: r, db: db, home: home, stranger: stranger,
		ownerTok: testutil.SignToken(t, owner), memberTok: testutil.SignToken(t, member), strTok: testutil.SignToken(t, stranger),
	}
}

func (f *joinFixture) request(t *testing.T) models.HouseholdJoinRequest {
	t.Helper()
	rec := doJSON(t, f.router, http.MethodPost, "/join-requests", f.strTok, map[string]any{"household_id": f.home.HouseholdID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: status %d, body %s", rec.Code, rec.Body.String())
	}
	var jr models.HouseholdJoinRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &jr); err != nil {
		t.Fatal(err)
	}
	return jr
}

func (f *joinFixture) joinable(t *testing.T, tok string) []JoinableHouseholdResponse {
	t.Helper()
	rec := doJSON(t, f.router, http.MethodGet, "/households/joinable", tok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("joinable: status %d, body %s", rec.Code, rec.Body.String())
	}
	var list []JoinableHouseholdResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	return list
}

// Joining is the only way into a household's expenses and balance: only one
// of its admins can let someone in, once, and the newcomer becomes a member.
func TestJoinRequest_OnlyAnAdminLetsSomeoneIn(t *testing.T) {
	f := setupJoinFixture(t)

	list := f.joinable(t, f.strTok)
	if len(list) != 1 || list[0].ID != f.home.HouseholdID || list[0].Address != "1 Oak Road" || list[0].MemberCount != 2 {
		t.Fatalf("joinable for the stranger = %+v, want the household with its primary home's address and 2 members", list)
	}
	if got := f.joinable(t, f.memberTok); len(got) != 0 {
		t.Errorf("joinable for a member = %+v, want none: they are in already", got)
	}

	jr := f.request(t)
	rec := doJSON(t, f.router, http.MethodPost, "/join-requests", f.strTok, map[string]any{"household_id": f.home.HouseholdID})
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "join_request_pending" {
		t.Errorf("second request: status %d, body %s; want 409 join_request_pending", rec.Code, rec.Body.String())
	}
	if got := f.joinable(t, f.strTok); len(got) != 0 {
		t.Errorf("joinable with a pending request = %+v, want none", got)
	}

	path := "/join-requests/" + itoa(jr.ID)
	approve := map[string]any{"status": "approved"}
	for name, tok := range map[string]string{"the requester": f.strTok, "a plain member": f.memberTok} {
		if rec := doJSON(t, f.router, http.MethodPatch, path, tok, approve); rec.Code != http.StatusForbidden || errorCode(t, rec) != "household_admin_only" {
			t.Errorf("approval by %s: status %d, body %s; want 403 household_admin_only", name, rec.Code, rec.Body.String())
		}
	}
	if n := f.memberCount(t); n != 2 {
		t.Fatalf("members after refused approvals: %d, want 2", n)
	}

	rec = doJSON(t, f.router, http.MethodGet, "/join-requests", f.ownerTok, nil)
	var pending []models.HouseholdJoinRequest
	if err := json.Unmarshal(rec.Body.Bytes(), &pending); err != nil || len(pending) != 1 || pending[0].ID != jr.ID {
		t.Fatalf("admin's list = %s, want the pending request", rec.Body.String())
	}

	if rec := doJSON(t, f.router, http.MethodPatch, path, f.ownerTok, approve); rec.Code != http.StatusOK {
		t.Fatalf("approval by the admin: status %d, body %s", rec.Code, rec.Body.String())
	}
	var joined models.HouseholdMember
	if err := f.db.Where("household_id = ? AND user_id = ?", f.home.HouseholdID, f.stranger.ID).First(&joined).Error; err != nil {
		t.Fatalf("approved user is not a member: %v", err)
	}
	if joined.IsVirtual || joined.Role != "member" || joined.Name != "Stranger" {
		t.Errorf("new member = virtual %v, role %q, name %q; want a registered member named after the account", joined.IsVirtual, joined.Role, joined.Name)
	}
	var home models.Property
	f.db.First(&home, f.home.ID)
	if home.Residents != 3 {
		t.Errorf("residents = %d, want 3", home.Residents)
	}
	var stored models.HouseholdJoinRequest
	f.db.First(&stored, jr.ID)
	if stored.Status != "approved" || stored.ResolvedBy == nil || stored.ResolvedAt == nil {
		t.Errorf("request = status %q, resolved_by %v, resolved_at %v; want approved with resolver and time", stored.Status, stored.ResolvedBy, stored.ResolvedAt)
	}

	if rec := doJSON(t, f.router, http.MethodPatch, path, f.ownerTok, approve); rec.Code != http.StatusConflict || errorCode(t, rec) != "join_request_resolved" {
		t.Errorf("approving twice: status %d; want 409 join_request_resolved", rec.Code)
	}
	if n := f.memberCount(t); n != 3 {
		t.Errorf("members = %d, want 3 (one approval, one member)", n)
	}
	rec = doJSON(t, f.router, http.MethodPost, "/join-requests", f.strTok, map[string]any{"household_id": f.home.HouseholdID})
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "already_in_household" {
		t.Errorf("request once a member: status %d, body %s; want 409 already_in_household", rec.Code, rec.Body.String())
	}
}

// An account belongs to one household: someone in one can't ask to join
// another, and an approval can't add someone who found another meanwhile.
func TestJoinRequest_OneHouseholdPerAccount(t *testing.T) {
	f := setupJoinFixture(t)
	neighbour := &models.User{Email: "neighbour@example.com", PasswordHash: "x", Name: "Neighbour", Role: "user", IsActive: true}
	mustCreate(t, f.db, neighbour)
	flat := models.Property{Household: &models.Household{Name: "Flat"}, UserID: neighbour.ID, Name: "Flat", Type: "rented", StartDate: time.Now(), IsCurrent: true, Residents: 1}
	mustCreate(t, f.db, &flat)
	mustCreate(t, f.db, &models.HouseholdMember{HouseholdID: flat.HouseholdID, UserID: &neighbour.ID, Name: "Neighbour", Role: "admin"})

	rec := doJSON(t, f.router, http.MethodPost, "/join-requests", f.memberTok, map[string]any{"household_id": flat.HouseholdID})
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "already_in_household" {
		t.Errorf("a member asks to join another household: status %d, body %s; want 409 already_in_household", rec.Code, rec.Body.String())
	}

	jr := f.request(t)
	mustCreate(t, f.db, &models.HouseholdMember{HouseholdID: flat.HouseholdID, UserID: &f.stranger.ID, Name: "Stranger", Role: "member"})
	rec = doJSON(t, f.router, http.MethodPatch, "/join-requests/"+itoa(jr.ID), f.ownerTok, map[string]any{"status": "approved"})
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "requester_in_household" {
		t.Errorf("approval after the requester joined another household: status %d, body %s; want 409 requester_in_household", rec.Code, rec.Body.String())
	}
	if n := f.memberCount(t); n != 2 {
		t.Errorf("members = %d, want 2: nobody added", n)
	}
	var stored models.HouseholdJoinRequest
	f.db.First(&stored, jr.ID)
	if stored.Status != "pending" {
		t.Errorf("request status = %q, want still pending after a refused approval", stored.Status)
	}
}

func TestJoinRequest_RejectionLetsNobodyIn(t *testing.T) {
	f := setupJoinFixture(t)
	jr := f.request(t)

	if rec := doJSON(t, f.router, http.MethodPatch, "/join-requests/"+itoa(jr.ID), f.ownerTok, map[string]any{"status": "rejected"}); rec.Code != http.StatusOK {
		t.Fatalf("reject: status %d, body %s", rec.Code, rec.Body.String())
	}
	if n := f.memberCount(t); n != 2 {
		t.Errorf("members after a rejection: %d, want 2", n)
	}
	var home models.Property
	f.db.First(&home, f.home.ID)
	if home.Residents != 2 {
		t.Errorf("residents after a rejection = %d, want 2", home.Residents)
	}

	rec := doJSON(t, f.router, http.MethodPost, "/join-requests", f.strTok, map[string]any{"household_id": 99999})
	if rec.Code != http.StatusNotFound || errorCode(t, rec) != "household_not_found" {
		t.Errorf("request to a missing household: status %d, body %s; want 404 household_not_found", rec.Code, rec.Body.String())
	}
}

func (f *joinFixture) memberCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := f.db.Model(&models.HouseholdMember{}).Where("household_id = ?", f.home.HouseholdID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}
