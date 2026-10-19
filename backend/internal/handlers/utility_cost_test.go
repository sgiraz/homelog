package handlers

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

// monthBill is a bill covering one calendar month, with a single-register reading.
func monthBill(id uint, y int, m time.Month, amount, billed float64, reading *float64) models.Bill {
	start := day(y, m, 1)
	return models.Bill{
		ID: id, BillNumber: "B-" + itoa(id), PeriodStart: start, PeriodEnd: start.AddDate(0, 1, -1),
		AmountTotal: amount, ConsumptionTotal: billed, ProviderReading: reading,
	}
}

func metered(utilityType string) models.Utility {
	return models.Utility{Type: utilityType, IsMetered: true, ComparisonThreshold: 2}
}

func datePtr(y int, m time.Month, d int) *time.Time {
	t := day(y, m, d)
	return &t
}

func TestCostAnalysis_WindowEdgesAndPreviousWindow(t *testing.T) {
	// Readings grow 100 per month at 0.5/unit until June, then 125 at 0.6/unit.
	bills := []models.Bill{
		monthBill(1, 2026, 1, 40, 100, fp(1000)), // anchor: measured nothing
		monthBill(2, 2026, 2, 50, 100, fp(1100)),
		monthBill(3, 2026, 3, 50, 100, fp(1200)),
		monthBill(4, 2026, 4, 75, 125, fp(1325)),
		monthBill(5, 2026, 5, 75, 125, fp(1450)),
	}

	got := computeCostAnalysis(metered("water"), bills, datePtr(2026, 4, 30), datePtr(2026, 5, 31))

	cur := got.Current
	if cur.BillCount != 2 || !approx(cur.Amount, 150) {
		t.Fatalf("current window = %d bills / %.2f, want the bills ending on 30 Apr and 31 May (2 / 150)", cur.BillCount, cur.Amount)
	}
	if !approx(cur.MeasuredConsumption, 250) || cur.UnitCost == nil || !approx(*cur.UnitCost, 0.6) {
		t.Errorf("measured %.1f, unit cost %v; want 250 at 0.6", cur.MeasuredConsumption, cur.UnitCost)
	}
	if cur.Days != 61 || cur.CostPerDay == nil || !approx(*cur.CostPerDay, 150.0/61) {
		t.Errorf("days %d, cost/day %v; want 61 days", cur.Days, cur.CostPerDay)
	}

	// 31 days back from 30 Apr: 30 Mar → 29 Apr holds only the March bill.
	prev := got.Previous
	if prev == nil || prev.BillCount != 1 || !approx(prev.Amount, 50) {
		t.Fatalf("previous window = %+v, want the March bill alone", prev)
	}
	if prev.UnitCost == nil || !approx(*prev.UnitCost, 0.5) {
		t.Errorf("previous unit cost %v, want 0.5", prev.UnitCost)
	}
	if got.UnitCostChange == nil || !approx(*got.UnitCostChange, 0.2) {
		t.Errorf("unit cost change %v, want +20%%", got.UnitCostChange)
	}
	if got.CostPerDayChange == nil || !approx(*got.CostPerDayChange, (150.0/61)/(50.0/31)-1) {
		t.Errorf("cost/day change %v", got.CostPerDayChange)
	}
}

func TestCostAnalysis_OpenWindowHasNoComparison(t *testing.T) {
	bills := []models.Bill{
		monthBill(1, 2026, 1, 40, 100, fp(1000)),
		monthBill(2, 2026, 2, 50, 100, fp(1100)),
	}
	for _, w := range [][2]*time.Time{{nil, nil}, {datePtr(2026, 1, 1), nil}, {nil, datePtr(2026, 12, 31)}} {
		got := computeCostAnalysis(metered("water"), bills, w[0], w[1])
		if got.Previous != nil || got.UnitCostChange != nil {
			t.Errorf("window %v→%v: a window without both ends has nothing to compare with", w[0], w[1])
		}
		if got.Current.BillCount != 2 {
			t.Errorf("window %v→%v: %d bills, want 2", w[0], w[1], got.Current.BillCount)
		}
	}

	got := computeCostAnalysis(metered("water"), bills, datePtr(2026, 1, 1), datePtr(2026, 1, 31))
	if got.Previous != nil {
		t.Errorf("a previous window with no bills must be omitted, got %+v", got.Previous)
	}
}

