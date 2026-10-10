package handlers

// calculateConsumptionAnalysis is pure: bills in, periods + summary out. These
// tests pin the algorithm documented on the function — sort by period_end, the
// first bill is an anchor with no row, and a bill without an associated
// self-reading falls back to the provider reading so its contribution is
// neutral.

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

func fp(v float64) *float64 { return &v }

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// gasBill builds a single-register bill whose period ends on the given day.
func gasBill(id uint, end time.Time, provider float64, user *float64) models.Bill {
	b := models.Bill{
		ID:              id,
		PeriodStart:     end.AddDate(0, -1, 0),
		PeriodEnd:       end,
		ProviderReading: fp(provider),
	}
	if user != nil {
		b.UserReading = &models.MeterReading{Value: user}
	}
	return b
}

func analyse(utilityType string, bills []models.Bill, threshold float64) ([]ConsumptionPeriod, *ConsumptionSummary) {
	return (&UtilityHandler{}).calculateConsumptionAnalysis(utilityType, bills, threshold)
}

func TestConsumptionAnalysis_NeedsTwoBills(t *testing.T) {
	for _, bills := range [][]models.Bill{nil, {}, {gasBill(1, day(2026, 1, 31), 100, nil)}} {
		periods, summary := analyse("gas", bills, 2)
		if periods != nil || summary != nil {
			t.Errorf("%d bill(s): got periods=%v summary=%v, want nil/nil", len(bills), periods, summary)
		}
	}
}

func TestConsumptionAnalysis_SortsByPeriodEndAndSkipsAnchor(t *testing.T) {
	bills := []models.Bill{
		gasBill(3, day(2026, 3, 31), 130, nil),
		gasBill(1, day(2026, 1, 31), 100, nil),
		gasBill(2, day(2026, 2, 28), 112, nil),
	}

	periods, summary := analyse("gas", bills, 2)

	if len(periods) != 2 {
		t.Fatalf("got %d periods, want 2 (the first bill is an anchor with no row)", len(periods))
	}
	if *periods[0].BillID != 2 || *periods[1].BillID != 3 {
		t.Errorf("periods out of order: bill ids %d, %d; want 2, 3", *periods[0].BillID, *periods[1].BillID)
	}
	if !approx(*periods[0].ProviderConsumption, 12) || !approx(*periods[1].ProviderConsumption, 18) {
		t.Errorf("billed consumption = %.1f, %.1f; want 12, 18", *periods[0].ProviderConsumption, *periods[1].ProviderConsumption)
	}
	if !approx(summary.TotalProvider, 30) {
		t.Errorf("TotalProvider = %.1f, want 30 (column sum)", summary.TotalProvider)
	}
	if !summary.FirstPeriod.Equal(bills[1].PeriodStart) || !summary.LastPeriod.Equal(day(2026, 3, 31)) {
		t.Errorf("covered range = %v → %v, want the anchor's start → the last period end", summary.FirstPeriod, summary.LastPeriod)
	}
	if bills[0].ID != 3 {
		t.Error("the caller's slice must not be reordered")
	}
}

func TestConsumptionAnalysis_NoUserReadingIsNeutral(t *testing.T) {
	bills := []models.Bill{
		gasBill(1, day(2026, 1, 31), 100, nil),
		gasBill(2, day(2026, 2, 28), 112, nil),
	}

	periods, summary := analyse("gas", bills, 2)

	if !approx(*periods[0].UserConsumption, 12) {
		t.Errorf("effective consumption = %.1f, want it to fall back to the billed 12", *periods[0].UserConsumption)
	}
	if !approx(*periods[0].Difference, 0) || !approx(summary.CumulativeDifference, 0) {
		t.Errorf("difference = %.1f / cumulative %.1f, want 0", *periods[0].Difference, summary.CumulativeDifference)
	}
	if summary.HasCumulativeAlert || summary.CumulativeCredit {
		t.Errorf("a neutral history must not raise anything: %+v", summary)
	}
}

