package handlers

// calculateConsumptionAnalysis is pure: bills in, periods + summary out. These
// tests pin the algorithm documented on the function — sort by period_end, the
// first bill is an anchor with no row, and a bill without an associated
// self-reading falls back to the provider reading so its contribution is
// neutral.

import (
	"testing"
	"time"

	"github.com/sgiraz/homelog/internal/models"
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
	if summary.HasCumulativeAlert || summary.CumulativeMessage != "" {
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
	// informational message (no alert) below -2. The wording is not asserted
	// here, only the level and whether a message exists.
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
		wantMessage bool
	}{
		{"within threshold", 10, 9, false, "", false},
		{"exactly at threshold", 12, 10, false, "", false},
		{"warning", 13, 10, true, "warning", true},
		{"exactly at alert boundary", 14, 10, true, "warning", true},
		{"alert", 15, 10, true, "alert", true},
		{"provider billed less: message only", 7, 10, false, "", true},
		{"exactly at the under-billing boundary", 8, 10, false, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := run(tc.billed, tc.own)
			if s.HasCumulativeAlert != tc.wantAlert || s.CumulativeAlertLevel != tc.wantLevel {
				t.Errorf("alert=%v level=%q, want alert=%v level=%q", s.HasCumulativeAlert, s.CumulativeAlertLevel, tc.wantAlert, tc.wantLevel)
			}
			if (s.CumulativeMessage != "") != tc.wantMessage {
				t.Errorf("message %q, want present=%v", s.CumulativeMessage, tc.wantMessage)
			}
		})
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
