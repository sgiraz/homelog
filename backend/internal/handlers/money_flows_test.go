package handlers

// Money-flow handler tests. These lock the behaviour of the financially
// sensitive paths — expense splits, settlement, bill/installment auto-expense
// and account deletion — against the in-memory testutil harness.
//
// Helpers itoa() and doGET() live in authorization_test.go (same package).

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// ── helpers ────────────────────────────────────────────────────────────────

func approx(a, b float64) bool { return math.Abs(a-b) < 0.001 }

// doJSON issues an authenticated request with a JSON body.
func doJSON(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		buf = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func wireMoneyRoutes(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	exp := NewExpenseHandler(db)
	settle := NewSettlementHandler(db)
	util := NewUtilityHandler(db)
	settings := NewSettingsHandler(db)
	debt := NewDebtHandler(db)

	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.POST("/expenses", exp.Create)
	p.PUT("/expenses/:id", exp.Update)
	p.DELETE("/expenses/:id", exp.Delete)
	p.PATCH("/expenses/:id/long-term-debt", debt.SetLongTermDebt)
	p.GET("/properties/:id/debts", debt.List)
	p.POST("/settlements", settle.Create)
	p.POST("/settlements/compensate", settle.Compensate)
	p.DELETE("/settlements/:id", settle.Delete)
	p.POST("/utilities/:id/bills", util.AddBill)
	p.PUT("/utilities/:id/bills/:billId", util.UpdateBill)
	p.PATCH("/utilities/:id/bills/:billId/installments/:instId", util.UpdateBillInstallment)
	p.DELETE("/utilities/:id/bills/:billId", util.DeleteBill)
	p.DELETE("/settings/account", settings.DeleteAccount)
	return r
}

// moneyFixture: one property with an admin payer (alice), a registered member
// (bob), and a virtual member (carol), plus the default "Casa" category needed
// by the bill auto-expense path.
type moneyFixture struct {
	router    *gin.Engine
	db        *gorm.DB
	alice     *models.User
	bob       *models.User
	prop      models.Property
	mAlice    models.HouseholdMember
	mBob      models.HouseholdMember
	mCarol    models.HouseholdMember
	casaCatID uint
	aliceTok  string
	bobTok    string
}

func setupMoneyFixture(t *testing.T) *moneyFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	now := time.Now()

	alice := &models.User{Email: "alice@example.com", PasswordHash: "x", Name: "Alice", Role: "admin", IsActive: true}
	bob := &models.User{Email: "bob@example.com", PasswordHash: "x", Name: "Bob", Role: "user", IsActive: true}
	mustCreate(t, db, alice, bob)

	prop := models.Property{UserID: alice.ID, Name: "Casa", Type: "owned", StartDate: now, IsCurrent: true, Residents: 2}
	mustCreate(t, db, &prop)

	mAlice := models.HouseholdMember{PropertyID: prop.ID, UserID: &alice.ID, Name: "Alice", Role: "admin"}
	mBob := models.HouseholdMember{PropertyID: prop.ID, UserID: &bob.ID, Name: "Bob", Role: "member"}
	mCarol := models.HouseholdMember{PropertyID: prop.ID, Name: "Carol", Role: "member", IsVirtual: true}
	mustCreate(t, db, &mAlice, &mBob, &mCarol)

	mustCreate(t, db, &models.HouseholdSettings{PropertyID: prop.ID, SplitMode: true})

	casa := models.Category{Name: "Casa", IsDefault: true}
	mustCreate(t, db, &casa)
	mustCreate(t, db, &models.Subcategory{CategoryID: casa.ID, Name: "Utenze"})

	return &moneyFixture{
		router:    wireMoneyRoutes(db),
		db:        db,
		alice:     alice,
		bob:       bob,
		prop:      prop,
		mAlice:    mAlice,
		mBob:      mBob,
		mCarol:    mCarol,
		casaCatID: casa.ID,
		aliceTok:  testutil.SignToken(t, alice),
		bobTok:    testutil.SignToken(t, bob),
	}
}

func mustCreate(t *testing.T, db *gorm.DB, recs ...any) {
	t.Helper()
	for _, r := range recs {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("create %T: %v", r, err)
		}
	}
}

func (f *moneyFixture) splits(t *testing.T, expenseID uint) []models.ExpenseSplit {
	t.Helper()
	var s []models.ExpenseSplit
	if err := f.db.Where("expense_id = ?", expenseID).Order("member_id ASC").Find(&s).Error; err != nil {
		t.Fatalf("load splits: %v", err)
	}
	return s
}

func (f *moneyFixture) splitFor(t *testing.T, expenseID, memberID uint) models.ExpenseSplit {
	t.Helper()
	var s models.ExpenseSplit
	if err := f.db.Where("expense_id = ? AND member_id = ?", expenseID, memberID).First(&s).Error; err != nil {
		t.Fatalf("split for member %d on expense %d not found: %v", memberID, expenseID, err)
	}
	return s
}

