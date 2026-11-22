package main

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/handlers"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// twoHouseholdsFixture is what the upgrade has made, before merging, for a
// couple who had two homes: two households with the same two accounts and a
// virtual member each, their own expenses, settlements and balances.
type twoHouseholdsFixture struct {
	db                     *gorm.DB
	alice, bob             *models.User
	flat, lake             models.Property
	aliceFlat, bobFlat     models.HouseholdMember
	carolFlat              models.HouseholdMember
	aliceLake, bobLake     models.HouseholdMember
	carolLake, danLake     models.HouseholdMember
	lakeUtility            models.Utility
	balanceBefore          float64
	balanceCarolBobBefore  float64
	lakeJoinFromNewcomerID uint
}

func setupTwoHouseholds(t *testing.T) *twoHouseholdsFixture {
	t.Helper()
	db := testutil.NewDB(t)
	// Before merging, the schema allows an account in two households.
	if err := db.Exec(`DROP INDEX idx_household_members_account`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	f := &twoHouseholdsFixture{db: db}

	f.alice = &models.User{Email: "alice@example.com", PasswordHash: "x", Name: "Alice", IsActive: true}
	f.bob = &models.User{Email: "bob@example.com", PasswordHash: "x", Name: "Bob", IsActive: true}
	newcomer := &models.User{Email: "eve@example.com", PasswordHash: "x", Name: "Eve", IsActive: true}
	mustCreate(t, db, f.alice, f.bob, newcomer)

	f.flat = models.Property{Household: &models.Household{Name: "Flat"}, UserID: f.alice.ID, Name: "Flat", Type: "rented", StartDate: now, IsCurrent: true, Residents: 2}
	f.lake = models.Property{Household: &models.Household{Name: "Lake house"}, UserID: f.alice.ID, Name: "Lake house", Type: "owned", StartDate: now, IsCurrent: true, Residents: 2}
	mustCreate(t, db, &f.flat, &f.lake)
	mustCreate(t, db,
		&models.HouseholdSettings{HouseholdID: f.flat.HouseholdID, SplitMode: true, DefaultSplitType: "equal"},
		&models.HouseholdSettings{HouseholdID: f.lake.HouseholdID, SplitMode: true, DefaultSplitType: "equal"},
	)

	f.aliceFlat = models.HouseholdMember{HouseholdID: f.flat.HouseholdID, UserID: &f.alice.ID, Name: "Alice", Role: "admin"}
	f.bobFlat = models.HouseholdMember{HouseholdID: f.flat.HouseholdID, UserID: &f.bob.ID, Name: "Bob", Role: "member"}
	f.carolFlat = models.HouseholdMember{HouseholdID: f.flat.HouseholdID, Name: "Carol", IsVirtual: true}
	// Bob is admin of the lake house only: he stays admin after the merge.
	f.aliceLake = models.HouseholdMember{HouseholdID: f.lake.HouseholdID, UserID: &f.alice.ID, Name: "Alice", Role: "admin"}
	f.bobLake = models.HouseholdMember{HouseholdID: f.lake.HouseholdID, UserID: &f.bob.ID, Name: "Bob", Role: "admin"}
	f.carolLake = models.HouseholdMember{HouseholdID: f.lake.HouseholdID, Name: " carol ", IsVirtual: true}
	f.danLake = models.HouseholdMember{HouseholdID: f.lake.HouseholdID, Name: "Dan", IsVirtual: true}
	mustCreate(t, db, &f.aliceFlat, &f.bobFlat, &f.carolFlat, &f.aliceLake, &f.bobLake, &f.carolLake, &f.danLake)

	cat := models.Category{Name: "Food", Slug: "food_dining"}
	mustCreate(t, db, &cat)
	split := func(household uint, payer models.HouseholdMember, amount float64, debtors ...models.HouseholdMember) {
		t.Helper()
		e := models.Expense{UserID: f.alice.ID, HouseholdID: household, CategoryID: cat.ID, Amount: amount, Date: now, Description: "x", PaidByMemberID: payer.ID, IsSplit: true}
		mustCreate(t, db, &e)
		share := amount / float64(len(debtors)+1)
		mustCreate(t, db, &models.ExpenseSplit{ExpenseID: e.ID, MemberID: payer.ID, Amount: share, IsSettled: true, SettledAt: &now})
		for _, d := range debtors {
			mustCreate(t, db, &models.ExpenseSplit{ExpenseID: e.ID, MemberID: d.ID, Amount: share})
		}
	}
	split(f.flat.HouseholdID, f.aliceFlat, 100, f.bobFlat)             // Bob owes Alice 50
	split(f.lake.HouseholdID, f.bobLake, 60, f.aliceLake)              // Alice owes Bob 30
	split(f.lake.HouseholdID, f.aliceLake, 90, f.bobLake, f.carolLake) // Bob, Carol owe Alice 30 each
	split(f.lake.HouseholdID, f.danLake, 20, f.bobLake)                // Bob owes Dan 10
	mustCreate(t, db, &models.Settlement{HouseholdID: f.lake.HouseholdID, FromMemberID: f.carolLake.ID, ToMemberID: f.aliceLake.ID, Amount: 1, Date: now})

	f.lakeUtility = models.Utility{UserID: f.alice.ID, PropertyID: f.lake.ID, Type: "gas", Provider: "Acme", IsActive: true,
		PaidByMemberID: &f.bobLake.ID, SplitOverride: "custom", SplitMemberIDs: "[" + itoa(f.aliceLake.ID) + "," + itoa(f.danLake.ID) + "]"}
	mustCreate(t, db, &f.lakeUtility)
	settings := models.NewUserSettings(f.alice.ID, "en")
	settings.DefaultSplitWithMemberIDs = "[" + itoa(f.bobFlat.ID) + "," + itoa(f.bobLake.ID) + "]"
	mustCreate(t, db, &settings)

	// Bob is already a member of the flat: his request to the lake house goes.
	stale := models.HouseholdJoinRequest{UserID: f.bob.ID, HouseholdID: f.lake.HouseholdID, Status: "rejected"}
	fresh := models.HouseholdJoinRequest{UserID: newcomer.ID, HouseholdID: f.lake.HouseholdID, Status: "pending"}
	mustCreate(t, db, &stale, &fresh)
	f.lakeJoinFromNewcomerID = fresh.ID

	f.balanceBefore = f.balance(t, f.aliceFlat.ID, f.bobFlat.ID, f.flat.HouseholdID) +
		f.balance(t, f.aliceLake.ID, f.bobLake.ID, f.lake.HouseholdID)
	f.balanceCarolBobBefore = f.balance(t, f.aliceLake.ID, f.carolLake.ID, f.lake.HouseholdID)
	return f
}

func (f *twoHouseholdsFixture) balance(t *testing.T, current, other, household uint) float64 {
	t.Helper()
	b, err := handlers.CalculateBalance(current, other, household, f.db)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func itoa(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}

func TestMergeHouseholds_SamePeopleBecomeOneHousehold(t *testing.T) {
	f := setupTwoHouseholds(t)
	db := f.db

	plans, err := planHouseholdMerges(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Target.ID != f.flat.HouseholdID || plans[0].Source.ID != f.lake.HouseholdID || plans[0].Skip != "" {
		t.Fatalf("plans = %+v, want the lake house into the flat", plans)
	}
	if err := mergeHouseholds(db); err != nil {
		t.Fatal(err)
	}

	var households int64
	db.Model(&models.Household{}).Count(&households)
	if households != 1 {
		t.Errorf("%d households after the merge, want 1", households)
	}
	var lake models.Property
	db.First(&lake, f.lake.ID)
	var flat models.Property
	db.First(&flat, f.flat.ID)
	if lake.HouseholdID != f.flat.HouseholdID || lake.IsCurrent || !flat.IsCurrent {
		t.Errorf("lake house: household %d primary %v, flat primary %v; want it in the flat's household, the flat still primary", lake.HouseholdID, lake.IsCurrent, flat.IsCurrent)
	}

	// Alice and Bob: one balance, the sum of the two.
	if got := f.balance(t, f.aliceFlat.ID, f.bobFlat.ID, f.flat.HouseholdID); math.Abs(got-f.balanceBefore) > 0.001 {
		t.Errorf("Alice-Bob balance = %.2f, want %.2f (the sum of both households)", got, f.balanceBefore)
	}
	// Carol of the lake house is Carol of the flat (names match regardless of
	// case and spaces), and her settlement came with her.
	if got := f.balance(t, f.aliceFlat.ID, f.carolFlat.ID, f.flat.HouseholdID); math.Abs(got-f.balanceCarolBobBefore) > 0.001 {
		t.Errorf("Alice-Carol balance = %.2f, want %.2f", got, f.balanceCarolBobBefore)
	}
	var members []models.HouseholdMember
	db.Unscoped().Where("household_id = ?", f.flat.HouseholdID).Order("id").Find(&members)
	var names []string
	for _, m := range members {
		names = append(names, m.Name)
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "Alice,Bob,Carol,Dan" {
		t.Errorf("members = %v, want Alice, Bob, Carol and Dan once each", names)
	}
	var bob models.HouseholdMember
	db.First(&bob, f.bobFlat.ID)
	if bob.Role != "admin" {
		t.Errorf("Bob's role = %q, want admin: he administered the lake house", bob.Role)
	}

	var utility models.Utility
	db.First(&utility, f.lakeUtility.ID)
	if utility.PaidByMemberID == nil || *utility.PaidByMemberID != f.bobFlat.ID ||
		utility.SplitMemberIDs != "["+itoa(f.aliceFlat.ID)+","+itoa(f.danLake.ID)+"]" {
		t.Errorf("utility payer %v split %s; want Bob of the flat and [Alice of the flat, Dan]", utility.PaidByMemberID, utility.SplitMemberIDs)
	}
	var settings models.UserSettings
	db.Where("user_id = ?", f.alice.ID).First(&settings)
	if settings.DefaultSplitWithMemberIDs != "["+itoa(f.bobFlat.ID)+"]" {
		t.Errorf("default split = %s, want Bob once", settings.DefaultSplitWithMemberIDs)
	}

	var requests []models.HouseholdJoinRequest
	db.Unscoped().Find(&requests)
	if len(requests) != 1 || requests[0].ID != f.lakeJoinFromNewcomerID || requests[0].HouseholdID != f.flat.HouseholdID {
		t.Errorf("join requests = %+v, want only the newcomer's, now for the flat", requests)
	}

	var broken []map[string]any
	db.Raw(`PRAGMA foreign_key_check`).Scan(&broken)
	if len(broken) != 0 {
		t.Errorf("foreign keys broken by the merge: %v", broken)
	}

	// Done: nothing left to merge, and running it again changes nothing.
	if again, _ := planHouseholdMerges(db); len(again) != 0 {
		t.Errorf("still planning %+v after the merge", again)
	}
}

// Two virtual members of the target with the same name: the merge cannot know
// which one is meant, so it stops and says why.
func TestMergeHouseholds_AmbiguousVirtualMemberStops(t *testing.T) {
	f := setupTwoHouseholds(t)
	mustCreate(t, f.db, &models.HouseholdMember{HouseholdID: f.flat.HouseholdID, Name: "Carol", IsVirtual: true})

	err := mergeHouseholds(f.db)
	if err == nil || !strings.Contains(err.Error(), "Rename the member") {
		t.Fatalf("merge: %v, want it to stop on the ambiguous member", err)
	}
	var households int64
	f.db.Model(&models.Household{}).Count(&households)
	if households != 2 {
		t.Errorf("%d households, want both kept", households)
	}
}

// One person sharing two homes with different people can't be in both
// households, and which one they keep is not the upgrade's to guess.
func TestMergeHouseholds_AccountWithDifferentPeopleStops(t *testing.T) {
	f := setupTwoHouseholds(t)
	f.db.Model(&models.HouseholdMember{}).Where("id = ?", f.bobLake.ID).
		Updates(map[string]any{"user_id": nil, "is_virtual": true})

	err := mergeHouseholds(f.db)
	if err == nil || !strings.Contains(err.Error(), `alice@example.com is in "Flat", "Lake house"`) {
		t.Fatalf("merge: %v, want it to name Alice and both homes", err)
	}
	if strings.Contains(err.Error(), "bob@example.com") {
		t.Errorf("merge: %v; Bob is in one household only", err)
	}
}

// Every foreign key into households and household_members must be handled by
// mergeHousehold: a new one would otherwise be left pointing at a deleted row.
func TestMergeHouseholds_CoversSchema(t *testing.T) {
	db := testutil.NewDB(t)
	handled := map[string]bool{
		"properties.household_id":              true,
		"household_members.household_id":       true,
		"expenses.household_id":                true,
		"settlements.household_id":             true,
		"household_settings.household_id":      true,
		"household_join_requests.household_id": true,
		"expenses.paid_by_member_id":           true,
		"expense_splits.member_id":             true,
		"settlements.from_member_id":           true,
		"settlements.to_member_id":             true,
		"utilities.paid_by_member_id":          true,
	}
	var edges []string
	if err := db.Raw(`SELECT m.name || '.' || f."from" FROM sqlite_master m, pragma_foreign_key_list(m.name) f
		WHERE m.type = 'table' AND f."table" IN ('households', 'household_members')`).Scan(&edges).Error; err != nil {
		t.Fatal(err)
	}
	if len(edges) == 0 {
		t.Fatal("no foreign key into households found: the scan is broken")
	}
	seen := map[string]bool{}
	for _, e := range edges {
		seen[e] = true
		if !handled[e] {
			t.Errorf("mergeHousehold does not handle %s", e)
		}
	}
	for e := range handled {
		if !seen[e] {
			t.Errorf("handled edge %s no longer exists in the schema: drop it from the list", e)
		}
	}
}