func TestCostAnalysis_GasUsesTheConversionCoefficient(t *testing.T) {
	bills := []models.Bill{
		monthBill(1, 2026, 1, 0, 0, fp(1000)),
		monthBill(2, 2026, 2, 100, 102, fp(1100)),
		monthBill(3, 2026, 3, 100, 102, fp(1200)),
	}
	bills[1].ConversionCoefficient = fp(1.02)
	// bill 3 carries no coefficient: the last known one still applies

	got := computeCostAnalysis(metered("gas"), bills, nil, nil).Current

	if !approx(got.MeasuredConsumption, 204) || !approx(got.BilledConsumption, 204) {
		t.Errorf("measured %.3f / billed %.3f Smc, want 204 / 204", got.MeasuredConsumption, got.BilledConsumption)
	}
	if got.ConsumptionGap == nil || !approx(*got.ConsumptionGap, 0) || got.GapLevel != "" {
		t.Errorf("gap %v level %q, want 0 and no flag", got.ConsumptionGap, got.GapLevel)
	}
}

func TestCostAnalysis_SelfReadingsAreTheMeasure(t *testing.T) {
	bills := []models.Bill{
		monthBill(1, 2026, 1, 0, 0, fp(1000)),
		monthBill(2, 2026, 2, 60, 100, fp(1100)),
	}
	bills[1].UserReading = &models.MeterReading{Value: fp(1080)}

	got := computeCostAnalysis(metered("water"), bills, nil, nil).Current

	if !approx(got.MeasuredConsumption, 80) || got.UnitCost == nil || !approx(*got.UnitCost, 0.75) {
		t.Errorf("measured %.1f, unit cost %v; want 80 at 0.75", got.MeasuredConsumption, got.UnitCost)
	}
	// billed 100 vs measured 80, threshold 2 per bill
	if got.ConsumptionGap == nil || !approx(*got.ConsumptionGap, 20) || got.GapLevel != "alert" {
		t.Errorf("gap %v level %q, want +20 alert", got.ConsumptionGap, got.GapLevel)
	}
}

func TestCostAnalysis_GapLevels(t *testing.T) {
	cases := []struct {
		billed float64
		want   string
	}{
		{102, ""}, {103, "warning"}, {105, "alert"}, {98, ""}, {97, "credit"},
	}
	for _, tc := range cases {
		bills := []models.Bill{
			monthBill(1, 2026, 1, 0, 0, fp(1000)),
			monthBill(2, 2026, 2, 50, tc.billed, fp(1100)),
		}
		got := computeCostAnalysis(metered("water"), bills, nil, nil).Current
		if got.GapLevel != tc.want {
			t.Errorf("billed %.0f vs measured 100: level %q, want %q", tc.billed, got.GapLevel, tc.want)
		}
	}
}

func TestCostAnalysis_MissingReadingsLeaveBillsUnmeasured(t *testing.T) {
	bills := []models.Bill{
		monthBill(1, 2026, 1, 40, 100, fp(1000)),
		monthBill(2, 2026, 2, 50, 100, nil),
		monthBill(3, 2026, 3, 60, 100, fp(1200)),
	}

	got := computeCostAnalysis(metered("water"), bills, nil, nil).Current

	if got.BillCount != 3 || !approx(got.Amount, 150) {
		t.Errorf("all bills still count for spending: %d / %.0f", got.BillCount, got.Amount)
	}
	if got.MeasuredBillCount != 0 || got.UnitCost != nil || got.ConsumptionGap != nil {
		t.Errorf("no consecutive readings, nothing measured: %+v", got)
	}
}