// createSplitExpense posts a split expense paid by mAlice and returns its ID.
func (f *moneyFixture) createSplitExpense(t *testing.T, amount float64, splitWith []uint) uint {
	t.Helper()
	return f.createSplitExpenseBy(t, f.aliceTok, f.mAlice.ID, amount, splitWith)
}

// createSplitExpenseBy posts a split expense paid by an arbitrary member.
func (f *moneyFixture) createSplitExpenseBy(t *testing.T, token string, payerMemberID uint, amount float64, splitWith []uint) uint {
	t.Helper()
	rec := doJSON(t, f.router, http.MethodPost, "/expenses", token, map[string]any{
		"amount":                amount,
		"category_id":           f.casaCatID,
		"property_id":           f.prop.ID,
		"date":                  "2026-05-01",
		"description":           "Spesa condivisa",
		"paid_by_member_id":     payerMemberID,
		"is_split":              true,
		"split_with_member_ids": splitWith,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create split expense: status %d, body %s", rec.Code, rec.Body.String())
	}
	var out models.Expense
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode expense: %v", err)
	}
	return out.ID
}

// markLongTermDebt moves an expense into the long-term debt ledger.
func (f *moneyFixture) markLongTermDebt(t *testing.T, expenseID uint) {
	t.Helper()
	rec := doJSON(t, f.router, http.MethodPatch, "/expenses/"+itoa(expenseID)+"/long-term-debt", f.aliceTok,
		map[string]any{"is_long_term_debt": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("mark long-term debt: status %d, body %s", rec.Code, rec.Body.String())
	}
}

// ── 1. Split create + edit amount ───────────────────────────────────────────

func TestSplitCreate_PayerSelfSplitSettled_QuotasEqual(t *testing.T) {
	f := setupMoneyFixture(t)

	id := f.createSplitExpense(t, 120, []uint{f.mBob.ID, f.mCarol.ID})

	splits := f.splits(t, id)
	if len(splits) != 3 {
		t.Fatalf("expected 3 splits (payer + 2), got %d", len(splits))
	}
	for _, s := range splits {
		if !approx(s.Amount, 40) {
			t.Errorf("split member %d amount = %.4f, want 40", s.MemberID, s.Amount)
		}
	}
	// Payer's own split must be pre-settled; the others must be pending.
	payer := f.splitFor(t, id, f.mAlice.ID)
	if !payer.IsSettled || payer.SettledAt == nil {
		t.Errorf("payer self-split must be settled with SettledAt set, got settled=%v at=%v", payer.IsSettled, payer.SettledAt)
	}
	if f.splitFor(t, id, f.mBob.ID).IsSettled {
		t.Error("bob's split should be unsettled at creation")
	}
	if f.splitFor(t, id, f.mCarol.ID).IsSettled {
		t.Error("carol's split should be unsettled at creation")
	}
}

func TestSplitEditAmount_RecomputesQuotasWhenAllowed(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 120, []uint{f.mBob.ID, f.mCarol.ID})

	rec := doJSON(t, f.router, http.MethodPut, "/expenses/"+itoa(id), f.aliceTok, map[string]any{"amount": 150.0})
	if rec.Code != http.StatusOK {
		t.Fatalf("edit amount: status %d, body %s", rec.Code, rec.Body.String())
	}

	var exp models.Expense
	f.db.First(&exp, id)
	if !approx(exp.Amount, 150) {
		t.Errorf("expense amount = %.2f, want 150", exp.Amount)
	}
	for _, s := range f.splits(t, id) {
		if !approx(s.Amount, 50) {
			t.Errorf("recomputed split member %d = %.4f, want 50", s.MemberID, s.Amount)
		}
	}
}

func TestSplitEditAmount_BlockedWhenNonPayerSplitSettled(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 120, []uint{f.mBob.ID, f.mCarol.ID})

	// Settle bob's share only (carol stays pending → expense not fully settled).
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id":    f.prop.ID,
		"from_member_id": f.mBob.ID,
		"to_member_id":   f.mAlice.ID,
		"amount":         40.0,
		"date":           "2026-05-02",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}
	if !f.splitFor(t, id, f.mBob.ID).IsSettled {
		t.Fatal("precondition: bob's split should be settled after settlement")
	}

	// Now editing the amount must be refused (409) to avoid rewriting balances.
	rec = doJSON(t, f.router, http.MethodPut, "/expenses/"+itoa(id), f.aliceTok, map[string]any{"amount": 300.0})
	if rec.Code != http.StatusConflict {
		t.Fatalf("edit amount with a settled non-payer split: status %d, want 409 (body %s)", rec.Code, rec.Body.String())
	}
	var exp models.Expense
	f.db.First(&exp, id)
	if !approx(exp.Amount, 120) {
		t.Errorf("amount must be unchanged after refused edit, got %.2f", exp.Amount)
	}
}

// ── 2. Settlement ────────────────────────────────────────────────────────────

