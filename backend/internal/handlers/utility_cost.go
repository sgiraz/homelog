package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// CostWindow is what a service cost over the bills whose period ends inside a
// window. Unit cost pairs each bill's amount with the consumption measured over
// it, so a bill without a usable reading delta stays out of both sides.
type CostWindow struct {
	From       *time.Time `json:"from,omitempty"`
	To         *time.Time `json:"to,omitempty"`
	BillCount  int        `json:"bill_count"`
	Amount     float64    `json:"amount"`
	Days       int        `json:"days"` // union of the bills' periods
	CostPerDay *float64   `json:"cost_per_day,omitempty"`

	MeasuredBillCount   int      `json:"measured_bill_count"`
	MeasuredAmount      float64  `json:"measured_amount"`
	MeasuredConsumption float64  `json:"measured_consumption"`
	BilledConsumption   float64  `json:"billed_consumption"`
	UnitCost            *float64 `json:"unit_cost,omitempty"`
	// ConsumptionGap is billed minus measured; GapLevel grades it against the
	// comparison threshold per bill: warning, alert, or credit when billed less.
	ConsumptionGap *float64 `json:"consumption_gap,omitempty"`
	GapLevel       string   `json:"gap_level,omitempty"`
}

// ReadingAnomaly flags a bill whose provider reading deserves a second look.
type ReadingAnomaly struct {
	Kind       string    `json:"kind"` // consecutive_estimates, reading_decrease
	BillID     uint      `json:"bill_id"`
	BillNumber string    `json:"bill_number"`
	PeriodEnd  time.Time `json:"period_end"`
	Count      int       `json:"count,omitempty"`    // estimated bills in a row, ending at this one
	Register   string    `json:"register,omitempty"` // f1/f2/f3; empty on a single-register meter
	Previous   *float64  `json:"previous,omitempty"`
	Current    *float64  `json:"current,omitempty"`
}

// CostAnalysis compares a window with the one of equal length before it.
type CostAnalysis struct {
	Current          CostWindow       `json:"current"`
	Previous         *CostWindow      `json:"previous,omitempty"`
	UnitCostChange   *float64         `json:"unit_cost_change,omitempty"` // fraction, 0.1 = +10%
	CostPerDayChange *float64         `json:"cost_per_day_change,omitempty"`
	Anomalies        []ReadingAnomaly `json:"anomalies"`
}

// GetCostAnalysis returns unit cost, cost per day and reading anomalies for the
// bills whose period ends in [from, to] (both optional, YYYY-MM-DD).
// GET /api/v1/utilities/:id/cost-analysis
func (h *UtilityHandler) GetCostAnalysis(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_utility_id", "Invalid service id")
		return
	}

	var from, to *time.Time
	if v := c.Query("from"); v != "" {
		d, err := time.Parse("2006-01-02", v)
		if err != nil {
			apierr.Fail(c, http.StatusBadRequest, "invalid_date_from", "Invalid start date format")
			return
		}
		from = &d
	}
	if v := c.Query("to"); v != "" {
		d, err := time.Parse("2006-01-02", v)
		if err != nil || (from != nil && d.Before(*from)) {
			apierr.Fail(c, http.StatusBadRequest, "invalid_date_to", "Invalid end date format")
			return
		}
		to = &d
	}

	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", utilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		apierr.Fail(c, http.StatusNotFound, "utility_not_found", "Service not found")
		return
	}

	var bills []models.Bill
	query := h.db.Where("utility_id = ?", utilityID).Order("period_end ASC")
	if utility.IsMetered {
		query = query.Preload("UserReading")
	}
	if err := query.Find(&bills).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to fetch bills")
		return
	}

	c.JSON(http.StatusOK, computeCostAnalysis(utility, bills, from, to))
}

