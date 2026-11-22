package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// TestDeleteAccount_CoversSchema fails on any foreign key into users that
// deleteAccount does not handle. Adding one means teaching deleteAccount what
// becomes of those rows, then listing the key here.
func TestDeleteAccount_CoversSchema(t *testing.T) {
	db := testutil.NewDB(t)
	handled := map[string]string{
		"bill_templates.user_id":          "kept, without an author",
		"categories.user_id":              "deleted, or kept without an owner while the ledger uses it",
		"contract_templates.user_id":      "deleted",
		"expense_templates.user_id":       "deleted",
		"projects.user_id":                "handed to another project member, or deleted",
		"properties.user_id":              "handed to the successor, or deleted with the home",
		"user_settings.user_id":           "deleted",
		"household_members.user_id":       "turned virtual while the ledger refers to it, else deleted",
		"notifications.user_id":           "deleted",
		"project_members.user_id":         "deleted, after ownership moved on",
		"household_join_requests.user_id": "deleted",
	}

	var edges []string
	if err := db.Raw(`SELECT m.name || '.' || f."from" FROM sqlite_master m, pragma_foreign_key_list(m.name) f
		WHERE m.type = 'table' AND f."table" = 'users'`).Scan(&edges).Error; err != nil {
		t.Fatal(err)
	}
	if len(edges) == 0 {
		t.Fatal("no foreign key into users found: the scan is broken")
	}
	seen := map[string]bool{}
	var missing []string
	for _, edge := range edges {
		seen[edge] = true
		if _, ok := handled[edge]; !ok {
			missing = append(missing, edge)
		}
	}
	sort.Strings(missing)
	for _, edge := range missing {
		t.Errorf("deleteAccount does not handle %s -> users", edge)
	}
	for edge := range handled {
		if !seen[edge] {
			t.Errorf("handled edge %s no longer exists in the schema: drop it from the list", edge)
		}
	}
}