func TestSettlement_SettlesOnlyOwedSharesAndNeverPayerSelfSplit(t *testing.T) {
	f := setupMoneyFixture(t)

	// Expense A: alice ↔ bob. Expense B: alice ↔ carol (must stay untouched).
	idAB := f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	idAC := f.createSplitExpense(t, 80, []uint{f.mCarol.ID})

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id":    f.prop.ID,
		"from_member_id": f.mBob.ID,
		"to_member_id":   f.mAlice.ID,
		"amount":         50.0,
		"date":           "2026-05-03",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	// Bob's owed share is settled and linked to the settlement.
	bob := f.splitFor(t, idAB, f.mBob.ID)
	if !bob.IsSettled || bob.SettlementID == nil {
		t.Errorf("bob's share must be settled and linked, got settled=%v settlementID=%v", bob.IsSettled, bob.SettlementID)
	}
	// Payer self-splits never go pending — on both expenses.
	if !f.splitFor(t, idAB, f.mAlice.ID).IsSettled {
		t.Error("alice self-split (A) must remain settled")
	}
	if !f.splitFor(t, idAC, f.mAlice.ID).IsSettled {
		t.Error("alice self-split (B) must remain settled")
	}
	// Carol's unrelated share must be untouched.
	if f.splitFor(t, idAC, f.mCarol.ID).IsSettled {
		t.Error("carol's share must remain unsettled — the bob↔alice settlement must not touch it")
	}
}

// ── 2b. Partial settlement ledger ───────────────────────────────────────────
//
// A single outsized expense (e.g. a mortgage down payment) used to create a
// monolithic debt: the boolean is_settled flag meant nothing could be paid
// down except in one lump sum covering the whole share, which then blocked
// settling any other, unrelated expense between the same pair. These tests
// lock the ledger behaviour that replaced it: settlements apply partially,
// oldest outstanding split first (FIFO), and can be reversed precisely.

func TestSettlement_PartialPayment_LeavesSplitUnsettledWithReducedBalance(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 100000, []uint{f.mBob.ID}) // bob owes 50000

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id":    f.prop.ID,
		"from_member_id": f.mBob.ID,
		"to_member_id":   f.mAlice.ID,
		"amount":         500.0,
		"date":           "2026-05-03",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	bob := f.splitFor(t, id, f.mBob.ID)
	if bob.IsSettled {
		t.Error("bob's split must stay unsettled after a payment smaller than the owed amount")
	}
	if !approx(bob.SettledAmount, 500) {
		t.Errorf("bob.SettledAmount = %.2f, want 500", bob.SettledAmount)
	}

	balance, err := CalculateBalance(f.mAlice.ID, f.mBob.ID, f.prop.ID, f.db)
	if err != nil {
		t.Fatalf("CalculateBalance: %v", err)
	}
	if !approx(balance, 49500) {
		t.Errorf("alice's balance = %.2f, want 49500 still owed to her", balance)
	}

	var allocCount int64
	f.db.Model(&models.SettlementAllocation{}).Where("expense_split_id = ?", bob.ID).Count(&allocCount)
	if allocCount != 1 {
		t.Errorf("expected 1 settlement allocation for bob's split, got %d", allocCount)
	}
}

func TestSettlement_MultiplePartialPayments_ApplyFIFOAcrossOldestSplitFirst(t *testing.T) {
	f := setupMoneyFixture(t)
	bigID := f.createSplitExpense(t, 100000, []uint{f.mBob.ID}) // dated 2026-05-01, bob owes 50000

	// First partial payment eats into the big (older) split.
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 500.0, "date": "2026-05-03",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("first settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	smallID := f.createSplitExpense(t, 40, []uint{f.mBob.ID}) // newer, bob owes 20

	rec = doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 200.0, "date": "2026-05-04",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("second settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	if !approx(f.splitFor(t, bigID, f.mBob.ID).SettledAmount, 700) {
		t.Errorf("older/bigger split should absorb payments first (FIFO), got %.2f", f.splitFor(t, bigID, f.mBob.ID).SettledAmount)
	}
	if !approx(f.splitFor(t, smallID, f.mBob.ID).SettledAmount, 0) {
		t.Errorf("newer split should stay untouched while the older one still has room, got %.2f", f.splitFor(t, smallID, f.mBob.ID).SettledAmount)
	}
}

func TestSettlement_Delete_ReversesOnlyItsOwnAllocation(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 100000, []uint{f.mBob.ID}) // bob owes 50000

	rec1 := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 500.0, "date": "2026-05-03",
	})
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first settlement: status %d, body %s", rec1.Code, rec1.Body.String())
	}
	rec2 := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 200.0, "date": "2026-05-04",
	})
	if rec2.Code != http.StatusCreated {
		t.Fatalf("second settlement: status %d, body %s", rec2.Code, rec2.Body.String())
	}
	var s2 models.Settlement
	if err := json.Unmarshal(rec2.Body.Bytes(), &s2); err != nil {
		t.Fatalf("decode second settlement: %v", err)
	}
	if !approx(f.splitFor(t, id, f.mBob.ID).SettledAmount, 700) {
		t.Fatalf("precondition: bob.SettledAmount should be 700 before deletion")
	}

	del := doJSON(t, f.router, http.MethodDelete, "/settlements/"+itoa(s2.ID), f.aliceTok, nil)
	if del.Code != http.StatusOK {
		t.Fatalf("delete settlement: status %d, body %s", del.Code, del.Body.String())
	}

	bob := f.splitFor(t, id, f.mBob.ID)
	if !approx(bob.SettledAmount, 500) {
		t.Errorf("deleting the second settlement should leave only the first's 500, got %.2f", bob.SettledAmount)
	}
	if bob.IsSettled {
		t.Error("bob's split must be unsettled again after reversing a partial payment")
	}
}

