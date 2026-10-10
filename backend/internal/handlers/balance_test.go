package handlers

// Balance endpoint tests. The balance is the number the household reads as
// "who owes whom", so these pin its sign convention (positive = the other
// member owes me), what it nets, and what it must leave out.

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// balanceRouter exposes the balance routes next to the fixture's own router
// (which is what creates expenses and settlements).
func (f *moneyFixture) balanceRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewBalanceHandler(f.db)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.GET("/properties/:id/balance", h.GetBalance)
	p.GET("/properties/:id/balance/details", h.GetBalanceDetails)
	return r
}

func (f *moneyFixture) balanceURL(suffix, query string) string {
	u := "/properties/" + itoa(f.prop.ID) + "/balance" + suffix
	if query != "" {
		u += "?" + query
	}
	return u
}

func (f *moneyFixture) getBalance(t *testing.T, token, query string) BalanceResponse {
	t.Helper()
	rec := doGET(t, f.balanceRouter(), f.balanceURL("", query), token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET balance: status %d, body %s", rec.Code, rec.Body.String())
	}
	var out BalanceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode balance: %v", err)
	}
	return out
}

func (f *moneyFixture) getBalanceDetails(t *testing.T, token, query string) BalanceDetailsResponse {
	t.Helper()
	rec := doGET(t, f.balanceRouter(), f.balanceURL("/details", query), token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET balance details: status %d, body %s", rec.Code, rec.Body.String())
	}
	var out BalanceDetailsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode balance details: %v", err)
	}
	return out
}

func TestBalance_SignFollowsWhoPaid(t *testing.T) {
	f := setupMoneyFixture(t)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID}) // Alice pays, Bob owes his half

	alice := f.getBalance(t, f.aliceTok, "")
	if !approx(alice.Balance, 50) {
		t.Errorf("Alice's balance = %.2f, want +50 (Bob owes her)", alice.Balance)
	}
	if alice.CurrentMemberID != f.mAlice.ID || alice.OtherMemberID != f.mBob.ID {
		t.Errorf("members = %d/%d, want Alice/Bob", alice.CurrentMemberID, alice.OtherMemberID)
	}
	if alice.CurrentMemberName != "Alice" || alice.OtherMemberName == "" {
		t.Errorf("names = %q/%q", alice.CurrentMemberName, alice.OtherMemberName)
	}

	bob := f.getBalance(t, f.bobTok, "")
	if !approx(bob.Balance, -50) {
		t.Errorf("Bob's balance = %.2f, want -50 (he owes Alice)", bob.Balance)
	}
	if bob.OtherMemberID != f.mAlice.ID {
		t.Errorf("Bob's counterpart = %d, want Alice (%d)", bob.OtherMemberID, f.mAlice.ID)
	}
}

func TestBalance_NetsExpensesPaidInBothDirections(t *testing.T) {
	f := setupMoneyFixture(t)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})                         // Bob owes Alice 50
	f.createSplitExpenseBy(t, f.bobTok, f.mBob.ID, 40, []uint{f.mAlice.ID}) // Alice owes Bob 20

	got := f.getBalance(t, f.aliceTok, "")

	if !approx(got.Balance, 30) {
		t.Errorf("balance = %.2f, want 30 (50 owed to Alice minus 20 she owes)", got.Balance)
	}
}

func TestBalance_PartialSettlementShrinksBalance(t *testing.T) {
	f := setupMoneyFixture(t)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 20.0, "date": "2026-05-05",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	if got := f.getBalance(t, f.aliceTok, ""); !approx(got.Balance, 30) {
		t.Errorf("balance after paying 20 of 50 = %.2f, want 30 (settlements must not be counted twice)", got.Balance)
	}
}

func TestBalance_FullySettledIsZero(t *testing.T) {
	f := setupMoneyFixture(t)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 50.0, "date": "2026-05-05",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	got := f.getBalance(t, f.aliceTok, "")

	if !approx(got.Balance, 0) {
		t.Errorf("balance = %.2f, want 0", got.Balance)
	}
}

func TestBalance_LeavesOutLongTermDebts(t *testing.T) {
	f := setupMoneyFixture(t)
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})
	f.markLongTermDebt(t, big)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})

	got := f.getBalance(t, f.aliceTok, "")

	if !approx(got.Balance, 50) {
		t.Errorf("balance = %.2f, want 50: the long-term debt lives in the Debiti ledger, not here", got.Balance)
	}
}