func TestConsumptionAnalysis_UserReadingsDriveDifference(t *testing.T) {
	// Provider billed 12 then 20; the household's own readings say 10 then 15.
	bills := []models.Bill{
		gasBill(1, day(2026, 1, 31), 100, fp(100)),
		gasBill(2, day(2026, 2, 28), 112, fp(110)),
		gasBill(3, day(2026, 3, 31), 132, fp(125)),
	}

	periods, summary := analyse("gas", bills, 100)

	wantUser := []float64{10, 15}
	wantDiff := []float64{2, 5}
	for i, p := range periods {
		if !approx(*p.UserConsumption, wantUser[i]) {
			t.Errorf("period %d user consumption = %.1f, want %.1f", i, *p.UserConsumption, wantUser[i])
		}
		if !approx(*p.Difference, wantDiff[i]) {
			t.Errorf("period %d difference = %.1f, want %.1f", i, *p.Difference, wantDiff[i])
		}
	}
	if !approx(summary.TotalUser, 25) || !approx(summary.TotalProvider, 32) || !approx(summary.CumulativeDifference, 7) {
		t.Errorf("totals user=%.1f provider=%.1f diff=%.1f; want 25, 32, 7", summary.TotalUser, summary.TotalProvider, summary.CumulativeDifference)
	}
}

func TestConsumptionAnalysis_MissingProviderReadingLeavesGap(t *testing.T) {
	bills := []models.Bill{
		gasBill(1, day(2026, 1, 31), 100, nil),
		{ID: 2, PeriodStart: day(2026, 2, 1), PeriodEnd: day(2026, 2, 28)}, // no readings at all
		gasBill(3, day(2026, 3, 31), 130, nil),
	}

	periods, summary := analyse("gas", bills, 2)

	if len(periods) != 2 {
		t.Fatalf("got %d periods, want 2", len(periods))
	}
	for i, p := range periods {
		if p.ProviderConsumption != nil || p.UserConsumption != nil || p.Difference != nil {
			t.Errorf("period %d touches the reading-less bill, so it must carry no figures: %+v", i, p)
		}
	}
	if summary.TotalProvider != 0 || summary.TotalUser != 0 {
		t.Errorf("gaps must not add to the totals: %+v", summary)
	}
}

func TestConsumptionAnalysis_ElectricityBands(t *testing.T) {
	bill := func(id uint, end time.Time, f1, f2 float64, f3 *float64, user *models.MeterReading) models.Bill {
		return models.Bill{
			ID: id, PeriodStart: end.AddDate(0, -1, 0), PeriodEnd: end,
			ProviderReadingF1: fp(f1), ProviderReadingF2: fp(f2), ProviderReadingF3: f3,
			UserReading: user,
		}
	}
	bills := []models.Bill{
		bill(1, day(2026, 1, 31), 1000, 2000, fp(3000), nil),
		// Self-reading only covers F1 and F2; F3 must fall back to the provider's.
		bill(2, day(2026, 2, 28), 1100, 2300, fp(3050), &models.MeterReading{ValueF1: fp(1090), ValueF2: fp(2250)}),
	}

	periods, summary := analyse("electricity", bills, 100)
	p := periods[0]

	if !approx(*p.ProviderConsumptionF1, 100) || !approx(*p.ProviderConsumptionF2, 300) || !approx(*p.ProviderConsumptionF3, 50) {
		t.Errorf("billed bands = %.0f/%.0f/%.0f, want 100/300/50", *p.ProviderConsumptionF1, *p.ProviderConsumptionF2, *p.ProviderConsumptionF3)
	}
	if !approx(*p.ProviderConsumption, 450) {
		t.Errorf("billed total = %.0f, want 450 (F1+F2+F3)", *p.ProviderConsumption)
	}
	// The previous bill has no self-reading, so its effective values are the
	// provider's: user F1 = 1090-1000, F2 = 2250-2000, F3 = 3050-3000.
	if !approx(*p.UserConsumptionF1, 90) || !approx(*p.UserConsumptionF2, 250) || !approx(*p.UserConsumptionF3, 50) {
		t.Errorf("user bands = %.0f/%.0f/%.0f, want 90/250/50", *p.UserConsumptionF1, *p.UserConsumptionF2, *p.UserConsumptionF3)
	}
	if !approx(*p.DifferenceF1, 10) || !approx(*p.DifferenceF2, 50) || !approx(*p.DifferenceF3, 0) {
		t.Errorf("band differences = %.0f/%.0f/%.0f, want 10/50/0", *p.DifferenceF1, *p.DifferenceF2, *p.DifferenceF3)
	}
	if !approx(*p.Difference, 60) || !approx(summary.CumulativeDifference, 60) {
		t.Errorf("total difference = %.0f / cumulative %.0f, want 60", *p.Difference, summary.CumulativeDifference)
	}
	if !approx(summary.CumulativeDifferenceF2, 50) {
		t.Errorf("CumulativeDifferenceF2 = %.0f, want 50", summary.CumulativeDifferenceF2)
	}
}

