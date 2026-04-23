package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// TrendPoint represents a single data point in the trend chart (day/month/quarter)
type TrendPoint struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// CategoryStats represents expense statistics by category
type CategoryStats struct {
	CategoryID   uint    `json:"category_id"`
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
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
			Select("strftime('%Y-%m-%d', MIN(date))").
			Scan(&minDateStr)
		if minDateStr != "" {
			if t, err := time.Parse("2006-01-02", minDateStr); err == nil {
				startDate = t
			}
		}
		endDate = now
	}

	// Build reusable WHERE conditions shared across all sub-queries
	baseWhere := "property_id IN ? AND date >= ? AND date <= ?"
	joinWhere := "expenses.property_id IN ? AND expenses.date >= ? AND expenses.date <= ?"
	baseArgs := []any{memberPropertyIDs, startDate, endDate}

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

	itMonths := []string{"", "Gen", "Feb", "Mar", "Apr", "Mag", "Giu", "Lug", "Ago", "Set", "Ott", "Nov", "Dic"}
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
			Select("CAST(strftime('%d', date) AS INTEGER) as day, CAST(strftime('%m', date) AS INTEGER) as month, CAST(strftime('%Y', date) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("strftime('%Y-%m-%d', date)").
			Order("year, month, day").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			mon := ""
			if r.Month >= 1 && r.Month <= 12 {
				mon = itMonths[r.Month]
			}
			trend[i] = TrendPoint{Label: fmt.Sprintf("%d %s", r.Day, mon), Amount: r.Amount, Count: r.Count}
		}

	case "quarter":
		var rows []struct {
			Quarter int     `json:"quarter"`
			Year    int     `json:"year"`
			Amount  float64 `json:"amount"`
			Count   int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST((CAST(strftime('%m', date) AS INTEGER) + 2) / 3 AS INTEGER) as quarter, CAST(strftime('%Y', date) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("year, quarter").
			Order("year, quarter").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			trend[i] = TrendPoint{Label: fmt.Sprintf("Q%d %d", r.Quarter, r.Year), Amount: r.Amount, Count: r.Count}
		}

	default: // month
		var rows []struct {
			Month  int     `json:"month"`
			Year   int     `json:"year"`
			Amount float64 `json:"amount"`
			Count  int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("CAST(strftime('%m', date) AS INTEGER) as month, CAST(strftime('%Y', date) AS INTEGER) as year, SUM(amount) as amount, COUNT(*) as count").
			Where(baseWhere, baseArgs...).
			Group("strftime('%Y-%m', date)").
			Order("year, month").
			Scan(&rows)
		trend = make([]TrendPoint, len(rows))
		for i, r := range rows {
			mon := ""
			if r.Month >= 1 && r.Month <= 12 {
				mon = itMonths[r.Month]
			}
			trend[i] = TrendPoint{Label: fmt.Sprintf("%s %d", mon, r.Year), Amount: r.Amount, Count: r.Count}
		}
	}

	// Category/Subcategory aggregation
	var totalAmount float64
	var byCategory []CategoryStats

	if categoryID != "" {
		// Subcategory breakdown when filtering by category
		var subResults []struct {
			SubcategoryID *uint   `json:"subcategory_id"`
			CategoryName  string  `json:"category_name"`
			Amount        float64 `json:"amount"`
			Count         int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("expenses.subcategory_id, COALESCE(subcategories.name, 'Senza sottocategoria') as category_name, SUM(expenses.amount) as amount, COUNT(*) as count").
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
				CategoryName: r.CategoryName,
				Amount:       r.Amount,
				Count:        r.Count,
				Percentage:   percentage,
			}
		}
	} else {
		var categoryResults []struct {
			CategoryID   uint    `json:"category_id"`
			CategoryName string  `json:"category_name"`
			Amount       float64 `json:"amount"`
			Count        int     `json:"count"`
		}
		h.db.Model(&models.Expense{}).
			Select("expenses.category_id, categories.name as category_name, SUM(expenses.amount) as amount, COUNT(*) as count").
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
	monthWhere := "property_id IN ? AND date >= ? AND date <= ?"
	monthArgs := []any{memberPropertyIDs, monthStart, monthEnd}
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
	yearWhere := "property_id IN ? AND date >= ? AND date <= ?"
	yearArgs := []any{memberPropertyIDs, yearStart, yearEnd}
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

	c.JSON(http.StatusOK, gin.H{
		"trend":          trend,
		"granularity":    granularity,
		"by_category":    byCategory,
		"is_subcategory": categoryID != "",
		"total_month":    totalMonth,
		"total_year":     totalYear,
		"total_period":   totalAmount,
		"average_month":  avgMonth,
		"period": gin.H{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
	})
}