func TestBalance_IgnoresDeletedExpenses(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	if rec := doJSON(t, f.router, http.MethodDelete, "/expenses/"+itoa(id), f.aliceTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("delete expense: status %d, body %s", rec.Code, rec.Body.String())
	}

	if got := f.getBalance(t, f.aliceTok, ""); !approx(got.Balance, 0) {
		t.Errorf("balance = %.2f, want 0 once the expense is deleted", got.Balance)
	}
}

func TestBalance_SplitModeOffReportsZero(t *testing.T) {
	f := setupMoneyFixture(t)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	if err := f.db.Model(&models.HouseholdSettings{}).Where("property_id = ?", f.prop.ID).Update("split_mode", false).Error; err != nil {
		t.Fatalf("disable split mode: %v", err)
	}

	got := f.getBalance(t, f.aliceTok, "")

	if got.Balance != 0 || got.CurrentMemberID != 0 {
		t.Errorf("got %+v, want a zero balance with no members when split mode is off", got)
	}
}

func TestBalance_NoSettingsReportsZero(t *testing.T) {
	f := setupMoneyFixture(t)
	if err := f.db.Where("property_id = ?", f.prop.ID).Delete(&models.HouseholdSettings{}).Error; err != nil {
		t.Fatalf("delete settings: %v", err)
	}

	if got := f.getBalance(t, f.aliceTok, ""); got.Balance != 0 {
		t.Errorf("got %+v, want a zero balance", got)
	}
}

func TestBalance_UserWithoutMemberProfile(t *testing.T) {
	f := setupMoneyFixture(t)
	dave := &models.User{Email: "dave@example.com", PasswordHash: "x", Name: "Dave", Role: "user", IsActive: true}
	mustCreate(t, f.db, dave)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})

	got := f.getBalance(t, testutil.SignToken(t, dave), "")

	if got.Balance != 0 || got.CurrentMemberID != 0 {
		t.Errorf("a user outside the household must see nothing: %+v", got)
	}
}

func TestBalance_NoOtherMember(t *testing.T) {
	f := setupMoneyFixture(t)
	if err := f.db.Where("id != ?", f.mAlice.ID).Delete(&models.HouseholdMember{}).Error; err != nil {
		t.Fatalf("delete members: %v", err)
	}

	got := f.getBalance(t, f.aliceTok, "")

	if got.CurrentMemberID != f.mAlice.ID || got.OtherMemberID != 0 || got.Balance != 0 {
		t.Errorf("got %+v, want Alice alone with a zero balance", got)
	}
}

func TestBalance_ExplicitOtherMember(t *testing.T) {
	f := setupMoneyFixture(t)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	f.createSplitExpense(t, 60, []uint{f.mCarol.ID})

	if got := f.getBalance(t, f.aliceTok, "other_member_id="+itoa(f.mCarol.ID)); !approx(got.Balance, 30) {
		t.Errorf("Alice↔Carol = %.2f, want 30", got.Balance)
	}
	if got := f.getBalance(t, f.aliceTok, "other_member_id="+itoa(f.mBob.ID)); !approx(got.Balance, 50) {
		t.Errorf("Alice↔Bob = %.2f, want 50 (Carol's share must not leak in)", got.Balance)
	}
}