func TestConsumptionAnalysis_ElectricityPartialBands(t *testing.T) {
	// A bill that only reports F1 (no F2/F3): the other bands must stay unset
	// rather than appearing as zero consumption.
	mk := func(id uint, end time.Time, f1 float64) models.Bill {
		return models.Bill{ID: id, PeriodStart: end.AddDate(0, -1, 0), PeriodEnd: end, ProviderReadingF1: fp(f1)}
	}
	periods, summary := analyse("electricity", []models.Bill{mk(1, day(2026, 1, 31), 500), mk(2, day(2026, 2, 28), 560)}, 2)

	p := periods[0]
	if p.ProviderConsumptionF2 != nil || p.ProviderConsumptionF3 != nil {
		t.Error("F2/F3 consumption must stay nil when the bill has no such reading")
	}
	if p.ProviderConsumption == nil || !approx(*p.ProviderConsumption, 60) {
		t.Errorf("billed total = %v, want 60", p.ProviderConsumption)
	}
	if !approx(summary.TotalProviderF1, 60) {
		t.Errorf("TotalProviderF1 = %.0f, want 60", summary.TotalProviderF1)
	}
}

func TestConsumptionAnalysis_CumulativeAlertLevels(t *testing.T) {
	// One period, threshold 2 → warning above 2, alert above 4, an
	// informational credit (no alert) below -2.
	run := func(billed, own float64) *ConsumptionSummary {
		_, s := analyse("gas", []models.Bill{
			gasBill(1, day(2026, 1, 31), 100, fp(100)),
			gasBill(2, day(2026, 2, 28), 100+billed, fp(100+own)),
		}, 2)
		return s
	}

	cases := []struct {
		name        string
		billed, own float64
		wantAlert   bool
		wantLevel   string
		wantCredit  bool
	}{
		{"within threshold", 10, 9, false, "", false},
		{"exactly at threshold", 12, 10, false, "", false},
		{"warning", 13, 10, true, "warning", false},
		{"exactly at alert boundary", 14, 10, true, "warning", false},
		{"alert", 15, 10, true, "alert", false},
		{"provider billed less: credit only", 7, 10, false, "", true},
		{"exactly at the under-billing boundary", 8, 10, false, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := run(tc.billed, tc.own)
			if s.HasCumulativeAlert != tc.wantAlert || s.CumulativeAlertLevel != tc.wantLevel {
				t.Errorf("alert=%v level=%q, want alert=%v level=%q", s.HasCumulativeAlert, s.CumulativeAlertLevel, tc.wantAlert, tc.wantLevel)
			}
			if s.CumulativeCredit != tc.wantCredit {
				t.Errorf("credit = %v, want %v", s.CumulativeCredit, tc.wantCredit)
			}
		})
	}
}

func TestConsumptionAnalysis_ReportsPeriodCountForTheClient(t *testing.T) {
	// The cumulative wording lives in the client; it needs the period count.
	bills := []models.Bill{
		gasBill(1, day(2026, 1, 31), 100, nil),
		gasBill(2, day(2026, 2, 28), 110, nil),
		gasBill(3, day(2026, 3, 31), 120, nil),
	}

	_, s := analyse("gas", bills, 2)

	if s.PeriodCount != 2 {
		t.Errorf("PeriodCount = %d, want 2", s.PeriodCount)
	}
}