// accountRoutes adds the account endpoints to the money fixture's router;
// alice is the instance admin.
func (f *moneyFixture) accountRoutes(t *testing.T) {
	t.Helper()
	settings := NewSettingsHandler(f.db)
	admin := NewAdminHandler(f.db)
	g := f.router.Group("", middleware.AuthRequired())
	g.GET("/settings/account/delete-check", settings.DeleteAccountCheck)
	g.POST("/settings/account/promote-admin", settings.PromoteAdmin)
	g.DELETE("/admin/users/:id", admin.DeleteUser)

	hash, err := bcrypt.GenerateFromPassword([]byte(acctPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&models.User{}).Where("id IN ?", []uint{f.alice.ID, f.bob.ID}).
		Update("password_hash", string(hash)).Error; err != nil {
		t.Fatal(err)
	}
}

func (f *moneyFixture) settle(t *testing.T, token string, from, to uint, amount float64) {
	t.Helper()
	rec := doJSON(t, f.router, http.MethodPost, "/settlements", token, map[string]any{
		"household_id": f.prop.HouseholdID, "from_member_id": from, "to_member_id": to, "amount": amount, "date": "2026-05-03",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("settlement: status %d, body %s", rec.Code, rec.Body.String())
	}
}

func (f *moneyFixture) deleteOwnAccount(t *testing.T, token string) int {
	t.Helper()
	rec := doJSON(t, f.router, http.MethodDelete, "/settings/account", token, map[string]any{"password": acctPassword})
	if rec.Code != http.StatusOK && rec.Code != http.StatusConflict {
		t.Fatalf("delete account: status %d, body %s", rec.Code, rec.Body.String())
	}
	return rec.Code
}

func assertNoForeignKeyViolations(t *testing.T, db *gorm.DB) {
	t.Helper()
	var rows []struct {
		Table  string `gorm:"column:table"`
		Parent string `gorm:"column:parent"`
	}
	if err := db.Raw(`SELECT "table", "parent" FROM pragma_foreign_key_check`).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		t.Errorf("orphaned row: %s -> %s", r.Table, r.Parent)
	}
}

// ledger fingerprints what the household's balances are made of.
type ledger struct {
	Splits      []models.ExpenseSplit
	Settlements []models.Settlement
	Allocations []models.SettlementAllocation
}

func readLedger(t *testing.T, db *gorm.DB) ledger {
	t.Helper()
	var l ledger
	for _, q := range []error{
		db.Order("id").Find(&l.Splits).Error,
		db.Order("id").Find(&l.Settlements).Error,
		db.Order("id").Find(&l.Allocations).Error,
	} {
		if q != nil {
			t.Fatal(q)
		}
	}
	return l
}

// A member who leaves takes their account with them, not the household's
// history: every share, settlement and allocation stays as it was.
func TestDeleteAccount_SharedHomeKeepsTheLedger(t *testing.T) {
	f := setupMoneyFixture(t)
	f.accountRoutes(t)
	db := f.db
	now := time.Now()

	// Bob's own category, filed under a shared expense, and one nobody uses.
	usedCat := models.Category{UserID: &f.bob.ID, Name: "Hobbies"}
	unusedCat := models.Category{UserID: &f.bob.ID, Name: "Unused"}
	mustCreate(t, db, &usedCat, &unusedCat)
	mustCreate(t, db, &models.Subcategory{CategoryID: unusedCat.ID, Name: "Nothing"})

	// A project shared with alice, and one only bob is in.
	shared := models.Project{UserID: f.bob.ID, PropertyID: &f.prop.ID, Name: "Kitchen", Budget: 500, StartDate: now, EndDate: now}
	private := models.Project{UserID: f.bob.ID, Name: "Bike", Budget: 100, StartDate: now, EndDate: now}
	mustCreate(t, db, &shared, &private)
	mustCreate(t, db,
		&models.ProjectMember{ProjectID: shared.ID, UserID: f.bob.ID, Role: "owner"},
		&models.ProjectMember{ProjectID: shared.ID, UserID: f.alice.ID, Role: "member"},
		&models.ProjectMember{ProjectID: private.ID, UserID: f.bob.ID, Role: "owner"},
	)

	// Settled history both ways: alice paid, bob paid (under his category and
	// the shared project).
	aliceExp := f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	rec := doJSON(t, f.router, http.MethodPost, "/expenses", f.bobTok, map[string]any{
		"amount": 60, "category_id": usedCat.ID, "property_id": f.prop.ID, "project_id": shared.ID,
		"date": "2026-05-02", "description": "Paint", "paid_by_member_id": f.mBob.ID,
		"is_split": true, "split_with_member_ids": []uint{f.mAlice.ID},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("bob's expense: status %d, body %s", rec.Code, rec.Body.String())
	}
	var bobExp models.Expense
	json.Unmarshal(rec.Body.Bytes(), &bobExp)
	// Bob owes 50, is owed 30: paying the net 20 clears both.
	f.settle(t, f.aliceTok, f.mBob.ID, f.mAlice.ID, 20)

	// What only bob could see, or only concerned him.
	personal := models.Expense{UserID: f.bob.ID, HouseholdID: f.prop.HouseholdID, PropertyID: &f.prop.ID, CategoryID: f.homeCatID, Amount: 9, Date: now, PaidByMemberID: f.mBob.ID}
	trashed := models.Expense{UserID: f.bob.ID, HouseholdID: f.prop.HouseholdID, CategoryID: unusedCat.ID, Amount: 3, Date: now, PaidByMemberID: f.mBob.ID}
	mustCreate(t, db, &personal, &trashed)
	db.Delete(&trashed)
	mustCreate(t, db,
		&models.ExpenseTemplate{UserID: f.bob.ID, Name: "Coffee", CategoryID: usedCat.ID, ProjectID: &private.ID},
		&models.ContractTemplate{UserID: f.bob.ID, Name: "Gas", Provider: "Eni", UtilityType: "gas"},
		&models.Notification{UserID: f.bob.ID, Type: "expense_shared", Title: "New expense"},
		&models.UserSettings{UserID: f.bob.ID},
		&models.UserSettings{UserID: f.alice.ID, DefaultSplitWithMemberIDs: "[" + itoa(f.mBob.ID) + "," + itoa(f.mCarol.ID) + "]"},
	)

	// A home bob left earlier: his removed member row is still there.
	oldHome := models.Property{Household: &models.Household{Name: "Old home"}, UserID: f.alice.ID, Name: "Old home", Type: "rented", StartDate: now, Residents: 1}
	mustCreate(t, db, &oldHome)
	oldMember := models.HouseholdMember{HouseholdID: oldHome.HouseholdID, UserID: &f.bob.ID, Name: "Bob", Role: "member",
		DeletedAt: gorm.DeletedAt{Time: now, Valid: true}}
	mustCreate(t, db, &oldMember, &models.HouseholdMember{HouseholdID: oldHome.HouseholdID, Name: "Alice", Role: "admin", IsVirtual: true})

	// A service bob pays, split with him by default.
	utility := models.Utility{UserID: f.alice.ID, PropertyID: f.prop.ID, Type: "electricity", Provider: "Enel", IsActive: true,
		PaidByMemberID: &f.mBob.ID, SplitOverride: "custom", SplitMemberIDs: "[" + itoa(f.mBob.ID) + "," + itoa(f.mCarol.ID) + "]"}
	mustCreate(t, db, &utility)

	before := readLedger(t, db)
	balanceBefore, _ := CalculateBalance(f.mAlice.ID, f.mBob.ID, f.prop.ID, db)

	if code := f.deleteOwnAccount(t, f.bobTok); code != http.StatusOK {
		t.Fatalf("delete account: status %d", code)
	}

	after := readLedger(t, db)
	if len(after.Splits) != len(before.Splits) || len(after.Settlements) != len(before.Settlements) || len(after.Allocations) != len(before.Allocations) {
		t.Fatalf("ledger changed: %d/%d/%d splits/settlements/allocations, was %d/%d/%d",
			len(after.Splits), len(after.Settlements), len(after.Allocations),
			len(before.Splits), len(before.Settlements), len(before.Allocations))
	}
	for i := range before.Splits {
		b, a := before.Splits[i], after.Splits[i]
		if a.ID != b.ID || a.MemberID != b.MemberID || a.Amount != b.Amount || a.SettledAmount != b.SettledAmount || a.IsSettled != b.IsSettled {
			t.Errorf("split %d changed: %+v, was %+v", b.ID, a, b)
		}
	}
	if balanceAfter, _ := CalculateBalance(f.mAlice.ID, f.mBob.ID, f.prop.ID, db); balanceAfter != balanceBefore {
		t.Errorf("alice's balance with bob moved from %v to %v", balanceBefore, balanceAfter)
	}

	var member models.HouseholdMember
	db.First(&member, f.mBob.ID)
	if member.UserID != nil || !member.IsVirtual || member.Name != "Bob" {
		t.Errorf("bob's member: user_id %v, virtual %v, name %q; want a virtual Bob", member.UserID, member.IsVirtual, member.Name)
	}
	var n int64
	db.Unscoped().Model(&models.HouseholdMember{}).Where("id = ?", oldMember.ID).Count(&n)
	if n != 0 {
		t.Error("bob's removed member row in the old home survived, though nothing refers to it")
	}

	for name, gone := range map[string]any{
		"user":             &models.User{ID: f.bob.ID},
		"personal expense": &models.Expense{ID: personal.ID},
		"trashed expense":  &models.Expense{ID: trashed.ID},
		"unused category":  &models.Category{ID: unusedCat.ID},
		"private project":  &models.Project{ID: private.ID},
	} {
		if err := db.Unscoped().Where(gone).First(gone).Error; err == nil {
			t.Errorf("%s survived the account", name)
		}
	}
	for _, table := range []string{"expense_templates", "contract_templates", "notifications", "user_settings", "property_join_requests"} {
		db.Table(table).Where("user_id = ?", f.bob.ID).Count(&n)
		if n != 0 {
			t.Errorf("%s: %d rows of bob's left", table, n)
		}
	}

	var cat models.Category
	if err := db.First(&cat, usedCat.ID).Error; err != nil || cat.UserID != nil {
		t.Errorf("category under a shared expense: err %v, owner %v; want kept without an owner", err, cat.UserID)
	}
	var exp models.Expense
	if err := db.First(&exp, bobExp.ID).Error; err != nil || exp.ProjectID == nil || *exp.ProjectID != shared.ID {
		t.Errorf("bob's shared expense: err %v, project %v; want kept in the shared project", err, exp.ProjectID)
	}
	if err := db.First(&models.Expense{}, aliceExp).Error; err != nil {
		t.Errorf("alice's expense: %v", err)
	}

	var project models.Project
	db.First(&project, shared.ID)
	var role string
	db.Model(&models.ProjectMember{}).Where("project_id = ? AND user_id = ?", shared.ID, f.alice.ID).Pluck("role", &role)
	if project.UserID != f.alice.ID || role != "owner" {
		t.Errorf("shared project: owner %d, alice's role %q; want alice as owner", project.UserID, role)
	}
	var indexedOwner uint
	db.Raw("SELECT user_id FROM search_index WHERE entity_type = 'project' AND entity_id = ?", shared.ID).Scan(&indexedOwner)
	if indexedOwner != f.alice.ID {
		t.Errorf("search index lists the shared project under user %d, want alice", indexedOwner)
	}
	db.Raw("SELECT count(*) FROM search_index WHERE (entity_type = 'project' AND entity_id = ?) OR (entity_type = 'expense' AND entity_id IN ?)",
		private.ID, []uint{personal.ID, trashed.ID}).Scan(&n)
	if n != 0 {
		t.Errorf("search index still lists %d deleted rows", n)
	}

	var settings models.UserSettings
	db.Where("user_id = ?", f.alice.ID).First(&settings)
	db.First(&utility, utility.ID)
	want := "[" + itoa(f.mCarol.ID) + "]"
	if settings.DefaultSplitWithMemberIDs != want || utility.SplitMemberIDs != want || utility.PaidByMemberID != nil {
		t.Errorf("split defaults still name bob: alice %q, service %q (payer %v); want %q and no payer",
			settings.DefaultSplitWithMemberIDs, utility.SplitMemberIDs, utility.PaidByMemberID, want)
	}

	var prop models.Property
	db.First(&prop, f.prop.ID)
	if prop.Residents != 1 {
		t.Errorf("residents = %d, want 1", prop.Residents)
	}
	assertNoForeignKeyViolations(t, db)
}

// While a share is still open, the member's account cannot go: the household
// would be left with a debt nobody can settle.
func TestDeleteAccount_OpenBalanceIsRefused(t *testing.T) {
	f := setupMoneyFixture(t)
	f.accountRoutes(t)
	f.createSplitExpense(t, 100, []uint{f.mBob.ID})

	rec := doJSON(t, f.router, http.MethodGet, "/settings/account/delete-check", f.bobTok, nil)
	var check DeleteAccountCheckResponse
	json.Unmarshal(rec.Body.Bytes(), &check)
	if check.CanDelete || len(check.OpenBalanceHouseholds) != 1 || check.OpenBalanceHouseholds[0] != f.prop.Name {
		t.Errorf("delete check: %+v; want blocked on %q", check, f.prop.Name)
	}

	rec = doJSON(t, f.router, http.MethodDelete, "/settings/account", f.bobTok, map[string]any{"password": acctPassword})
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "open_balances_remaining" {
		t.Fatalf("delete: status %d, body %s; want 409 open_balances_remaining", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, f.router, http.MethodDelete, "/admin/users/"+itoa(f.bob.ID), f.aliceTok, nil)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "open_balances_remaining" {
		t.Fatalf("admin delete: status %d, body %s; want 409 open_balances_remaining", rec.Code, rec.Body.String())
	}
	var n int64
	f.db.Model(&models.User{}).Where("id = ?", f.bob.ID).Count(&n)
	if n != 1 {
		t.Error("bob's account was deleted")
	}

	// Once it is settled, the account can go.
	f.settle(t, f.aliceTok, f.mBob.ID, f.mAlice.ID, 50)
	if code := f.deleteOwnAccount(t, f.bobTok); code != http.StatusOK {
		t.Errorf("after settling: status %d, want 200", code)
	}
}

// The home's owner leaves it to the admin they appointed, who can then manage
// it: the home is not orphaned.
func TestDeleteAccount_OwnerHandsTheHomeToTheSuccessor(t *testing.T) {
	f := setupMoneyFixture(t)
	f.accountRoutes(t)

	if code := f.deleteOwnAccount(t, f.aliceTok); code != http.StatusConflict {
		t.Fatalf("without a successor: status %d, want 409", code)
	}
	rec := doJSON(t, f.router, http.MethodPost, "/settings/account/promote-admin", f.aliceTok,
		map[string]any{"household_id": f.prop.HouseholdID, "member_id": f.mBob.ID})
	if rec.Code != http.StatusOK {
		t.Fatalf("promote: status %d, body %s", rec.Code, rec.Body.String())
	}
	if code := f.deleteOwnAccount(t, f.aliceTok); code != http.StatusOK {
		t.Fatalf("with a successor: status %d, want 200", code)
	}

	var prop models.Property
	if err := f.db.First(&prop, f.prop.ID).Error; err != nil {
		t.Fatalf("home: %v", err)
	}
	if prop.UserID != f.bob.ID {
		t.Errorf("home owner = %d, want bob (%d)", prop.UserID, f.bob.ID)
	}
	if !prop.IsCurrent {
		t.Error("the household lost its primary home: handing it over must keep is_current")
	}
	var n int64
	f.db.Unscoped().Model(&models.HouseholdMember{}).Where("id = ?", f.mAlice.ID).Count(&n)
	if n != 0 {
		t.Error("alice's member row survived, though nothing refers to it")
	}
	assertNoForeignKeyViolations(t, f.db)
}

// An admin deleting a user goes through the same rules, and only them.
func TestAdminDeleteUser_SameRulesAsOwnDeletion(t *testing.T) {
	f := setupMoneyFixture(t)
	f.accountRoutes(t)

	rec := doJSON(t, f.router, http.MethodDelete, "/admin/users/"+itoa(f.alice.ID), f.aliceTok, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("deleting oneself: status %d, want 400", rec.Code)
	}
	rec = doJSON(t, f.router, http.MethodDelete, "/admin/users/99999", f.aliceTok, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown user: status %d, want 404", rec.Code)
	}

	f.createSplitExpense(t, 100, []uint{f.mBob.ID})
	f.settle(t, f.aliceTok, f.mBob.ID, f.mAlice.ID, 50)
	rec = doJSON(t, f.router, http.MethodDelete, "/admin/users/"+itoa(f.bob.ID), f.aliceTok, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin delete: status %d, body %s", rec.Code, rec.Body.String())
	}
	var member models.HouseholdMember
	f.db.First(&member, f.mBob.ID)
	if member.UserID != nil || !member.IsVirtual {
		t.Errorf("bob's member: user_id %v, virtual %v; want virtual", member.UserID, member.IsVirtual)
	}
	assertNoForeignKeyViolations(t, f.db)
}

// A member whose only expenses were their own leaves no trace in the home:
// those expenses go, and nothing is left to keep a virtual member for.
func TestDeleteAccount_PersonalExpensesKeepNoMember(t *testing.T) {
	f := setupMoneyFixture(t)
	f.accountRoutes(t)
	mustCreate(t, f.db, &models.Expense{UserID: f.bob.ID, HouseholdID: f.prop.HouseholdID, PropertyID: &f.prop.ID, CategoryID: f.homeCatID,
		Amount: 9, Date: time.Now(), PaidByMemberID: f.mBob.ID})

	if code := f.deleteOwnAccount(t, f.bobTok); code != http.StatusOK {
		t.Fatalf("delete account: status %d", code)
	}
	var n int64
	f.db.Unscoped().Model(&models.HouseholdMember{}).Where("id = ?", f.mBob.ID).Count(&n)
	if n != 0 {
		t.Error("bob's member row survived as a virtual member, though only his deleted expenses referred to it")
	}
	assertNoForeignKeyViolations(t, f.db)
}

// A home the account deleted earlier (soft) goes with the account, tree and all.
func TestDeleteAccount_RemovedHomeGoesToo(t *testing.T) {
	f := setupMoneyFixture(t)
	f.accountRoutes(t)
	now := time.Now()
	old := models.Property{Household: &models.Household{Name: "Old flat"}, UserID: f.bob.ID, Name: "Old flat", Type: "rented", StartDate: now, Residents: 1}
	mustCreate(t, f.db, &old)
	// Removed with it: an account is in one household, the current one.
	member := models.HouseholdMember{HouseholdID: old.HouseholdID, UserID: &f.bob.ID, Name: "Bob", Role: "admin",
		DeletedAt: gorm.DeletedAt{Time: now, Valid: true}}
	mustCreate(t, f.db, &member)
	utility := models.Utility{UserID: f.bob.ID, PropertyID: old.ID, Type: "gas", Provider: "Gas Co", IsActive: true}
	mustCreate(t, f.db, &utility)
	mustCreate(t, f.db, &models.Expense{UserID: f.bob.ID, HouseholdID: old.HouseholdID, PropertyID: &old.ID, CategoryID: f.homeCatID,
		Amount: 20, Date: now, PaidByMemberID: member.ID, IsSplit: true})
	f.db.Delete(&old)

	if code := f.deleteOwnAccount(t, f.bobTok); code != http.StatusOK {
		t.Fatalf("delete account: status %d", code)
	}
	for name, q := range map[string]*gorm.DB{
		"home":    f.db.Unscoped().Model(&models.Property{}).Where("id = ?", old.ID),
		"service": f.db.Unscoped().Model(&models.Utility{}).Where("property_id = ?", old.ID),
		"expense": f.db.Unscoped().Model(&models.Expense{}).Where("property_id = ?", old.ID),
		"member":  f.db.Unscoped().Model(&models.HouseholdMember{}).Where("property_id = ?", old.ID),
	} {
		var n int64
		q.Count(&n)
		if n != 0 {
			t.Errorf("removed home: %d %s rows left", n, name)
		}
	}
	assertNoForeignKeyViolations(t, f.db)
}

// A step that fails rolls the whole deletion back: no half-left account.
func TestDeleteAccount_FailureChangesNothing(t *testing.T) {
	f := setupMoneyFixture(t)
	f.accountRoutes(t)
	personal := models.Expense{UserID: f.bob.ID, HouseholdID: f.prop.HouseholdID, PropertyID: &f.prop.ID, CategoryID: f.homeCatID,
		Amount: 9, Date: time.Now(), PaidByMemberID: f.mBob.ID}
	mustCreate(t, f.db, &personal)
	if err := f.db.Exec("CREATE TRIGGER keep_users BEFORE DELETE ON users BEGIN SELECT RAISE(ABORT, 'no'); END").Error; err != nil {
		t.Fatal(err)
	}

	for _, rec := range []*httptest.ResponseRecorder{
		doJSON(t, f.router, http.MethodDelete, "/settings/account", f.bobTok, map[string]any{"password": acctPassword}),
		doJSON(t, f.router, http.MethodDelete, "/admin/users/"+itoa(f.bob.ID), f.aliceTok, nil),
	} {
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status %d, body %s; want 500", rec.Code, rec.Body.String())
		}
	}
	var member models.HouseholdMember
	f.db.First(&member, f.mBob.ID)
	if member.UserID == nil || *member.UserID != f.bob.ID {
		t.Error("bob's member row was detached by a deletion that failed")
	}
	if err := f.db.First(&models.Expense{}, personal.ID).Error; err != nil {
		t.Errorf("bob's expense was deleted by a deletion that failed: %v", err)
	}
}

// The avatar file goes with the account.
func TestDeleteAccount_RemovesTheAvatar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DB_PATH", filepath.Join(dir, "homelog.db"))
	f := setupMoneyFixture(t)
	f.accountRoutes(t)
	avatar := filepath.Join(dir, "avatars", "bob.png")
	if err := os.MkdirAll(filepath.Dir(avatar), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(avatar, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.db.Model(f.bob).Update("avatar_path", "avatars/bob.png")

	if code := f.deleteOwnAccount(t, f.bobTok); code != http.StatusOK {
		t.Fatalf("delete account: status %d", code)
	}
	if _, err := os.Stat(avatar); !os.IsNotExist(err) {
		t.Errorf("avatar still on disk: %v", err)
	}
}