func TestCostAnalysis_ElectricityBandsAndDecrease(t *testing.T) {
	band := func(id uint, m time.Month, amount, f1, f2, f3 float64) models.Bill {
		b := monthBill(id, 2026, m, amount, 0, nil)
		b.ProviderReadingF1, b.ProviderReadingF2, b.ProviderReadingF3 = fp(f1), fp(f2), fp(f3)
		return b
	}
	bills := []models.Bill{
		band(1, 1, 0, 100, 200, 300),
		band(2, 2, 60, 150, 250, 400), // +200 kWh
		band(3, 3, 60, 140, 300, 450), // F1 went back: not consumption
		band(4, 4, 30, 190, 320, 480), // +100 kWh
	}
	bills[1].ConsumptionTotal, bills[3].ConsumptionTotal = 200, 100

	got := computeCostAnalysis(metered("electricity"), bills, nil, nil)

	if got.Current.MeasuredBillCount != 2 || !approx(got.Current.MeasuredConsumption, 300) {
		t.Errorf("measured %d bills / %.0f kWh, want 2 / 300 (the bill with a negative band is skipped)",
			got.Current.MeasuredBillCount, got.Current.MeasuredConsumption)
	}
	if got.Current.UnitCost == nil || !approx(*got.Current.UnitCost, 0.3) {
		t.Errorf("unit cost %v, want 90/300", got.Current.UnitCost)
	}
	if len(got.Anomalies) != 1 {
		t.Fatalf("anomalies = %+v, want one decrease", got.Anomalies)
	}
	a := got.Anomalies[0]
	if a.Kind != "reading_decrease" || a.BillID != 3 || a.Register != "f1" || !approx(*a.Previous, 150) || !approx(*a.Current, 140) {
		t.Errorf("anomaly = %+v, want F1 150 → 140 on bill 3", a)
	}
}

func TestCostAnalysis_DecreaseComparesWithTheLastKnownReading(t *testing.T) {
	bills := []models.Bill{
		monthBill(1, 2026, 1, 0, 0, fp(500)),
		monthBill(2, 2026, 2, 0, 0, nil),
		monthBill(3, 2026, 3, 0, 0, fp(480)),
	}

	got := computeCostAnalysis(metered("gas"), bills, nil, nil).Anomalies

	if len(got) != 1 || got[0].BillID != 3 || got[0].Register != "" || !approx(*got[0].Previous, 500) {
		t.Errorf("anomalies = %+v, want 500 → 480 on bill 3", got)
	}
}

func TestCostAnalysis_ConsecutiveEstimates(t *testing.T) {
	bills := make([]models.Bill, 7)
	types := []string{"estimated", "actual", "estimated", "estimated", "estimated", "actual", "estimated"}
	for i, rt := range types {
		bills[i] = monthBill(uint(i+1), 2026, time.Month(i+1), 0, 0, fp(float64(100*i)))
		bills[i].ReadingType = rt
	}

	got := computeCostAnalysis(metered("water"), bills, nil, nil).Anomalies

	if len(got) != 1 {
		t.Fatalf("anomalies = %+v, want one run (a lone estimate is not flagged)", got)
	}
	if got[0].Kind != "consecutive_estimates" || got[0].Count != 3 || got[0].BillID != 5 {
		t.Errorf("anomaly = %+v, want 3 estimates in a row ending at bill 5", got[0])
	}

	if a := computeCostAnalysis(metered("water"), bills, datePtr(2026, 6, 1), nil).Anomalies; len(a) != 0 {
		t.Errorf("a run ending before the window must be left out, got %+v", a)
	}
}