func TestBalance_BadRequests(t *testing.T) {
	f := setupMoneyFixture(t)
	r := f.balanceRouter()

	cases := []struct {
		name, path string
		want       int
	}{
		{"balance: bad property id", "/properties/abc/balance", http.StatusBadRequest},
		{"balance: bad other member id", f.balanceURL("", "other_member_id=xyz"), http.StatusBadRequest},
		{"balance: unknown other member", f.balanceURL("", "other_member_id=99999"), http.StatusNotFound},
		{"details: bad property id", "/properties/abc/balance/details", http.StatusBadRequest},
		{"details: bad other member id", f.balanceURL("/details", "other_member_id=xyz"), http.StatusBadRequest},
		{"details: unknown other member", f.balanceURL("/details", "other_member_id=99999"), http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := doGET(t, r, tc.path, f.aliceTok); rec.Code != tc.want {
				t.Errorf("status %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestBalance_RequiresAuth(t *testing.T) {
	f := setupMoneyFixture(t)
	for _, suffix := range []string{"", "/details"} {
		if rec := doGET(t, f.balanceRouter(), f.balanceURL(suffix, ""), ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("balance%s without a token: status %d, want 401", suffix, rec.Code)
		}
	}
}

func TestBalanceDetails_ListsUnsettledSplitsAndSettlements(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 20.0, "date": "2026-05-05", "note": "acconto",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	got := f.getBalanceDetails(t, f.aliceTok, "")

	if !approx(got.Balance, 30) {
		t.Errorf("balance = %.2f, want 30", got.Balance)
	}
	if len(got.UnsettledSplits) != 1 {
		t.Fatalf("got %d unsettled splits, want 1 (the payer's own share is settled by design)", len(got.UnsettledSplits))
	}
	s := got.UnsettledSplits[0]
	if s.ExpenseID != id || !approx(s.Amount, 50) || !approx(s.SettledAmount, 20) || !approx(s.Remaining, 30) {
		t.Errorf("split = %+v, want expense %d, amount 50, settled 20, remaining 30", s, id)
	}
	if s.PaidByID != f.mAlice.ID || s.PaidByName != "Alice" || s.SplitID == 0 || s.Date != "2026-05-01" {
		t.Errorf("split metadata = %+v", s)
	}
	if len(got.Settlements) != 1 {
		t.Fatalf("got %d settlements, want 1", len(got.Settlements))
	}
	st := got.Settlements[0]
	if st.FromMemberID != f.mBob.ID || st.ToMemberID != f.mAlice.ID || !approx(st.Amount, 20) || st.Note != "acconto" {
		t.Errorf("settlement = %+v", st)
	}
	if st.FromMemberName != "Bob" || st.ToMemberName != "Alice" || st.Date != "2026-05-05" {
		t.Errorf("settlement names/date = %+v", st)
	}
}

func TestBalanceDetails_ExcludesLongTermDebtsAndTargetedPayments(t *testing.T) {
	f := setupMoneyFixture(t)
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})
	f.markLongTermDebt(t, big)
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 500.0, "date": "2026-05-05", "target_expense_id": big,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("targeted payment: status %d, body %s", rec.Code, rec.Body.String())
	}

	got := f.getBalanceDetails(t, f.aliceTok, "")

	if len(got.UnsettledSplits) != 0 || len(got.Settlements) != 0 {
		t.Errorf("the running balance must show neither the debt nor its payments: %+v", got)
	}
	if got.UnsettledSplits == nil || got.Settlements == nil {
		t.Error("empty lists must serialise as [] and not null")
	}
}

func TestBalanceDetails_EdgeStates(t *testing.T) {
	f := setupMoneyFixture(t)
	dave := &models.User{Email: "dave@example.com", PasswordHash: "x", Name: "Dave", Role: "user", IsActive: true}
	mustCreate(t, f.db, dave)

	outsider := f.getBalanceDetails(t, testutil.SignToken(t, dave), "")
	if outsider.CurrentMemberID != 0 || len(outsider.UnsettledSplits) != 0 || outsider.UnsettledSplits == nil {
		t.Errorf("outsider = %+v, want an empty ledger", outsider)
	}

	if err := f.db.Where("id != ?", f.mAlice.ID).Delete(&models.HouseholdMember{}).Error; err != nil {
		t.Fatalf("delete members: %v", err)
	}
	alone := f.getBalanceDetails(t, f.aliceTok, "")
	if alone.CurrentMemberID != f.mAlice.ID || alone.OtherMemberID != 0 || alone.Settlements == nil {
		t.Errorf("alone = %+v, want Alice with no counterpart and empty lists", alone)
	}
}

func TestCalculateBalance_NoSettingsIsZero(t *testing.T) {
	f := setupMoneyFixture(t)
	if err := f.db.Where("property_id = ?", f.prop.ID).Delete(&models.HouseholdSettings{}).Error; err != nil {
		t.Fatalf("delete settings: %v", err)
	}

	got, err := CalculateBalance(f.mAlice.ID, f.mBob.ID, f.prop.ID, f.db)

	if err != nil || got != 0 {
		t.Errorf("got %.2f, %v; want 0, nil", got, err)
	}
}

func TestBalance_OtherMemberMustBelongToTheProperty(t *testing.T) {
	f := setupMoneyFixture(t)
	eve := &models.User{Email: "eve@example.com", PasswordHash: "x", Name: "Eve", Role: "user", IsActive: true}
	mustCreate(t, f.db, eve)
	evesHome := models.Property{UserID: eve.ID, Name: "Eve", Type: "owned", StartDate: time.Now()}
	mustCreate(t, f.db, &evesHome)
	stranger := models.HouseholdMember{PropertyID: evesHome.ID, UserID: &eve.ID, Name: "Eve", Role: "admin"}
	mustCreate(t, f.db, &stranger)
	r := f.balanceRouter()

	for _, suffix := range []string{"", "/details"} {
		rec := doGET(t, r, f.balanceURL(suffix, "other_member_id="+itoa(stranger.ID)), f.aliceTok)
		if rec.Code != http.StatusNotFound {
			t.Errorf("balance%s with another household's member: status %d, want 404 (body %s)", suffix, rec.Code, rec.Body.String())
		}
	}
}