func TestSettlement_AmountExceedingOutstanding_Returns400(t *testing.T) {
	f := setupMoneyFixture(t)
	f.createSplitExpense(t, 100000, []uint{f.mBob.ID}) // bob owes 50000

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 60000.0, "date": "2026-05-03",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("settlement exceeding outstanding balance: status %d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
}

// ── 2c. Long-term debts ─────────────────────────────────────────────────────
//
// A mortgage down payment is not an ordinary expense: netted into the running
// balance it swallows everything else, so the debtor keeps fronting groceries
// they never get reimbursed for — their half just nibbles at the big debt.
// Flagged as long-term it moves to its own ledger, repaid deliberately, while
// day-to-day expenses keep splitting and settling normally.

// balance reports what the other member owes the current one (negative: the
// current member owes).
func (f *moneyFixture) balance(t *testing.T, current, other uint) float64 {
	t.Helper()
	b, err := CalculateBalance(current, other, f.prop.ID, f.db)
	if err != nil {
		t.Fatalf("CalculateBalance: %v", err)
	}
	return b
}

func TestLongTermDebt_KeepsOrdinaryExpensesReimbursable(t *testing.T) {
	f := setupMoneyFixture(t)

	// Alice fronts the down payment: Bob's half is 50.000.
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})
	f.markLongTermDebt(t, big)

	if bal := f.balance(t, f.mAlice.ID, f.mBob.ID); !approx(bal, 0) {
		t.Fatalf("running balance after parking the debt = %.2f, want 0", bal)
	}

	// Ordinary life continues: Bob pays the groceries, Alice owes him her half.
	f.createSplitExpenseBy(t, f.bobTok, f.mBob.ID, 100, []uint{f.mAlice.ID})
	if bal := f.balance(t, f.mAlice.ID, f.mBob.ID); !approx(bal, -50) {
		t.Fatalf("Alice should owe Bob 50 for the groceries, got %.2f", bal)
	}

	// ...and she can reimburse him for real, without the 50.000 getting in the way.
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mAlice.ID, "to_member_id": f.mBob.ID,
		"amount": 50.0, "date": "2026-05-05",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ordinary settlement: status %d, body %s", rec.Code, rec.Body.String())
	}
	if bal := f.balance(t, f.mAlice.ID, f.mBob.ID); !approx(bal, 0) {
		t.Errorf("running balance after reimbursement = %.2f, want 0", bal)
	}
	if sa := f.splitFor(t, big, f.mBob.ID).SettledAmount; !approx(sa, 0) {
		t.Errorf("an ordinary settlement must not touch the long-term debt, got settled=%.2f", sa)
	}
}

func TestLongTermDebt_OrdinarySettlementCannotSweepIt(t *testing.T) {
	f := setupMoneyFixture(t)
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})
	f.markLongTermDebt(t, big)

	// Nothing is outstanding in the running balance, so there is nothing to pay.
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 50000.0, "date": "2026-05-05",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("untargeted settlement covering a long-term debt: status %d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestLongTermDebt_TargetedPaymentReducesOnlyThatDebt(t *testing.T) {
	f := setupMoneyFixture(t)
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})
	f.markLongTermDebt(t, big)
	groceries := f.createSplitExpenseBy(t, f.bobTok, f.mBob.ID, 100, []uint{f.mAlice.ID})

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 500.0, "date": "2026-05-05", "target_expense_id": big,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("targeted debt payment: status %d, body %s", rec.Code, rec.Body.String())
	}

	if sa := f.splitFor(t, big, f.mBob.ID).SettledAmount; !approx(sa, 500) {
		t.Errorf("debt settled amount = %.2f, want 500", sa)
	}
	if sa := f.splitFor(t, groceries, f.mAlice.ID).SettledAmount; !approx(sa, 0) {
		t.Errorf("a targeted debt payment must not touch ordinary shares, got settled=%.2f", sa)
	}
	if bal := f.balance(t, f.mAlice.ID, f.mBob.ID); !approx(bal, -50) {
		t.Errorf("running balance = %.2f, want -50 (still owed for the groceries)", bal)
	}
}