func TestCostAnalysis_NotMeteredCostsPerDay(t *testing.T) {
	// Overlapping and gapped periods: 1–31 Jan, 15 Jan–15 Feb, 1–31 Mar.
	bills := []models.Bill{
		{ID: 1, PeriodStart: day(2026, 1, 1), PeriodEnd: day(2026, 1, 31), AmountTotal: 31, ProviderReading: fp(1)},
		{ID: 2, PeriodStart: day(2026, 1, 15), PeriodEnd: day(2026, 2, 15), AmountTotal: 15, ProviderReading: fp(0)},
		{ID: 3, PeriodStart: day(2026, 3, 1), PeriodEnd: day(2026, 3, 31), AmountTotal: 31, ProviderReading: fp(5), ReadingType: "estimated"},
	}
	rent := models.Utility{Type: "affitto"}

	got := computeCostAnalysis(rent, bills, nil, nil)

	if got.Current.Days != 46+31 || got.Current.CostPerDay == nil || !approx(*got.Current.CostPerDay, 1) {
		t.Errorf("days %d, cost/day %v; want 77 days at 1/day", got.Current.Days, got.Current.CostPerDay)
	}
	if got.Current.UnitCost != nil || got.Current.ConsumptionGap != nil || got.Current.MeasuredBillCount != 0 {
		t.Errorf("a service without a meter has no unit cost: %+v", got.Current)
	}
	if got.Anomalies == nil || len(got.Anomalies) != 0 {
		t.Errorf("anomalies = %#v, want an empty list", got.Anomalies)
	}
}

// ── GetCostAnalysis (HTTP) ──────────────────────────────────────────────────

func (f *moneyFixture) costAnalysis(t *testing.T, token string, utilityID uint, query string) (int, CostAnalysis) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	p := r.Group("")
	p.Use(middleware.AuthRequired())
	p.GET("/utilities/:id/cost-analysis", NewUtilityHandler(f.db).GetCostAnalysis)

	rec := doGET(t, r, "/utilities/"+itoa(utilityID)+"/cost-analysis?"+query, token)
	var out CostAnalysis
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode cost analysis: %v", err)
		}
	}
	return rec.Code, out
}

func TestGetCostAnalysis(t *testing.T) {
	f := setupMoneyFixture(t)
	u := f.gasUtility(t)
	for i, reading := range []float64{1000, 1100, 1250} {
		end := day(2026, time.Month(i+2), 1).AddDate(0, 0, -1)
		mustCreate(t, f.db, &models.Bill{
			UtilityID: u.ID, BillNumber: "B-" + itoa(uint(i+1)), IssueDate: end, DueDate: end,
			PeriodStart: day(2026, time.Month(i+1), 1), PeriodEnd: end, AmountTotal: 100, ProviderReading: fp(reading),
		})
	}
	mustCreate(t, f.db, &models.MeterReading{UtilityID: u.ID, ReadingDate: day(2026, 3, 31), Value: fp(1200)})
	var reading models.MeterReading
	f.db.Last(&reading)
	f.db.Model(&models.Bill{}).Where("bill_number = ?", "B-3").Update("user_reading_id", reading.ID)

	code, got := f.costAnalysis(t, f.aliceTok, u.ID, "from=2026-02-01&to=2026-03-31")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	// Feb: 100 units; Mar: the self-reading says 100, not the billed 150.
	if got.Current.BillCount != 2 || !approx(got.Current.MeasuredConsumption, 200) || !approx(*got.Current.UnitCost, 1) {
		t.Errorf("current = %+v, want 2 bills, 200 measured at 1/unit", got.Current)
	}

	for _, q := range []string{"from=2026-13-01", "to=yesterday", "from=2026-03-01&to=2026-02-01"} {
		if code, _ := f.costAnalysis(t, f.aliceTok, u.ID, q); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", q, code)
		}
	}

	eve := &models.User{Email: "eve@example.com", PasswordHash: "x", Name: "Eve", Role: "user", IsActive: true}
	mustCreate(t, f.db, eve)
	if code, _ := f.costAnalysis(t, testutil.SignToken(t, eve), u.ID, ""); code != http.StatusNotFound {
		t.Errorf("a non-member got status %d, want 404", code)
	}
}