func TestConsumptionAnalysis_ThresholdScalesWithPeriods(t *testing.T) {
	// Two periods, each 3 over: cumulative 6 against a threshold of 2×2=4 is a
	// warning, not an alert (alert needs more than 8).
	bills := []models.Bill{
		gasBill(1, day(2026, 1, 31), 100, fp(100)),
		gasBill(2, day(2026, 2, 28), 113, fp(110)),
		gasBill(3, day(2026, 3, 31), 126, fp(120)),
	}

	_, s := analyse("gas", bills, 2)

	if !s.HasCumulativeAlert || s.CumulativeAlertLevel != "warning" {
		t.Errorf("level = %q (alert=%v), want warning", s.CumulativeAlertLevel, s.HasCumulativeAlert)
	}
}

// ── CompareReadings (HTTP) ──────────────────────────────────────────────────

type compareResponse struct {
	Comparisons        []ReadingComparison `json:"comparisons"`
	ReadingMatchDays   int                 `json:"reading_match_days"`
	ConsumptionSummary *ConsumptionSummary `json:"consumption_summary"`
}

func (f *moneyFixture) compareReadings(t *testing.T, token string, utilityID uint, query string) (int, compareResponse) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.GET("/utilities/:id/compare-readings", NewUtilityHandler(f.db).CompareReadings)

	path := "/utilities/" + itoa(utilityID) + "/compare-readings"
	if query != "" {
		path += "?" + query
	}
	rec := doGET(t, r, path, token)
	var out compareResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode comparison: %v", err)
		}
	}
	return rec.Code, out
}

func (f *moneyFixture) gasUtility(t *testing.T) models.Utility {
	t.Helper()
	u := models.Utility{
		UserID: f.alice.ID, PropertyID: f.prop.ID, Type: "gas", Provider: "Test",
		IsMetered: true, IsActive: true, PaidByMemberID: &f.mAlice.ID,
	}
	mustCreate(t, f.db, &u)
	return u
}

func TestCompareReadings_StatusAndStructuredDifference(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	// Default thresholds: base 2, 1 per day of gap. Each self-reading sits on
	// its bill's period end, so there is no day gap and the threshold stays 2.
	periods := []struct {
		end            time.Time
		billed, own    float64
		wantStatus     string
		wantDifference float64
	}{
		{day(2026, 1, 31), 100, 100, "ok", 0},
		{day(2026, 2, 28), 120, 117, "warning", 3},
		{day(2026, 3, 31), 140, 130, "alert", 10},
	}
	for i, p := range periods {
		mustCreate(t, f.db,
			&models.Bill{
				UtilityID: u.ID, BillNumber: "B-" + itoa(uint(i+1)), IssueDate: p.end, DueDate: p.end.AddDate(0, 0, 20),
				PeriodStart: p.end.AddDate(0, -1, 1), PeriodEnd: p.end, AmountTotal: 10, ProviderReading: fp(p.billed),
			},
			&models.MeterReading{UtilityID: u.ID, ReadingDate: p.end, Value: fp(p.own)},
		)
	}

	code, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if len(got.Comparisons) != 3 {
		t.Fatalf("got %d comparisons, want 3", len(got.Comparisons))
	}
	// The handler lists the newest bill first.
	for i, p := range []int{2, 1, 0} {
		c, want := got.Comparisons[i], periods[p]
		if c.Status != want.wantStatus {
			t.Errorf("bill ending %s: status %q, want %q", want.end.Format("2006-01-02"), c.Status, want.wantStatus)
		}
		if c.MaxAbsDifference == nil || !approx(*c.MaxAbsDifference, want.wantDifference) {
			t.Errorf("bill ending %s: max_abs_difference = %v, want %.0f", want.end.Format("2006-01-02"), c.MaxAbsDifference, want.wantDifference)
		}
		if !approx(c.EffectiveThreshold, 2) || c.DaysDifference != 0 {
			t.Errorf("threshold %.1f over %d days, want 2 over 0", c.EffectiveThreshold, c.DaysDifference)
		}
	}
	if got.ConsumptionSummary == nil || got.ConsumptionSummary.PeriodCount != 2 {
		t.Errorf("summary = %+v, want 2 periods", got.ConsumptionSummary)
	}
}