func TestLongTermDebt_TargetedPaymentExceedingRemainder_Returns400(t *testing.T) {
	f := setupMoneyFixture(t)
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})
	f.markLongTermDebt(t, big)

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 60000.0, "date": "2026-05-05", "target_expense_id": big,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("overpaying a debt: status %d, want 400 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestCompensation_TurnsACreditIntoDebtRepayment(t *testing.T) {
	f := setupMoneyFixture(t)
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID}) // Bob owes Alice 50.000
	f.markLongTermDebt(t, big)
	groceries := f.createSplitExpenseBy(t, f.bobTok, f.mBob.ID, 100, []uint{f.mAlice.ID}) // Alice owes Bob 50
	source := f.splitFor(t, groceries, f.mAlice.ID)

	rec := doJSON(t, f.router, http.MethodPost, "/settlements/compensate", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "source_split_id": source.ID, "target_expense_id": big,
		"date": "2026-05-06",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("compensation: status %d, body %s", rec.Code, rec.Body.String())
	}
	var settlement models.Settlement
	if err := json.Unmarshal(rec.Body.Bytes(), &settlement); err != nil {
		t.Fatalf("decode settlement: %v", err)
	}
	if !approx(settlement.Amount, 50) {
		t.Errorf("compensation amount = %.2f, want 50 (capped by the credit)", settlement.Amount)
	}

	if !f.splitFor(t, groceries, f.mAlice.ID).IsSettled {
		t.Error("the credit that funded the compensation must be settled")
	}
	if sa := f.splitFor(t, big, f.mBob.ID).SettledAmount; !approx(sa, 50) {
		t.Errorf("debt settled amount = %.2f, want 50", sa)
	}
	if bal := f.balance(t, f.mAlice.ID, f.mBob.ID); !approx(bal, 0) {
		t.Errorf("running balance after compensation = %.2f, want 0", bal)
	}

	// Reversing it restores both sides, not just the debt.
	del := doJSON(t, f.router, http.MethodDelete, "/settlements/"+itoa(settlement.ID), f.bobTok, nil)
	if del.Code != http.StatusOK {
		t.Fatalf("delete compensation: status %d, body %s", del.Code, del.Body.String())
	}
	if f.splitFor(t, groceries, f.mAlice.ID).IsSettled {
		t.Error("the funding credit must be owed again after reversing the compensation")
	}
	if sa := f.splitFor(t, big, f.mBob.ID).SettledAmount; !approx(sa, 0) {
		t.Errorf("debt settled amount after reversal = %.2f, want 0", sa)
	}
}

func TestLongTermDebt_FlagBlockedOnceMoneyMoved(t *testing.T) {
	f := setupMoneyFixture(t)
	id := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 500.0, "date": "2026-05-05",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("partial settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, f.router, http.MethodPatch, "/expenses/"+itoa(id)+"/long-term-debt", f.aliceTok,
		map[string]any{"is_long_term_debt": true})
	if rec.Code != http.StatusConflict {
		t.Fatalf("flagging an already part-paid expense: status %d, want 409 (body %s)", rec.Code, rec.Body.String())
	}
}

func TestDebts_ListReportsRemainderAndPayments(t *testing.T) {
	f := setupMoneyFixture(t)
	big := f.createSplitExpense(t, 100000, []uint{f.mBob.ID})
	f.markLongTermDebt(t, big)

	rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.bobTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 500.0, "date": "2026-05-05", "target_expense_id": big,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("targeted debt payment: status %d, body %s", rec.Code, rec.Body.String())
	}

	// Bob's view: he is the debtor.
	got := doGET(t, f.router, "/properties/"+itoa(f.prop.ID)+"/debts?other_member_id="+itoa(f.mAlice.ID), f.bobTok)
	if got.Code != http.StatusOK {
		t.Fatalf("list debts: status %d, body %s", got.Code, got.Body.String())
	}
	var resp DebtsResponse
	if err := json.Unmarshal(got.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode debts: %v", err)
	}
	if len(resp.Debts) != 1 {
		t.Fatalf("expected 1 debt, got %d", len(resp.Debts))
	}
	d := resp.Debts[0]
	if !d.IOwe {
		t.Error("Bob must be reported as the debtor")
	}
	if !approx(d.Remaining, 49500) {
		t.Errorf("remaining = %.2f, want 49500", d.Remaining)
	}
	if !approx(resp.TotalIOwe, 49500) {
		t.Errorf("total owed = %.2f, want 49500", resp.TotalIOwe)
	}
	if len(d.Payments) != 1 || !approx(d.Payments[0].Amount, 500) {
		t.Errorf("expected one 500 payment in the debt history, got %+v", d.Payments)
	}
}

// ── 3. Bill paid/unpaid auto-expense (single installment) ────────────────────

