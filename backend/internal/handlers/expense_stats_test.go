package handlers

// Dates are always passed explicitly (from/to) so buckets and windows are
// deterministic; the "period" shortcuts depend on today and are checked loosely.

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

type statsResponse struct {
	Trend         []TrendPoint    `json:"trend"`
	Granularity   string          `json:"granularity"`
	ByCategory    []CategoryStats `json:"by_category"`
	IsSubcategory bool            `json:"is_subcategory"`
	TotalMonth    float64         `json:"total_month"`
	TotalYear     float64         `json:"total_year"`
	TotalPeriod   float64         `json:"total_period"`
	AverageMonth  float64         `json:"average_month"`
	Count         int             `json:"count"`
	Scope         string          `json:"scope"`
	Period        struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"period"`
	Previous struct {
		Total float64 `json:"total"`
		Start string  `json:"start"`
		End   string  `json:"end"`
	} `json:"previous"`
}

func (f *moneyFixture) statsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.GET("/expenses/stats", NewExpenseHandler(f.db).GetStats)
	return r
}

func (f *moneyFixture) getStats(t *testing.T, token, query string) statsResponse {
	t.Helper()
	rec := doGET(t, f.statsRouter(), "/expenses/stats?"+query, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET stats?%s: status %d, body %s", query, rec.Code, rec.Body.String())
	}
	var out statsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	return out
}

// addExpense inserts an expense straight into the DB (bypassing the handler,
// which would also build splits we do not need here).
func (f *moneyFixture) addExpense(t *testing.T, amount float64, date time.Time, categoryID uint, subcategoryID *uint) models.Expense {
	t.Helper()
	return f.addExpenseIn(t, f.prop.ID, amount, date, categoryID, subcategoryID)
}

func (f *moneyFixture) addExpenseIn(t *testing.T, propertyID uint, amount float64, date time.Time, categoryID uint, subcategoryID *uint) models.Expense {
	t.Helper()
	e := models.Expense{
		UserID: f.alice.ID, PropertyID: &propertyID, CategoryID: categoryID, SubcategoryID: subcategoryID,
		Amount: amount, Date: date, Description: "stat", PaidByMemberID: f.mAlice.ID,
	}
	mustCreate(t, f.db, &e)
	return e
}

func (f *moneyFixture) newCategory(t *testing.T, slug, name string) models.Category {
	t.Helper()
	c := models.Category{Slug: slug, Name: name}
	mustCreate(t, f.db, &c)
	return c
}

func utc(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 12, 0, 0, 0, time.UTC) }

func TestStats_MonthlyTrendTotalsAndCategories(t *testing.T) {
	f := setupMoneyFixture(t)
	food := f.newCategory(t, "food", "Cibo")
	f.addExpense(t, 100, utc(2026, 1, 10), f.casaCatID, nil)
	f.addExpense(t, 50, utc(2026, 1, 20), food.ID, nil)
	f.addExpense(t, 150, utc(2026, 2, 5), f.casaCatID, nil)
	f.addExpense(t, 999, utc(2026, 5, 1), f.casaCatID, nil) // outside the window

	got := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-03-31")

	if got.Granularity != "month" {
		t.Fatalf("granularity = %q, want month", got.Granularity)
	}
	wantTrend := []TrendPoint{{Date: "2026-01-01", Amount: 150, Count: 2}, {Date: "2026-02-01", Amount: 150, Count: 1}}
	if len(got.Trend) != len(wantTrend) {
		t.Fatalf("trend = %+v, want %+v", got.Trend, wantTrend)
	}
	for i, w := range wantTrend {
		if got.Trend[i].Date != w.Date || !approx(got.Trend[i].Amount, w.Amount) || got.Trend[i].Count != w.Count {
			t.Errorf("trend[%d] = %+v, want %+v", i, got.Trend[i], w)
		}
	}
	if !approx(got.TotalPeriod, 300) || got.Count != 3 {
		t.Errorf("total = %.2f over %d expenses, want 300 over 3", got.TotalPeriod, got.Count)
	}
	if !approx(got.AverageMonth, 150) {
		t.Errorf("average per bucket = %.2f, want 150 (300 over 2 months with data)", got.AverageMonth)
	}
	if got.Scope != "household" || got.IsSubcategory {
		t.Errorf("scope=%q is_subcategory=%v", got.Scope, got.IsSubcategory)
	}
	if got.Period.Start != "2026-01-01" || got.Period.End != "2026-03-31" {
		t.Errorf("period = %+v", got.Period)
	}

	if len(got.ByCategory) != 2 {
		t.Fatalf("by_category = %+v, want 2 rows", got.ByCategory)
	}
	top, second := got.ByCategory[0], got.ByCategory[1]
	if top.CategoryID != f.casaCatID || !approx(top.Amount, 250) || top.Count != 2 || !approx(top.Percentage, 250.0/300*100) {
		t.Errorf("top category = %+v, want Casa 250 (83.3%%)", top)
	}
	if top.CategorySlug != "home" && top.CategorySlug == "" {
		t.Errorf("built-in category must carry its slug, got %q", top.CategorySlug)
	}
	if second.CategoryID != food.ID || second.CategorySlug != "food" || !approx(second.Amount, 50) {
		t.Errorf("second category = %+v, want food 50", second)
	}
	if !approx(top.Percentage+second.Percentage, 100) {
		t.Errorf("percentages sum to %.2f, want 100", top.Percentage+second.Percentage)
	}
}

func TestStats_PreviousWindowHasSameLength(t *testing.T) {
	f := setupMoneyFixture(t)
	f.addExpense(t, 100, utc(2026, 2, 10), f.casaCatID, nil)
	f.addExpense(t, 40, utc(2025, 11, 15), f.casaCatID, nil) // inside the previous window
	f.addExpense(t, 500, utc(2025, 6, 1), f.casaCatID, nil)  // too old

	got := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-03-31")

	if got.Previous.End != "2025-12-31" || got.Previous.Start != "2025-10-03" {
		t.Errorf("previous window = %s → %s, want 2025-10-03 → 2025-12-31 (same length, ending the day before)", got.Previous.Start, got.Previous.End)
	}
	if !approx(got.Previous.Total, 40) {
		t.Errorf("previous total = %.2f, want 40", got.Previous.Total)
	}
}

func TestStats_DailyAndQuarterlyGranularity(t *testing.T) {
	f := setupMoneyFixture(t)
	f.addExpense(t, 10, utc(2026, 1, 3), f.casaCatID, nil)
	f.addExpense(t, 5, utc(2026, 1, 3), f.casaCatID, nil)
	f.addExpense(t, 7, utc(2026, 1, 9), f.casaCatID, nil)
	f.addExpense(t, 20, utc(2025, 2, 1), f.casaCatID, nil)
	f.addExpense(t, 30, utc(2026, 5, 1), f.casaCatID, nil)

	day := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-01-15")
	if day.Granularity != "day" || len(day.Trend) != 2 {
		t.Fatalf("daily: granularity=%q trend=%+v", day.Granularity, day.Trend)
	}
	if day.Trend[0].Date != "2026-01-03" || !approx(day.Trend[0].Amount, 15) || day.Trend[0].Count != 2 {
		t.Errorf("first day = %+v, want 2026-01-03 / 15 / 2", day.Trend[0])
	}
	if day.Trend[1].Date != "2026-01-09" {
		t.Errorf("second day = %+v", day.Trend[1])
	}

	q := f.getStats(t, f.aliceTok, "from=2025-01-01&to=2026-06-30")
	if q.Granularity != "quarter" {
		t.Fatalf("granularity = %q, want quarter for a range over a year", q.Granularity)
	}
	wantDates := []string{"2025-01-01", "2026-01-01", "2026-04-01"} // Q1 2025, Q1 2026, Q2 2026
	if len(q.Trend) != len(wantDates) {
		t.Fatalf("quarters = %+v, want %v", q.Trend, wantDates)
	}
	for i, d := range wantDates {
		if q.Trend[i].Date != d {
			t.Errorf("quarter[%d] starts %s, want %s", i, q.Trend[i].Date, d)
		}
	}
}

func TestStats_CategoryFilterBreaksDownBySubcategory(t *testing.T) {
	f := setupMoneyFixture(t)
	var utilities models.Subcategory
	if err := f.db.Where("category_id = ?", f.casaCatID).First(&utilities).Error; err != nil {
		t.Fatalf("load subcategory: %v", err)
	}
	other := f.newCategory(t, "food", "Cibo")
	f.addExpense(t, 80, utc(2026, 1, 10), f.casaCatID, &utilities.ID)
	f.addExpense(t, 20, utc(2026, 1, 11), f.casaCatID, nil)
	f.addExpense(t, 500, utc(2026, 1, 12), other.ID, nil)

	got := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-01-31&category_id="+itoa(f.casaCatID))

	if !got.IsSubcategory {
		t.Error("is_subcategory must be true when a category is selected")
	}
	if !approx(got.TotalPeriod, 100) || got.Count != 2 {
		t.Errorf("total = %.2f over %d, want 100 over 2 (other categories excluded)", got.TotalPeriod, got.Count)
	}
	if len(got.ByCategory) != 2 {
		t.Fatalf("breakdown = %+v, want 2 rows", got.ByCategory)
	}
	if got.ByCategory[0].CategoryID != utilities.ID || got.ByCategory[0].CategorySlug != utilities.Slug || !approx(got.ByCategory[0].Percentage, 80) {
		t.Errorf("first row = %+v, want the subcategory at 80%%", got.ByCategory[0])
	}
	none := got.ByCategory[1]
	if none.CategoryID != 0 || none.CategoryName != "" || none.CategorySlug != "" {
		t.Errorf("no-subcategory bucket = %+v, want id 0 and empty labels (the client names it)", none)
	}
}

func TestStats_PropertyFilterAndHouseholdIsolation(t *testing.T) {
	f := setupMoneyFixture(t)
	f.addExpense(t, 100, utc(2026, 1, 10), f.casaCatID, nil)

	// A second property of Alice's, and a stranger's household.
	second := models.Property{UserID: f.alice.ID, Name: "Garage", Type: "owned", StartDate: time.Now()}
	mustCreate(t, f.db, &second)
	mustCreate(t, f.db, &models.HouseholdMember{PropertyID: second.ID, UserID: &f.alice.ID, Name: "Alice", Role: "admin"})
	f.addExpenseIn(t, second.ID, 30, utc(2026, 1, 11), f.casaCatID, nil)

	eve := &models.User{Email: "eve@example.com", PasswordHash: "x", Name: "Eve", Role: "user", IsActive: true}
	mustCreate(t, f.db, eve)
	evesHome := models.Property{UserID: eve.ID, Name: "Eve", Type: "owned", StartDate: time.Now()}
	mustCreate(t, f.db, &evesHome)
	mustCreate(t, f.db, &models.HouseholdMember{PropertyID: evesHome.ID, UserID: &eve.ID, Name: "Eve", Role: "admin"})
	f.addExpenseIn(t, evesHome.ID, 7777, utc(2026, 1, 12), f.casaCatID, nil)

	all := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-01-31")
	if !approx(all.TotalPeriod, 130) {
		t.Errorf("all of Alice's properties = %.2f, want 130 and never Eve's 7777", all.TotalPeriod)
	}

	only := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-01-31&property_id="+itoa(second.ID))
	if !approx(only.TotalPeriod, 30) || only.Count != 1 {
		t.Errorf("property filter = %.2f over %d, want 30 over 1", only.TotalPeriod, only.Count)
	}

	// Naming a property she is not a member of must not open it up.
	probe := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-01-31&property_id="+itoa(evesHome.ID))
	if probe.TotalPeriod != 0 || probe.Count != 0 {
		t.Errorf("another household's property leaked: %.2f over %d", probe.TotalPeriod, probe.Count)
	}

	// Eve sees her own household only.
	mine := f.getStats(t, testutil.SignToken(t, eve), "from=2026-01-01&to=2026-01-31")
	if !approx(mine.TotalPeriod, 7777) {
		t.Errorf("Eve's total = %.2f, want 7777", mine.TotalPeriod)
	}
}

func TestStats_EmptyHouseholdHasNoNaN(t *testing.T) {
	f := setupMoneyFixture(t)

	got := f.getStats(t, f.aliceTok, "from=2026-01-01&to=2026-03-31")

	if got.TotalPeriod != 0 || got.Count != 0 || got.AverageMonth != 0 || got.Previous.Total != 0 {
		t.Errorf("empty stats = %+v, want all zeros", got)
	}
	if got.Trend == nil || got.ByCategory == nil {
		t.Error("empty series must serialise as [] and not null")
	}
}

func TestStats_AllStartsAtFirstExpense(t *testing.T) {
	f := setupMoneyFixture(t)
	f.addExpense(t, 10, utc(2024, 3, 15), f.casaCatID, nil)
	f.addExpense(t, 20, utc(2025, 7, 1), f.casaCatID, nil)

	got := f.getStats(t, f.aliceTok, "all=true")

	if got.Period.Start != "2024-03-15" {
		t.Errorf("period start = %s, want the first expense's date 2024-03-15", got.Period.Start)
	}
	if !approx(got.TotalPeriod, 30) {
		t.Errorf("total = %.2f, want 30", got.TotalPeriod)
	}
}

func TestStats_YearDefaultAndShortcuts(t *testing.T) {
	f := setupMoneyFixture(t)
	f.addExpense(t, 100, utc(2023, 6, 1), f.casaCatID, nil)

	byYear := f.getStats(t, f.aliceTok, "year=2023")
	if byYear.Period.Start != "2023-01-01" || byYear.Period.End != "2023-12-31" || !approx(byYear.TotalPeriod, 100) {
		t.Errorf("year=2023 → %+v total %.2f", byYear.Period, byYear.TotalPeriod)
	}
	if bad := f.getStats(t, f.aliceTok, "year=notayear"); bad.Period.Start == "" {
		t.Error("an unparseable year must fall back to the current one, not break the response")
	}

	// "1m" spans at most 31 days → daily buckets; "3m" is wider → monthly.
	if got := f.getStats(t, f.aliceTok, "period=1m"); got.Granularity != "day" {
		t.Errorf("period=1m granularity = %q, want day", got.Granularity)
	}
	if got := f.getStats(t, f.aliceTok, "period=3m"); got.Granularity != "month" {
		t.Errorf("period=3m granularity = %q, want month", got.Granularity)
	}
	if got := f.getStats(t, f.aliceTok, "period=6m"); got.Granularity != "month" {
		t.Errorf("period=6m granularity = %q, want month", got.Granularity)
	}
}

func TestStats_CurrentMonthAndYearTotalsIgnoreTheWindow(t *testing.T) {
	f := setupMoneyFixture(t)
	now := time.Now().UTC()
	f.addExpense(t, 25, time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, time.UTC), f.casaCatID, nil)

	// The selected window is years ago, yet the dashboard's "this month" and
	// "this year" tiles still count today's spending.
	got := f.getStats(t, f.aliceTok, "from=2020-01-01&to=2020-01-31")

	if !approx(got.TotalMonth, 25) || !approx(got.TotalYear, 25) {
		t.Errorf("month/year totals = %.2f/%.2f, want 25/25", got.TotalMonth, got.TotalYear)
	}
	if got.TotalPeriod != 0 {
		t.Errorf("period total = %.2f, want 0", got.TotalPeriod)
	}
}

func TestStats_RequiresAuth(t *testing.T) {
	f := setupMoneyFixture(t)
	if rec := doGET(t, f.statsRouter(), "/expenses/stats", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", rec.Code)
	}
}
