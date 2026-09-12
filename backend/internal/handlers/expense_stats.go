package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// TrendPoint is a single bucket of the trend chart. Date is the first day of
// the bucket in ISO form; the client formats it for the user's locale together
// with the response's granularity. The server ships no localized label — it
// used to emit Italian month names, which stayed Italian in every language.
type TrendPoint struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// CategoryStats represents expense statistics by category
type CategoryStats struct {
	CategoryID uint `json:"category_id"`
	// CategorySlug is set for built-in categories/subcategories; the client
	// renders t("categories.<slug>") and ignores CategoryName. Empty for
	// user-created ones, whose CategoryName is the label to show as-is.
	CategorySlug string  `json:"category_slug,omitempty"`
	CategoryName string  `json:"category_name"`
	Amount       float64 `json:"amount"`
	Count        int     `json:"count"`
	Percentage   float64 `json:"percentage"`
}

// GetStats returns statistics for charts
// GET /api/v1/expenses/stats?period=6m&property_id=1&year=2024
func (h *ExpenseHandler) GetStats(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	// Find property IDs where user is a member
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	// Determine date range
	now := time.Now()
	year := now.Year()
	if y := c.Query("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	// Default period is current year
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	// Handle period parameter (e.g., "6m" for last 6 months)
	if period := c.Query("period"); period != "" {
		switch period {
		case "1m":
			startDate = now.AddDate(0, -1, 0)
			endDate = now
		case "3m":
			startDate = now.AddDate(0, -3, 0)
			endDate = now
		case "6m":
			startDate = now.AddDate(0, -6, 0)
			endDate = now
		case "12m":
			startDate = now.AddDate(-1, 0, 0)
			endDate = now
		}
	}

	// Explicit from/to override year and period
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			startDate = t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			endDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
		}
	}

	// "all=true" overrides everything: use first expense date → today
	if c.Query("all") == "true" {
		var minDateStr string
		h.db.Model(&models.Expense{}).
			Where("property_id IN ?", memberPropertyIDs).
			Select("substr(MIN(date), 1, 10)").
			Scan(&minDateStr)
		if minDateStr != "" {
			if t, err := time.Parse("2006-01-02", minDateStr); err == nil {
				startDate = t
			}
		}
		endDate = now
	}

	// Build reusable WHERE conditions shared across all sub-queries. Compare
	// against the literal calendar-date prefix of the stored string (substr),
	// not SQLite's date()/strftime() functions — those convert through UTC
	// using any offset embedded in the value first, which shifts a value
	// recorded just after local midnight onto the previous UTC day. See the
	// same fix in List/count above for the full explanation.
	baseWhere := "property_id IN ? AND substr(date, 1, 10) >= ? AND substr(date, 1, 10) <= ?"
	joinWhere := "expenses.property_id IN ? AND substr(expenses.date, 1, 10) >= ? AND substr(expenses.date, 1, 10) <= ?"
	baseArgs := []any{memberPropertyIDs, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")}

	categoryID := c.Query("category_id")
	if categoryID != "" {
		baseWhere += " AND category_id = ?"
		joinWhere += " AND expenses.category_id = ?"
		baseArgs = append(baseArgs, categoryID)
	}
	if propertyID := c.Query("property_id"); propertyID != "" {
		baseWhere += " AND property_id = ?"
		joinWhere += " AND expenses.property_id = ?"
		baseArgs = append(baseArgs, propertyID)
	}

	// Determine trend granularity from date range
	dayRange := int(endDate.Sub(startDate).Hours() / 24)
	granularity := "month"
	if dayRange <= 31 {
		granularity = "day"
	} else if dayRange > 365 {
		granularity = "quarter"
	}

	var trend []TrendPoint

	switch granularity {
	case "day":
		var rows []struct {
			Day    int     `json:"day"`
			Month  int     `json:"month"`
			Year   int     `json:"year"`
			Amount float64 `json:"amount"`
			Count  int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST(substr(date, 9, 2) AS INTEGER) as day, CAST(substr(date, 6, 2) AS INTEGER) as month, CAST(substr(date, 1, 4) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("substr(date, 1, 10)").
			Order("year, month, day").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			trend[i] = TrendPoint{
				Date:   fmt.Sprintf("%04d-%02d-%02d", r.Year, r.Month, r.Day),
				Amount: r.Amount,
				Count:  r.Count,
			}
		}

	case "quarter":
		var rows []struct {
			Quarter int     `json:"quarter"`
			Year    int     `json:"year"`
			Amount  float64 `json:"amount"`
			Count   int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST((CAST(substr(date, 6, 2) AS INTEGER) + 2) / 3 AS INTEGER) as quarter, CAST(substr(date, 1, 4) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("year, quarter").
			Order("year, quarter").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			// First day of the quarter: Q1 -> January, Q2 -> April, ...
			trend[i] = TrendPoint{
				Date:   fmt.Sprintf("%04d-%02d-01", r.Year, (r.Quarter-1)*3+1),
				Amount: r.Amount,
				Count:  r.Count,
			}
		}

	default: // month
		var rows []struct {
			Month  int     `json:"month"`
			Year   int     `json:"year"`
			Amount float64 `json:"amount"`
			Count  int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST(substr(date, 6, 2) AS INTEGER) as month, CAST(substr(date, 1, 4) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("substr(date, 1, 7)").
			Order("year, month").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			trend[i] = TrendPoint{
				Date:   fmt.Sprintf("%04d-%02d-01", r.Year, r.Month),
				Amount: r.Amount,
				Count:  r.Count,
			}
		}
	}

	// Category/Subcategory aggregation
	var totalAmount float64
	var byCategory []CategoryStats

	if categoryID != "" {
		// Subcategory breakdown when filtering by category
		var subResults []struct {
			SubcategoryID *uint   `json:"subcategory_id"`
			CategorySlug  string  `json:"category_slug"`
			CategoryName  string  `json:"category_name"`
			Amount        float64 `json:"amount"`
			Count         int     `json:"count"`
		}
		// The "no subcategory" bucket comes back with an empty name and a zero
		// id; the client labels it. Never hardcode a localized string here.
		h.db.Model(&models.Expense{}).
			Select("expenses.subcategory_id, COALESCE(subcategories.slug, '') as category_slug, COALESCE(subcategories.name, '') as category_name, SUM(expenses.amount) as amount, COUNT(*) as count").
			Joins("LEFT JOIN subcategories ON subcategories.id = expenses.subcategory_id").
			Where(joinWhere, baseArgs...).
			Group("expenses.subcategory_id").
			Order("amount DESC").
			Scan(&subResults)

		for _, r := range subResults {
			totalAmount += r.Amount
		}
		byCategory = make([]CategoryStats, len(subResults))
		for i, r := range subResults {
			percentage := 0.0
			if totalAmount > 0 {
				percentage = (r.Amount / totalAmount) * 100
			}
			id := uint(0)
			if r.SubcategoryID != nil {
				id = *r.SubcategoryID
			}
			byCategory[i] = CategoryStats{
				CategoryID:   id,
				CategorySlug: r.CategorySlug,
				CategoryName: r.CategoryName,
				Amount:       r.Amount,
				Count:        r.Count,
				Percentage:   percentage,
			}
		}
	} else {
		var categoryResults []struct {
			CategoryID   uint    `json:"category_id"`
			CategorySlug string  `json:"category_slug"`
			CategoryName string  `json:"category_name"`
			Amount       float64 `json:"amount"`
			Count        int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("expenses.category_id, COALESCE(categories.slug, '') as category_slug, categories.name as category_name, SUM(expenses.amount) as amount, COUNT(*) as count").
			Joins("JOIN categories ON categories.id = expenses.category_id").
			Where(joinWhere, baseArgs...).
			Group("expenses.category_id").
			Order("amount DESC").
			Scan(&categoryResults)

		for _, r := range categoryResults {
			totalAmount += r.Amount
		}
		byCategory = make([]CategoryStats, len(categoryResults))
		for i, r := range categoryResults {
			percentage := 0.0
			if totalAmount > 0 {
				percentage = (r.Amount / totalAmount) * 100
			}
			byCategory[i] = CategoryStats{
				CategoryID:   r.CategoryID,
				CategorySlug: r.CategorySlug,
				CategoryName: r.CategoryName,
				Amount:       r.Amount,
				Count:        r.Count,
				Percentage:   percentage,
			}
		}
	}

	// Current month total (respects category filter)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	monthWhere := "property_id IN ? AND substr(date, 1, 10) >= ? AND substr(date, 1, 10) <= ?"
	monthArgs := []any{memberPropertyIDs, monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02")}
	if categoryID != "" {
		monthWhere += " AND category_id = ?"
		monthArgs = append(monthArgs, categoryID)
	}
	var totalMonth float64
	h.db.Model(&models.Expense{}).
		Where(monthWhere, monthArgs...).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalMonth)

	// Year total (respects category filter)
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.UTC)
	yearWhere := "property_id IN ? AND substr(date, 1, 10) >= ? AND substr(date, 1, 10) <= ?"
	yearArgs := []any{memberPropertyIDs, yearStart.Format("2006-01-02"), yearEnd.Format("2006-01-02")}
	if categoryID != "" {
		yearWhere += " AND category_id = ?"
		yearArgs = append(yearArgs, categoryID)
	}
	var totalYear float64
	h.db.Model(&models.Expense{}).
		Where(yearWhere, yearArgs...).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalYear)

	// Average per period point (day/month/quarter)
	var avgMonth float64
	if len(trend) > 0 {
		avgMonth = totalAmount / float64(len(trend))
	}

	// How many expenses the selected period holds. The client used to count the
	// rows it had paginated, which is a different number as soon as the list is
	// longer than one page.
	var periodCount int64
	h.db.Model(&models.Expense{}).
		Where(baseWhere, baseArgs...).
		Count(&periodCount)

	// Same-length window immediately before the selected one, so the client can
	// say "-12% vs the previous 30 days". A total on its own is not actionable:
	// it takes a reference to become a signal.
	previousEnd := startDate.Add(-time.Second)
	previousStart := previousEnd.Add(-endDate.Sub(startDate))
	previousArgs := append([]any{memberPropertyIDs, previousStart.Format("2006-01-02"), previousEnd.Format("2006-01-02")}, baseArgs[3:]...)
	var totalPrevious float64
	h.db.Model(&models.Expense{}).
		Where(baseWhere, previousArgs...).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalPrevious)

	c.JSON(http.StatusOK, gin.H{
		"trend":          trend,
		"granularity":    granularity,
		"by_category":    byCategory,
		"is_subcategory": categoryID != "",
		"total_month":    totalMonth,
		"total_year":     totalYear,
		"total_period":   totalAmount,
		"average_month":  avgMonth,
		"count":          periodCount,
		"period": gin.H{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
		// Every sum here is the full expense amount, never the signed-in
		// member's split share: the dashboard reports what the household spent.
		"scope": "household",
		"previous": gin.H{
			"total": totalPrevious,
			"start": previousStart.Format("2006-01-02"),
			"end":   previousEnd.Format("2006-01-02"),
		},
	})
}