func TestCompareReadings_ThresholdGrowsWithDayGap(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	end := day(2026, 2, 28)
	mustCreate(t, f.db,
		&models.Bill{
			UtilityID: u.ID, BillNumber: "B-1", IssueDate: end, DueDate: end.AddDate(0, 0, 20),
			PeriodStart: day(2026, 2, 1), PeriodEnd: end, AmountTotal: 10, ProviderReading: fp(120),
		},
		// Read 5 days before the period end: the allowance becomes 2 + 5×1 = 7.
		&models.MeterReading{UtilityID: u.ID, ReadingDate: end.AddDate(0, 0, -5), Value: fp(115)},
	)

	_, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	c := got.Comparisons[0]
	if c.DaysDifference != 5 || !approx(c.EffectiveThreshold, 7) {
		t.Errorf("gap %d days / threshold %.1f, want 5 / 7", c.DaysDifference, c.EffectiveThreshold)
	}
	if c.Status != "ok" {
		t.Errorf("a 5-unit gap inside a 7-unit allowance must be ok, got %q", c.Status)
	}
}

func TestCompareReadings_NoSelfReadingIsNoData(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	end := day(2026, 2, 28)
	mustCreate(t, f.db, &models.Bill{
		UtilityID: u.ID, BillNumber: "B-1", IssueDate: end, DueDate: end.AddDate(0, 0, 20),
		PeriodStart: day(2026, 2, 1), PeriodEnd: end, AmountTotal: 10, ProviderReading: fp(120),
	})

	_, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	if len(got.Comparisons) != 1 || got.Comparisons[0].Status != "no_data" {
		t.Fatalf("comparisons = %+v, want one no_data", got.Comparisons)
	}
	if got.Comparisons[0].MaxAbsDifference != nil {
		t.Error("nothing was compared, so there is no difference to report")
	}
	if got.Comparisons[0].NoDataReason != "no_readings" {
		t.Errorf("reason = %q, want no_readings", got.Comparisons[0].NoDataReason)
	}
}

func TestCompareReadings_FarAwayReadingIsDeclaredNotUsed(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	end := day(2026, 2, 28)
	mustCreate(t, f.db,
		&models.Bill{
			UtilityID: u.ID, BillNumber: "B-1", IssueDate: end, DueDate: end.AddDate(0, 0, 20),
			PeriodStart: day(2026, 2, 1), PeriodEnd: end, AmountTotal: 10, ProviderReading: fp(120),
		},
		// Four months before the period: far outside the default 15-day window.
		&models.MeterReading{UtilityID: u.ID, ReadingDate: day(2025, 10, 1), Value: fp(80)},
	)

	_, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	c := got.Comparisons[0]
	if c.Status != "no_data" || c.NoDataReason != "out_of_range" {
		t.Fatalf("status %q / reason %q, want no_data / out_of_range (a 4-month-old reading must not pass for ok)", c.Status, c.NoDataReason)
	}
	if c.UserReading != nil || c.MaxAbsDifference != nil {
		t.Errorf("the far-away reading must not be compared: %+v", c)
	}
	if c.NearestReadingDate == nil || !c.NearestReadingDate.Equal(day(2025, 10, 1)) {
		t.Errorf("nearest reading = %v, want 2025-10-01", c.NearestReadingDate)
	}
	if c.NearestGapDays == nil || *c.NearestGapDays != 123 {
		t.Errorf("gap = %v days, want 123 (Oct 1 → Feb 1)", c.NearestGapDays)
	}
	if got.ReadingMatchDays != 15 {
		t.Errorf("window = %d, want the default 15", got.ReadingMatchDays)
	}
}