// computeCostAnalysis expects bills sorted by period_end ascending.
func computeCostAnalysis(utility models.Utility, bills []models.Bill, from, to *time.Time) CostAnalysis {
	threshold := utility.ComparisonThreshold
	if threshold == 0 {
		threshold = 2.0
	}

	var measured map[uint]measure
	anomalies := []ReadingAnomaly{}
	if utility.IsMetered {
		measured = measuredConsumption(utility.Type, bills)
		for _, a := range readingAnomalies(utility.Type, bills) {
			if inWindow(a.PeriodEnd, from, to) {
				anomalies = append(anomalies, a)
			}
		}
	}

	result := CostAnalysis{
		Current:   costWindow(bills, measured, from, to, threshold),
		Anomalies: anomalies,
	}
	if from == nil || to == nil {
		return result
	}

	days := int(to.Sub(*from).Hours() / 24)
	prevFrom, prevTo := from.AddDate(0, 0, -days), from.AddDate(0, 0, -1)
	prev := costWindow(bills, measured, &prevFrom, &prevTo, threshold)
	if prev.BillCount == 0 {
		return result
	}
	result.Previous = &prev
	result.UnitCostChange = relativeChange(result.Current.UnitCost, prev.UnitCost)
	result.CostPerDayChange = relativeChange(result.Current.CostPerDay, prev.CostPerDay)
	return result
}

// measure is the consumption over one bill: actual from the self-readings,
// billed from the provider's closing readings.
type measure struct{ actual, billed float64 }

// measuredConsumption maps each bill to the consumption its readings measure,
// reusing calculateConsumptionAnalysis (self-reading, else provider reading).
// Billed consumption is the delta of the provider's closing readings, which
// nets out a re-billed estimate; the printed total only stands in when a
// reading is missing. Gas meters count m³: the bill's conversion coefficient
// (the last known one when missing) turns that into the Smc the bill charges.
// A negative delta is a meter swap or a typo, not consumption, so that bill is
// left unmeasured.
func measuredConsumption(utilityType string, bills []models.Bill) map[uint]measure {
	periods, _ := (&UtilityHandler{}).calculateConsumptionAnalysis(utilityType, bills, 0)
	byBill := make(map[uint]ConsumptionPeriod, len(periods))
	for _, p := range periods {
		byBill[*p.BillID] = p
	}

	measured := make(map[uint]measure, len(periods))
	coefficient := 1.0
	for _, b := range bills {
		if b.ConversionCoefficient != nil && *b.ConversionCoefficient > 0 {
			coefficient = *b.ConversionCoefficient
		}
		p, ok := byBill[b.ID]
		if !ok || p.UserConsumption == nil || anyNegative(p.UserConsumption, p.UserConsumptionF1, p.UserConsumptionF2, p.UserConsumptionF3) {
			continue
		}
		m := measure{actual: *p.UserConsumption, billed: b.ConsumptionTotal}
		if p.ProviderConsumption != nil {
			m.billed = *p.ProviderConsumption
			if utilityType == "gas" {
				m.billed *= coefficient
			}
		}
		if utilityType == "gas" {
			m.actual *= coefficient
		}
		measured[b.ID] = m
	}
	return measured
}

func costWindow(bills []models.Bill, measured map[uint]measure, from, to *time.Time, threshold float64) CostWindow {
	w := CostWindow{From: from, To: to}
	var spans [][2]time.Time
	for _, b := range bills {
		if !inWindow(b.PeriodEnd, from, to) {
			continue
		}
		w.BillCount++
		w.Amount += b.AmountTotal
		spans = append(spans, [2]time.Time{calendarDay(b.PeriodStart), calendarDay(b.PeriodEnd)})
		if m, ok := measured[b.ID]; ok {
			w.MeasuredBillCount++
			w.MeasuredAmount += b.AmountTotal
			w.MeasuredConsumption += m.actual
			w.BilledConsumption += m.billed
		}
	}

	w.Days = coveredDays(spans)
	if w.Days > 0 {
		perDay := w.Amount / float64(w.Days)
		w.CostPerDay = &perDay
	}
	if w.MeasuredConsumption > 0 {
		unit := w.MeasuredAmount / w.MeasuredConsumption
		w.UnitCost = &unit
	}
	if w.MeasuredBillCount > 0 {
		gap := w.BilledConsumption - w.MeasuredConsumption
		w.ConsumptionGap = &gap
		limit := threshold * float64(w.MeasuredBillCount)
		switch {
		case gap > limit*2:
			w.GapLevel = "alert"
		case gap > limit:
			w.GapLevel = "warning"
		case gap < -limit:
			w.GapLevel = "credit"
		}
	}
	return w
}