// seedBill creates a utility + bill + one unpaid installment directly, returning
// the utility, bill and installment. split controls the auto-expense split mode.
func (f *moneyFixture) seedBill(t *testing.T, splitOverride, splitMemberIDs string, installmentBased bool, amount float64) (models.Utility, models.Bill, []models.BillInstallment) {
	t.Helper()
	util := models.Utility{
		UserID: f.alice.ID, PropertyID: f.prop.ID, Type: "electricity", Provider: "Enel",
		IsMetered: true, IsActive: true, IsInstallmentBased: installmentBased,
		PaidByMemberID: &f.mAlice.ID, SplitOverride: splitOverride, SplitMemberIDs: splitMemberIDs,
	}
	mustCreate(t, f.db, &util)
	now := time.Now()
	bill := models.Bill{
		UtilityID: util.ID, BillNumber: "B-001", IssueDate: now,
		PeriodStart: now.AddDate(0, -1, 0), PeriodEnd: now, DueDate: now.AddDate(0, 0, 10),
		AmountTotal: amount,
	}
	mustCreate(t, f.db, &bill)
	return util, bill, nil
}

func TestBillPaid_CreatesSplitExpense_UnpaidDeletesIt(t *testing.T) {
	f := setupMoneyFixture(t)
	util, bill, _ := f.seedBill(t, "custom", "["+itoa(f.mBob.ID)+"]", false, 120)
	inst := models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: bill.DueDate, Amount: 120}
	mustCreate(t, f.db, &inst)

	base := "/utilities/" + itoa(util.ID) + "/bills/" + itoa(bill.ID)

	// is_paid:true → auto-create expense + splits.
	rec := doJSON(t, f.router, http.MethodPut, base, f.aliceTok, map[string]any{"is_paid": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("mark paid: status %d, body %s", rec.Code, rec.Body.String())
	}

	var exp models.Expense
	if err := f.db.Where("bill_id = ?", bill.ID).First(&exp).Error; err != nil {
		t.Fatalf("auto-expense should exist after paying bill: %v", err)
	}
	if exp.BillInstallmentID == nil || *exp.BillInstallmentID != inst.ID {
		t.Errorf("expense.bill_installment_id = %v, want %d", exp.BillInstallmentID, inst.ID)
	}
	if exp.PaidByMemberID != f.mAlice.ID {
		t.Errorf("expense paid_by = %d, want configured payer %d", exp.PaidByMemberID, f.mAlice.ID)
	}
	if !approx(exp.Amount, 120) || !exp.IsSplit {
		t.Errorf("expense amount=%.2f is_split=%v, want 120/true", exp.Amount, exp.IsSplit)
	}
	if s := f.splits(t, exp.ID); len(s) != 2 {
		t.Fatalf("expected 2 splits (alice+bob), got %d", len(s))
	}
	if p := f.splitFor(t, exp.ID, f.mAlice.ID); !p.IsSettled || !approx(p.Amount, 60) {
		t.Errorf("payer split: settled=%v amount=%.2f, want true/60", p.IsSettled, p.Amount)
	}
	if b := f.splitFor(t, exp.ID, f.mBob.ID); b.IsSettled || !approx(b.Amount, 60) {
		t.Errorf("bob split: settled=%v amount=%.2f, want false/60", b.IsSettled, b.Amount)
	}
	var freshBill models.Bill
	f.db.First(&freshBill, bill.ID)
	if !freshBill.IsPaid {
		t.Error("bill.is_paid should be true after paying the only installment")
	}

	// is_paid:false → expense + splits removed, installment unlinked.
	rec = doJSON(t, f.router, http.MethodPut, base, f.aliceTok, map[string]any{"is_paid": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("mark unpaid: status %d, body %s", rec.Code, rec.Body.String())
	}
	var expCount, splitCount int64
	f.db.Model(&models.Expense{}).Where("bill_id = ?", bill.ID).Count(&expCount)
	f.db.Model(&models.ExpenseSplit{}).Where("expense_id = ?", exp.ID).Count(&splitCount)
	if expCount != 0 || splitCount != 0 {
		t.Errorf("after unpaid: expenses=%d splits=%d, want 0/0", expCount, splitCount)
	}
	var freshInst models.BillInstallment
	f.db.First(&freshInst, inst.ID)
	if freshInst.IsPaid || freshInst.ExpenseID != nil {
		t.Errorf("installment after unpaid: is_paid=%v expense_id=%v, want false/nil", freshInst.IsPaid, freshInst.ExpenseID)
	}
}

func TestBillUnpaid_BlockedWhenLinkedSplitSettled(t *testing.T) {
	f := setupMoneyFixture(t)
	util, bill, _ := f.seedBill(t, "custom", "["+itoa(f.mBob.ID)+"]", false, 120)
	inst := models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: bill.DueDate, Amount: 120}
	mustCreate(t, f.db, &inst)
	base := "/utilities/" + itoa(util.ID) + "/bills/" + itoa(bill.ID)

	if rec := doJSON(t, f.router, http.MethodPut, base, f.aliceTok, map[string]any{"is_paid": true}); rec.Code != http.StatusOK {
		t.Fatalf("mark paid: status %d, body %s", rec.Code, rec.Body.String())
	}
	// Settle bob's share against alice → bill becomes locked.
	if rec := doJSON(t, f.router, http.MethodPost, "/settlements", f.aliceTok, map[string]any{
		"property_id": f.prop.ID, "from_member_id": f.mBob.ID, "to_member_id": f.mAlice.ID,
		"amount": 60.0, "date": "2026-05-04",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}

	rec := doJSON(t, f.router, http.MethodPut, base, f.aliceTok, map[string]any{"is_paid": false})
	if rec.Code != http.StatusConflict {
		t.Fatalf("unpaying a locked bill: status %d, want 409 (body %s)", rec.Code, rec.Body.String())
	}
	var expCount int64
	f.db.Model(&models.Expense{}).Where("bill_id = ?", bill.ID).Count(&expCount)
	if expCount != 1 {
		t.Errorf("locked bill's expense must survive the refused unpay, got %d expenses", expCount)
	}
}

// ── 4. Installment paid/unpaid (per-rata, BillInstallmentID) ─────────────────

func TestInstallmentPaidUnpaid_PerRataExpenseLifecycle(t *testing.T) {
	f := setupMoneyFixture(t)
	util, bill, _ := f.seedBill(t, "no_split", "", true, 100)
	inst1 := models.BillInstallment{BillID: bill.ID, Number: 1, DueDate: bill.DueDate, Amount: 50}
	inst2 := models.BillInstallment{BillID: bill.ID, Number: 2, DueDate: bill.DueDate.AddDate(0, 1, 0), Amount: 50}
	mustCreate(t, f.db, &inst1, &inst2)

	instPath := func(id uint) string {
		return "/utilities/" + itoa(util.ID) + "/bills/" + itoa(bill.ID) + "/installments/" + itoa(id)
	}

	// Pay installment 1 → one expense linked to inst1; bill not fully paid yet.
	if rec := doJSON(t, f.router, http.MethodPatch, instPath(inst1.ID), f.aliceTok, map[string]any{"is_paid": true}); rec.Code != http.StatusOK {
		t.Fatalf("pay inst1: status %d, body %s", rec.Code, rec.Body.String())
	}
	var exp1 models.Expense
	if err := f.db.Where("bill_installment_id = ?", inst1.ID).First(&exp1).Error; err != nil {
		t.Fatalf("expense for inst1 should exist: %v", err)
	}
	if !approx(exp1.Amount, 50) || exp1.IsSplit {
		t.Errorf("inst1 expense amount=%.2f split=%v, want 50/false", exp1.Amount, exp1.IsSplit)
	}
	var b models.Bill
	f.db.First(&b, bill.ID)
	if b.IsPaid {
		t.Error("bill must not be fully paid with inst2 still open")
	}

	// Pay installment 2 → bill fully paid.
	if rec := doJSON(t, f.router, http.MethodPatch, instPath(inst2.ID), f.aliceTok, map[string]any{"is_paid": true}); rec.Code != http.StatusOK {
		t.Fatalf("pay inst2: status %d, body %s", rec.Code, rec.Body.String())
	}
	f.db.First(&b, bill.ID)
	if !b.IsPaid {
		t.Error("bill should be fully paid once every installment is paid")
	}
	var total int64
	f.db.Model(&models.Expense{}).Where("bill_id = ?", bill.ID).Count(&total)
	if total != 2 {
		t.Fatalf("expected 2 auto-expenses (one per rata), got %d", total)
	}

	// Unpay installment 1 → only its expense disappears; inst2's survives.
	if rec := doJSON(t, f.router, http.MethodPatch, instPath(inst1.ID), f.aliceTok, map[string]any{"is_paid": false}); rec.Code != http.StatusOK {
		t.Fatalf("unpay inst1: status %d, body %s", rec.Code, rec.Body.String())
	}
	var c1, c2 int64
	f.db.Model(&models.Expense{}).Where("bill_installment_id = ?", inst1.ID).Count(&c1)
	f.db.Model(&models.Expense{}).Where("bill_installment_id = ?", inst2.ID).Count(&c2)
	if c1 != 0 || c2 != 1 {
		t.Errorf("after unpaying inst1: inst1 expenses=%d (want 0), inst2 expenses=%d (want 1)", c1, c2)
	}
	f.db.First(&b, bill.ID)
	if b.IsPaid {
		t.Error("bill must drop back to unpaid after a rata is unpaid")
	}
}

// ── 5. Delete account ────────────────────────────────────────────────────────

type accountFixture struct {
	router *gin.Engine
	db     *gorm.DB
	user   *models.User
	prop   models.Property
	tok    string
}

const acctPassword = "correct-horse-battery"

// setupAccountFixture builds a property owned solely by the test user, populated
// with a utility, bill and a solo expense (so the FTS index has rows to clean).
// When withSecondMember is true a registered non-admin member is added, which
// must block self-deletion.
func setupAccountFixture(t *testing.T, withSecondMember bool) *accountFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	now := time.Now()

	hash, err := bcrypt.GenerateFromPassword([]byte(acctPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	user := &models.User{Email: "owner@example.com", PasswordHash: string(hash), Name: "Owner", Role: "admin", IsActive: true}
	mustCreate(t, db, user)

	prop := models.Property{UserID: user.ID, Name: "Casa", Type: "owned", StartDate: now, IsCurrent: true, Residents: 1}
	mustCreate(t, db, &prop)
	member := models.HouseholdMember{PropertyID: prop.ID, UserID: &user.ID, Name: "Owner", Role: "admin"}
	mustCreate(t, db, &member)

	casa := models.Category{Name: "Casa", IsDefault: true}
	mustCreate(t, db, &casa)

	// Utility + bill (both index into search_index via AfterSave hooks).
	util := models.Utility{UserID: user.ID, PropertyID: prop.ID, Type: "gas", Provider: "Eni", IsMetered: true, IsActive: true}
	mustCreate(t, db, &util)
	bill := models.Bill{UtilityID: util.ID, BillNumber: "G-1", IssueDate: now, PeriodStart: now, PeriodEnd: now, DueDate: now, AmountTotal: 50}
	mustCreate(t, db, &bill)
	// A solo expense owned by the user.
	mustCreate(t, db, &models.Expense{UserID: user.ID, PropertyID: &prop.ID, CategoryID: casa.ID, Amount: 30, Date: now, Description: "Spesa", PaidByMemberID: member.ID})

	if withSecondMember {
		other := &models.User{Email: "second@example.com", PasswordHash: "x", Name: "Second", Role: "user", IsActive: true}
		mustCreate(t, db, other)
		mustCreate(t, db, &models.HouseholdMember{PropertyID: prop.ID, UserID: &other.ID, Name: "Second", Role: "member"})
	}

	return &accountFixture{router: wireMoneyRoutes(db), db: db, user: user, prop: prop, tok: testutil.SignToken(t, user)}
}

func (a *accountFixture) searchIndexCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := a.db.Raw("SELECT count(*) FROM search_index").Scan(&n).Error; err != nil {
		t.Fatalf("count search_index: %v", err)
	}
	return n
}

func TestDeleteAccount_CascadesAndCleansSearchIndex(t *testing.T) {
	a := setupAccountFixture(t, false)

	if a.searchIndexCount(t) == 0 {
		t.Fatal("precondition: search_index should hold the seeded utility/bill/expense rows")
	}

	rec := doJSON(t, a.router, http.MethodDelete, "/settings/account", a.tok, map[string]any{"password": acctPassword})
	if rec.Code != http.StatusOK {
		t.Fatalf("delete account: status %d, body %s", rec.Code, rec.Body.String())
	}

	// User and every owned record must be gone (Unscoped — hard deleted).
	type check struct {
		name  string
		model any
	}
	for _, c := range []check{
		{"users", &models.User{}},
		{"properties", &models.Property{}},
		{"household_members", &models.HouseholdMember{}},
		{"utilities", &models.Utility{}},
		{"bills", &models.Bill{}},
		{"expenses", &models.Expense{}},
		{"expense_splits", &models.ExpenseSplit{}},
		{"settlements", &models.Settlement{}},
	} {
		var n int64
		a.db.Unscoped().Model(c.model).Count(&n)
		if n != 0 {
			t.Errorf("%s: %d rows remain after account deletion, want 0", c.name, n)
		}
	}
	if n := a.searchIndexCount(t); n != 0 {
		t.Errorf("search_index: %d rows remain, want 0 (orphaned FTS entries)", n)
	}
}

func TestDeleteAccount_WrongPassword_401(t *testing.T) {
	a := setupAccountFixture(t, false)

	rec := doJSON(t, a.router, http.MethodDelete, "/settings/account", a.tok, map[string]any{"password": "nope"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: status %d, want 401 (body %s)", rec.Code, rec.Body.String())
	}
	var n int64
	a.db.Model(&models.User{}).Where("id = ?", a.user.ID).Count(&n)
	if n != 1 {
		t.Error("user must still exist after a failed deletion")
	}
}

func TestDeleteAccount_BlockedWhenSoleAdminWithMembers_409(t *testing.T) {
	a := setupAccountFixture(t, true)

	rec := doJSON(t, a.router, http.MethodDelete, "/settings/account", a.tok, map[string]any{"password": acctPassword})
	if rec.Code != http.StatusConflict {
		t.Fatalf("sole admin with other members: status %d, want 409 (body %s)", rec.Code, rec.Body.String())
	}
	var n int64
	a.db.Model(&models.User{}).Where("id = ?", a.user.ID).Count(&n)
	if n != 1 {
		t.Error("user must still exist while deletion is blocked")
	}
}