func TestCompareReadings_WideningTheWindowBringsTheReadingBack(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	end := day(2026, 2, 28)
	mustCreate(t, f.db,
		&models.Bill{
			UtilityID: u.ID, BillNumber: "B-1", IssueDate: end, DueDate: end.AddDate(0, 0, 20),
			PeriodStart: day(2026, 2, 1), PeriodEnd: end, AmountTotal: 10, ProviderReading: fp(120),
		},
		&models.MeterReading{UtilityID: u.ID, ReadingDate: day(2026, 1, 1), Value: fp(110)}, // 31 days before
	)

	_, narrow := f.compareReadings(t, f.aliceTok, u.ID, "")
	if narrow.Comparisons[0].Status != "no_data" {
		t.Fatalf("31 days is outside the default window, got %q", narrow.Comparisons[0].Status)
	}

	if err := f.db.Model(&u).Update("reading_match_days", 45).Error; err != nil {
		t.Fatalf("widen window: %v", err)
	}
	_, wide := f.compareReadings(t, f.aliceTok, u.ID, "")
	c := wide.Comparisons[0]
	if c.Status == "no_data" || c.UserReading == nil {
		t.Fatalf("with a 45-day window the reading must be used: %+v", c)
	}
	if wide.ReadingMatchDays != 45 {
		t.Errorf("window = %d, want 45", wide.ReadingMatchDays)
	}

	// The query parameter overrides the stored value, like the thresholds do.
	_, overridden := f.compareReadings(t, f.aliceTok, u.ID, "reading_match_days=10")
	if overridden.Comparisons[0].Status != "no_data" || overridden.ReadingMatchDays != 10 {
		t.Errorf("override: status %q window %d, want no_data / 10", overridden.Comparisons[0].Status, overridden.ReadingMatchDays)
	}
}

func TestMatchReading(t *testing.T) {
	start, end := day(2026, 2, 1), day(2026, 2, 28)
	r := func(id uint, d time.Time) models.MeterReading {
		return models.MeterReading{ID: id, ReadingDate: d}
	}

	t.Run("inside the period beats a nearer-looking outside reading", func(t *testing.T) {
		readings := []models.MeterReading{r(1, day(2026, 3, 1)), r(2, day(2026, 2, 10))}
		if m, _, _ := matchReading(readings, start, end, 15); m == nil || m.ID != 2 {
			t.Errorf("match = %+v, want reading 2", m)
		}
	})
	t.Run("nearest within the window wins", func(t *testing.T) {
		readings := []models.MeterReading{r(1, day(2026, 1, 10)), r(2, day(2026, 1, 25))}
		if m, _, _ := matchReading(readings, start, end, 30); m == nil || m.ID != 2 {
			t.Errorf("match = %+v, want reading 2 (7 days away, not 22)", m)
		}
	})
	t.Run("the window edge is inclusive", func(t *testing.T) {
		readings := []models.MeterReading{r(1, day(2026, 1, 17))}
		if m, _, _ := matchReading(readings, start, end, 15); m == nil {
			t.Error("a reading exactly 15 days before must match")
		}
		if m, _, _ := matchReading(readings, start, end, 14); m != nil {
			t.Error("a reading 15 days before must not match a 14-day window")
		}
	})
	t.Run("outside the window there is no match but a nearest", func(t *testing.T) {
		readings := []models.MeterReading{r(1, day(2025, 10, 1)), r(2, day(2025, 12, 1))}
		m, nearest, gap := matchReading(readings, start, end, 15)
		if m != nil || nearest == nil || nearest.ID != 2 || int(gap) != 62 {
			t.Errorf("match=%v nearest=%+v gap=%.0f, want nil / reading 2 / 62", m, nearest, gap)
		}
	})
	t.Run("after the period counts too", func(t *testing.T) {
		readings := []models.MeterReading{r(1, day(2026, 3, 10))}
		if m, _, gap := matchReading(readings, start, end, 15); m == nil || int(gap) != 10 {
			t.Errorf("match=%v gap=%.0f, want a match 10 days after", m, gap)
		}
	})
	t.Run("equal distance keeps the first reading", func(t *testing.T) {
		readings := []models.MeterReading{r(1, day(2026, 2, 10)), r(2, day(2026, 2, 12))}
		if m, _, _ := matchReading(readings, start, end, 15); m == nil || m.ID != 1 {
			t.Errorf("match = %+v, want the first (newest) of two in-period readings", m)
		}
	})
	t.Run("no readings at all", func(t *testing.T) {
		if m, n, _ := matchReading(nil, start, end, 15); m != nil || n != nil {
			t.Error("expected nil/nil")
		}
	})
}