// readingAnomalies expects bills sorted by period_end ascending. A decrease is
// measured against the last bill that carried that register on the same meter.
func readingAnomalies(utilityType string, bills []models.Bill) []ReadingAnomaly {
	type register struct {
		name  string
		value func(models.Bill) *float64
	}
	registers := []register{{"", func(b models.Bill) *float64 { return b.ProviderReading }}}
	if utilityType == "electricity" {
		registers = []register{
			{"f1", func(b models.Bill) *float64 { return b.ProviderReadingF1 }},
			{"f2", func(b models.Bill) *float64 { return b.ProviderReadingF2 }},
			{"f3", func(b models.Bill) *float64 { return b.ProviderReadingF3 }},
		}
	}

	var anomalies []ReadingAnomaly
	last := make([]*float64, len(registers))
	run, runIdx := 0, -1
	var prev models.Bill
	for i, b := range bills {
		if i > 0 && !sameMeter(prev, b) {
			clear(last)
		}
		prev = b
		if b.ReadingType == "estimated" {
			run++
		} else {
			run = 0
		}
		switch {
		case run == 2:
			runIdx = len(anomalies)
			anomalies = append(anomalies, ReadingAnomaly{Kind: "consecutive_estimates"})
			fallthrough
		case run > 2:
			anomalies[runIdx].BillID = b.ID
			anomalies[runIdx].BillNumber = b.BillNumber
			anomalies[runIdx].PeriodEnd = b.PeriodEnd
			anomalies[runIdx].Count = run
		}

		for i, r := range registers {
			current := r.value(b)
			if current == nil {
				continue
			}
			if last[i] != nil && *current < *last[i] {
				anomalies = append(anomalies, ReadingAnomaly{
					Kind: "reading_decrease", BillID: b.ID, BillNumber: b.BillNumber, PeriodEnd: b.PeriodEnd,
					Register: r.name, Previous: last[i], Current: current,
				})
			}
			last[i] = current
		}
	}
	return anomalies
}

func inWindow(t time.Time, from, to *time.Time) bool {
	day := calendarDay(t)
	return (from == nil || !day.Before(*from)) && (to == nil || !day.After(*to))
}

// calendarDay keeps the date as stored, whatever offset it was saved with.
func calendarDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// coveredDays counts the calendar days inside at least one span (inclusive), so
// overlapping bills are not counted twice and gaps between them not at all.
func coveredDays(spans [][2]time.Time) int {
	sort.Slice(spans, func(i, j int) bool { return spans[i][0].Before(spans[j][0]) })
	days := 0
	var start, end time.Time
	for i, s := range spans {
		if i > 0 && !s[0].After(end.AddDate(0, 0, 1)) {
			if s[1].After(end) {
				end = s[1]
			}
			continue
		}
		if i > 0 {
			days += int(end.Sub(start).Hours()/24) + 1
		}
		start, end = s[0], s[1]
	}
	if len(spans) > 0 {
		days += int(end.Sub(start).Hours()/24) + 1
	}
	return days
}

func relativeChange(current, previous *float64) *float64 {
	if current == nil || previous == nil || *previous <= 0 {
		return nil
	}
	change := *current / *previous - 1
	return &change
}

func anyNegative(values ...*float64) bool {
	for _, v := range values {
		if v != nil && *v < 0 {
			return true
		}
	}
	return false
}