func TestCompareReadings_OnlyForHouseholdMembers(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	eve := &models.User{Email: "eve@example.com", PasswordHash: "x", Name: "Eve", Role: "user", IsActive: true}
	mustCreate(t, f.db, eve)

	if code, _ := f.compareReadings(t, testutil.SignToken(t, eve), u.ID, ""); code != http.StatusNotFound {
		t.Errorf("a non-member got status %d, want 404", code)
	}
}

func TestCompareReadings_ElectricityUsesTheWidestBandGap(t *testing.T) {
	f := setupMoneyFixture(t)
	u := models.Utility{
		UserID: f.alice.ID, PropertyID: f.prop.ID, Type: "electricity", Provider: "Test",
		IsMetered: true, IsActive: true, PaidByMemberID: &f.mAlice.ID,
	}
	mustCreate(t, f.db, &u)
	cases := []struct {
		end               time.Time
		billed, own       [3]float64
		wantStatus        string
		wantMaxDifference float64
	}{
		// Bands differ by 1, 10 and 0: the widest (10) is above 2×2.
		{day(2026, 1, 31), [3]float64{1000, 2000, 3000}, [3]float64{1001, 1990, 3000}, "alert", 10},
		// Widest gap 3: above the allowance of 2, not above 4.
		{day(2026, 2, 28), [3]float64{1100, 2100, 3100}, [3]float64{1100, 2100, 3097}, "warning", 3},
		{day(2026, 3, 31), [3]float64{1200, 2200, 3200}, [3]float64{1201, 2200, 3200}, "ok", 1},
	}
	for i, c := range cases {
		mustCreate(t, f.db,
			&models.Bill{
				UtilityID: u.ID, BillNumber: "E-" + itoa(uint(i+1)), IssueDate: c.end, DueDate: c.end.AddDate(0, 0, 20),
				PeriodStart: c.end.AddDate(0, -1, 1), PeriodEnd: c.end, AmountTotal: 10,
				ProviderReadingF1: fp(c.billed[0]), ProviderReadingF2: fp(c.billed[1]), ProviderReadingF3: fp(c.billed[2]),
			},
			&models.MeterReading{UtilityID: u.ID, ReadingDate: c.end, ValueF1: fp(c.own[0]), ValueF2: fp(c.own[1]), ValueF3: fp(c.own[2])},
		)
	}

	code, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	if code != http.StatusOK || len(got.Comparisons) != 3 {
		t.Fatalf("status %d with %d comparisons, want 200 with 3", code, len(got.Comparisons))
	}
	// Newest bill first.
	for i, idx := range []int{2, 1, 0} {
		c, want := got.Comparisons[i], cases[idx]
		if c.Status != want.wantStatus {
			t.Errorf("bill ending %s: status %q, want %q", want.end.Format("2006-01-02"), c.Status, want.wantStatus)
		}
		if c.MaxAbsDifference == nil || !approx(*c.MaxAbsDifference, want.wantMaxDifference) {
			t.Errorf("bill ending %s: max_abs_difference = %v, want %.0f", want.end.Format("2006-01-02"), c.MaxAbsDifference, want.wantMaxDifference)
		}
	}
	alert := got.Comparisons[2]
	if alert.DifferenceF2 == nil || !approx(*alert.DifferenceF2, 10) || alert.UserF2 == nil || !approx(*alert.UserF2, 1990) {
		t.Errorf("F2 detail = diff %v user %v, want 10 / 1990", alert.DifferenceF2, alert.UserF2)
	}
}

func TestCompareReadings_ElectricityWithoutReadingsIsNoData(t *testing.T) {
	f := setupMoneyFixture(t)
	u := models.Utility{
		UserID: f.alice.ID, PropertyID: f.prop.ID, Type: "electricity", Provider: "Test",
		IsMetered: true, IsActive: true, PaidByMemberID: &f.mAlice.ID,
	}
	mustCreate(t, f.db, &u)
	end := day(2026, 1, 31)
	mustCreate(t, f.db, &models.Bill{
		UtilityID: u.ID, BillNumber: "E-1", IssueDate: end, DueDate: end.AddDate(0, 0, 20),
		PeriodStart: day(2026, 1, 1), PeriodEnd: end, AmountTotal: 10, ProviderReadingF1: fp(1000),
	})

	_, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	if len(got.Comparisons) != 1 || got.Comparisons[0].Status != "no_data" || got.Comparisons[0].NoDataReason != "no_readings" {
		t.Errorf("comparisons = %+v, want one no_data / no_readings", got.Comparisons)
	}
}

func TestCompareReadings_OwnReadingAboveTheProviderIsStillADifference(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	end := day(2026, 2, 28)
	mustCreate(t, f.db,
		&models.Bill{
			UtilityID: u.ID, BillNumber: "B-1", IssueDate: end, DueDate: end.AddDate(0, 0, 20),
			PeriodStart: day(2026, 2, 1), PeriodEnd: end, AmountTotal: 10, ProviderReading: fp(100),
		},
		// The household reads 10 more than the provider billed.
		&models.MeterReading{UtilityID: u.ID, ReadingDate: end, Value: fp(110)},
	)

	_, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	c := got.Comparisons[0]
	if c.Status != "alert" || c.MaxAbsDifference == nil || !approx(*c.MaxAbsDifference, 10) {
		t.Errorf("status %q, max difference %v; want alert / 10 (the sign must not hide the gap)", c.Status, c.MaxAbsDifference)
	}
	if c.Difference == nil || !approx(*c.Difference, -10) {
		t.Errorf("signed difference = %v, want -10", c.Difference)
	}
}

func TestCompareReadings_StoredZeroWindowFallsBackToTheDefault(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	if err := f.db.Model(&u).Update("reading_match_days", 0).Error; err != nil {
		t.Fatalf("zero the window: %v", err)
	}

	_, got := f.compareReadings(t, f.aliceTok, u.ID, "")

	if got.ReadingMatchDays != defaultReadingMatchDays {
		t.Errorf("window = %d, want the default %d", got.ReadingMatchDays, defaultReadingMatchDays)
	}
}

func TestUpdateUtility_ReadingMatchDays(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.PUT("/utilities/:id", NewUtilityHandler(f.db).Update)

	stored := func() int {
		var out models.Utility
		if err := f.db.First(&out, u.ID).Error; err != nil {
			t.Fatalf("reload utility: %v", err)
		}
		return out.ReadingMatchDays
	}
	put := func(days int) {
		t.Helper()
		rec := doJSON(t, r, http.MethodPut, "/utilities/"+itoa(u.ID), f.aliceTok, map[string]any{"reading_match_days": days})
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT reading_match_days=%d: status %d, body %s", days, rec.Code, rec.Body.String())
		}
	}

	if got := stored(); got != defaultReadingMatchDays {
		t.Fatalf("a new service starts at %d days, got %d", defaultReadingMatchDays, got)
	}
	put(45)
	if got := stored(); got != 45 {
		t.Errorf("stored window = %d, want 45", got)
	}
	put(100000)
	if got := stored(); got != maxReadingMatchDays {
		t.Errorf("stored window = %d, want it capped at %d", got, maxReadingMatchDays)
	}
	put(0)
	put(-3)
	if got := stored(); got != maxReadingMatchDays {
		t.Errorf("stored window = %d, a zero or negative value must leave it unchanged", got)
	}
}
