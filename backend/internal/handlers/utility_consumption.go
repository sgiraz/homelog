package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// ReadingComparison represents the comparison between provider and user readings
type ReadingComparison struct {
	BillID              uint       `json:"bill_id"`
	BillNumber          string     `json:"bill_number"`
	PeriodEnd           time.Time  `json:"period_end"`
	UtilityType         string     `json:"utility_type"`
	ReadingType         string     `json:"reading_type"` // actual or estimated
	ProviderReadingDate *time.Time `json:"provider_reading_date"`
	UserReadingDate     *time.Time `json:"user_reading_date"`
	DaysDifference      int        `json:"days_difference"`     // Days between user and provider readings
	EffectiveThreshold  float64    `json:"effective_threshold"` // Threshold adjusted for days difference
	// For electricity (F1/F2/F3)
	ProviderF1   *float64 `json:"provider_f1,omitempty"`
	ProviderF2   *float64 `json:"provider_f2,omitempty"`
	ProviderF3   *float64 `json:"provider_f3,omitempty"`
	UserF1       *float64 `json:"user_f1,omitempty"`
	UserF2       *float64 `json:"user_f2,omitempty"`
	UserF3       *float64 `json:"user_f3,omitempty"`
	DifferenceF1 *float64 `json:"difference_f1,omitempty"` // Absolute difference in kWh
	DifferenceF2 *float64 `json:"difference_f2,omitempty"`
	DifferenceF3 *float64 `json:"difference_f3,omitempty"`
	// For gas/water (single value)
	ProviderReading *float64 `json:"provider_reading,omitempty"`
	UserReading     *float64 `json:"user_reading,omitempty"`
	Difference      *float64 `json:"difference,omitempty"` // Absolute difference in mc/Smc
	// Status
	Status       string `json:"status"`        // ok, warning, alert, no_data
	AlertMessage string `json:"alert_message"` // Human readable message
}

// ConsumptionPeriod represents consumption for a period comparing user vs provider readings
type ConsumptionPeriod struct {
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	BillID      *uint     `json:"bill_id,omitempty"`
	// User consumption (from self-readings difference)
	UserConsumptionF1 *float64 `json:"user_consumption_f1,omitempty"`
	UserConsumptionF2 *float64 `json:"user_consumption_f2,omitempty"`
	UserConsumptionF3 *float64 `json:"user_consumption_f3,omitempty"`
	UserConsumption   *float64 `json:"user_consumption,omitempty"` // For gas/water
	// Provider consumption (from billed readings difference)
	ProviderConsumptionF1 *float64 `json:"provider_consumption_f1,omitempty"`
	ProviderConsumptionF2 *float64 `json:"provider_consumption_f2,omitempty"`
	ProviderConsumptionF3 *float64 `json:"provider_consumption_f3,omitempty"`
	ProviderConsumption   *float64 `json:"provider_consumption,omitempty"` // For gas/water
	// Differences
	DifferenceF1 *float64 `json:"difference_f1,omitempty"`
	DifferenceF2 *float64 `json:"difference_f2,omitempty"`
	DifferenceF3 *float64 `json:"difference_f3,omitempty"`
	Difference   *float64 `json:"difference,omitempty"`
}

// ConsumptionSummary contains cumulative consumption analysis
type ConsumptionSummary struct {
	// Cumulative totals from user self-readings
	TotalUserF1 float64 `json:"total_user_f1"`
	TotalUserF2 float64 `json:"total_user_f2"`
	TotalUserF3 float64 `json:"total_user_f3"`
	TotalUser   float64 `json:"total_user"` // For gas/water or sum of F1+F2+F3
	// Cumulative totals from provider readings (billed)
	TotalProviderF1 float64 `json:"total_provider_f1"`
	TotalProviderF2 float64 `json:"total_provider_f2"`
	TotalProviderF3 float64 `json:"total_provider_f3"`
	TotalProvider   float64 `json:"total_provider"` // For gas/water or sum of F1+F2+F3
	// Cumulative differences (positive = provider charged more than actual consumption)
	CumulativeDifferenceF1 float64 `json:"cumulative_difference_f1"`
	CumulativeDifferenceF2 float64 `json:"cumulative_difference_f2"`
	CumulativeDifferenceF3 float64 `json:"cumulative_difference_f3"`
	CumulativeDifference   float64 `json:"cumulative_difference"` // Total difference
	// Period covered
	FirstPeriod time.Time `json:"first_period"`
	LastPeriod  time.Time `json:"last_period"`
	// Alert if cumulative difference is significant
	HasCumulativeAlert   bool   `json:"has_cumulative_alert"`
	CumulativeAlertLevel string `json:"cumulative_alert_level,omitempty"` // warning, alert
	CumulativeMessage    string `json:"cumulative_message,omitempty"`
}

// CompareReadings compares provider readings from bills with user's manual readings
// GET /api/v1/utilities/:id/compare-readings
func (h *UtilityHandler) CompareReadings(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	utilityID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid utility ID"})
		return
	}

	// Verify access to utility
	var memberPropertyIDs []uint
	h.db.Model(&models.HouseholdMember{}).
		Where("user_id = ?", userID).
		Pluck("property_id", &memberPropertyIDs)

	var utility models.Utility
	if err := h.db.Where("id = ? AND property_id IN ?", utilityID, memberPropertyIDs).
		First(&utility).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utility not found"})
		return
	}

	// Get all bills with provider readings, preloading the explicit user reading association
	var bills []models.Bill
	h.db.Where("utility_id = ?", utilityID).
		Preload("UserReading").
		Order("period_end DESC").
		Find(&bills)

	log.Printf("CompareReadings: Found %d bills for utility %d (type: %s)", len(bills), utilityID, utility.Type)

	// Get all user readings
	var readings []models.MeterReading
	h.db.Where("utility_id = ?", utilityID).
		Order("reading_date DESC").
		Find(&readings)

	log.Printf("CompareReadings: Found %d user readings", len(readings))

	// Use utility's comparison threshold (base threshold for same-day readings)
	baseThreshold := utility.ComparisonThreshold
	if baseThreshold == 0 {
		baseThreshold = 2.0
	}
	// Per-day tolerance
	thresholdPerDay := utility.ThresholdPerDay
	if thresholdPerDay == 0 {
		thresholdPerDay = 1.0
	}
	// Allow override via query params
	if t := c.Query("threshold"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil {
			baseThreshold = parsed
		}
	}
	if t := c.Query("threshold_per_day"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil {
			thresholdPerDay = parsed
		}
	}

	var comparisons []ReadingComparison

	for _, bill := range bills {
		// Skip bills without provider readings
		hasProviderReading := false
		if utility.Type == "electricity" {
			hasProviderReading = bill.ProviderReadingF1 != nil || bill.ProviderReadingF2 != nil || bill.ProviderReadingF3 != nil
			log.Printf("Bill %d: F1=%v, F2=%v, F3=%v, hasReading=%v",
				bill.ID, bill.ProviderReadingF1, bill.ProviderReadingF2, bill.ProviderReadingF3, hasProviderReading)
		} else {
			hasProviderReading = bill.ProviderReading != nil
			log.Printf("Bill %d: ProviderReading=%v, hasReading=%v",
				bill.ID, bill.ProviderReading, hasProviderReading)
		}

		if !hasProviderReading {
			log.Printf("Skipping bill %d - no provider readings", bill.ID)
			continue
		}

		comparison := ReadingComparison{
			BillID:              bill.ID,
			BillNumber:          bill.BillNumber,
			PeriodEnd:           bill.PeriodEnd,
			UtilityType:         utility.Type,
			ReadingType:         bill.ReadingType,
			ProviderReadingDate: bill.ProviderReadingDate,
			Status:              "ok",
			EffectiveThreshold:  baseThreshold, // Will be adjusted based on days difference
		}

		// Find user reading that falls within the bill period
		// The user's autolettura should be within or close to the billing period
		var closestReading *models.MeterReading
		var bestScore int = -1 // Higher is better

		log.Printf("Bill %d: period %v - %v", bill.ID, bill.PeriodStart.Format("2006-01-02"), bill.PeriodEnd.Format("2006-01-02"))

		for i := range readings {
			readingDate := readings[i].ReadingDate
			score := 0

			// Best match: reading date is within the bill period
			if !readingDate.Before(bill.PeriodStart) && !readingDate.After(bill.PeriodEnd) {
				score = 100
			} else {
				// Also consider readings slightly before period start or after period end
				// (user might read meter a few days early/late)
				daysBefore := bill.PeriodStart.Sub(readingDate).Hours() / 24
				daysAfter := readingDate.Sub(bill.PeriodEnd).Hours() / 24

				if daysBefore > 0 && daysBefore <= 15 {
					score = 50 - int(daysBefore) // Within 15 days before period start
				} else if daysAfter > 0 && daysAfter <= 15 {
					score = 50 - int(daysAfter) // Within 15 days after period end
				}
			}

			log.Printf("  Reading %d: date=%v, score=%d (F1=%v, F2=%v, F3=%v)",
				readings[i].ID, readingDate.Format("2006-01-02"), score,
				readings[i].ValueF1, readings[i].ValueF2, readings[i].ValueF3)

			if score > bestScore {
				bestScore = score
				closestReading = &readings[i]
			}
		}

		if closestReading != nil {
			log.Printf("Best match: Reading %d with score %d", closestReading.ID, bestScore)
		} else {
			log.Printf("No matching reading found for bill %d", bill.ID)
		}

		if closestReading != nil {
			comparison.UserReadingDate = &closestReading.ReadingDate

			// Calculate days difference between user reading and provider reading
			providerDate := bill.PeriodEnd // Default to period end if no specific date
			if bill.ProviderReadingDate != nil {
				providerDate = *bill.ProviderReadingDate
			}
			daysDiff := int(providerDate.Sub(closestReading.ReadingDate).Hours() / 24)
			if daysDiff < 0 {
				daysDiff = -daysDiff
			}
			comparison.DaysDifference = daysDiff

			// Calculate effective threshold: base + (days * perDay)
			comparison.EffectiveThreshold = baseThreshold + float64(daysDiff)*thresholdPerDay
			log.Printf("Days difference: %d, Effective threshold: %.2f (base: %.2f + %d * %.2f)",
				daysDiff, comparison.EffectiveThreshold, baseThreshold, daysDiff, thresholdPerDay)
		}

		// Compare based on utility type
		switch utility.Type {
		case "electricity":
			comparison.ProviderF1 = bill.ProviderReadingF1
			comparison.ProviderF2 = bill.ProviderReadingF2
			comparison.ProviderF3 = bill.ProviderReadingF3

			if closestReading != nil {
				comparison.UserF1 = closestReading.ValueF1
				comparison.UserF2 = closestReading.ValueF2
				comparison.UserF3 = closestReading.ValueF3

				// Calculate absolute differences for each band
				maxAbsDiff := 0.0

				if bill.ProviderReadingF1 != nil && closestReading.ValueF1 != nil {
					diff := *bill.ProviderReadingF1 - *closestReading.ValueF1
					comparison.DifferenceF1 = &diff
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}
					if absDiff > maxAbsDiff {
						maxAbsDiff = absDiff
					}
				}

				if bill.ProviderReadingF2 != nil && closestReading.ValueF2 != nil {
					diff := *bill.ProviderReadingF2 - *closestReading.ValueF2
					comparison.DifferenceF2 = &diff
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}
					if absDiff > maxAbsDiff {
						maxAbsDiff = absDiff
					}
				}

				if bill.ProviderReadingF3 != nil && closestReading.ValueF3 != nil {
					diff := *bill.ProviderReadingF3 - *closestReading.ValueF3
					comparison.DifferenceF3 = &diff
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}
					if absDiff > maxAbsDiff {
						maxAbsDiff = absDiff
					}
				}

				// Determine status based on absolute difference using effective threshold
				effectiveThreshold := comparison.EffectiveThreshold
				if maxAbsDiff > effectiveThreshold*2 {
					comparison.Status = "alert"
					comparison.AlertMessage = fmt.Sprintf("Discrepanza di %.1f kWh (soglia effettiva: %.1f kWh per %d giorni di differenza)",
						maxAbsDiff, effectiveThreshold, comparison.DaysDifference)
				} else if maxAbsDiff > effectiveThreshold {
					comparison.Status = "warning"
					comparison.AlertMessage = fmt.Sprintf("Differenza di %.1f kWh (soglia effettiva: %.1f kWh per %d giorni di differenza)",
						maxAbsDiff, effectiveThreshold, comparison.DaysDifference)
				}
			} else {
				comparison.Status = "no_data"
				comparison.AlertMessage = "Nessuna autolettura disponibile per il confronto"
			}

		case "gas", "water":
			comparison.ProviderReading = bill.ProviderReading

			if closestReading != nil && closestReading.Value != nil {
				comparison.UserReading = closestReading.Value

				if bill.ProviderReading != nil {
					diff := *bill.ProviderReading - *closestReading.Value
					comparison.Difference = &diff

					// Determine status based on absolute difference using effective threshold
					absDiff := diff
					if absDiff < 0 {
						absDiff = -absDiff
					}

					unit := "mc"
					if utility.Type == "gas" {
						unit = "Smc"
					}

					effectiveThreshold := comparison.EffectiveThreshold
					if absDiff > effectiveThreshold*2 {
						comparison.Status = "alert"
						comparison.AlertMessage = fmt.Sprintf("Discrepanza di %.1f %s (soglia effettiva: %.1f %s per %d giorni di differenza)",
							absDiff, unit, effectiveThreshold, unit, comparison.DaysDifference)
					} else if absDiff > effectiveThreshold {
						comparison.Status = "warning"
						comparison.AlertMessage = fmt.Sprintf("Differenza di %.1f %s (soglia effettiva: %.1f %s per %d giorni di differenza)",
							absDiff, unit, effectiveThreshold, unit, comparison.DaysDifference)
					}
				}
			} else {
				comparison.Status = "no_data"
				comparison.AlertMessage = "Nessuna autolettura disponibile per il confronto"
			}
		}

		comparisons = append(comparisons, comparison)
	}

	// Calculate consumption analysis (differences between consecutive readings)
	consumptionPeriods, consumptionSummary := h.calculateConsumptionAnalysis(utility.Type, bills, baseThreshold)

	c.JSON(http.StatusOK, gin.H{
		"comparisons":         comparisons,
		"base_threshold":      baseThreshold,
		"threshold_per_day":   thresholdPerDay,
		"utility_type":        utility.Type,
		"consumption_periods": consumptionPeriods,
		"consumption_summary": consumptionSummary,
	})
}

// calculateConsumptionAnalysis computes period-by-period consumption comparison.
//
// Algorithm (deterministic):
//   - Sort bills by period_end ascending.
//   - For each bill define its “effective user reading value”:
//     • If bill.UserReading != nil → use the explicitly associated self-reading value.
//     • Otherwise              → fall back to bill.ProviderReading (so Δeffettivo = Δfatturato).
//   - Consumo Fatturato[i]  = providerReading[i]  − providerReading[i-1]  (consecutive absolute bill readings)
//   - Consumo Effettivo[i]  = effectiveReading[i] − effectiveReading[i-1] (consecutive effective readings)
//   - Period rows start from index 1 (bill[0] is the reference anchor; no row generated for it).
//   - Guard: need at least 2 bills with provider_reading set.
func (h *UtilityHandler) calculateConsumptionAnalysis(utilityType string, bills []models.Bill, threshold float64) ([]ConsumptionPeriod, *ConsumptionSummary) {
	if len(bills) < 2 {
		return nil, nil
	}

	// Sort bills by period_end ascending
	sortedBills := make([]models.Bill, len(bills))
	copy(sortedBills, bills)
	for i := 0; i < len(sortedBills)-1; i++ {
		for j := i + 1; j < len(sortedBills); j++ {
			if sortedBills[i].PeriodEnd.After(sortedBills[j].PeriodEnd) {
				sortedBills[i], sortedBills[j] = sortedBills[j], sortedBills[i]
			}
		}
	}

	// effectiveValue returns the user reading value for a bill.
	// Uses the explicitly associated UserReading when available;
	// falls back to ProviderReading so the period contribution is neutral (Δ=0).
	effectiveValue := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.Value != nil {
			return b.UserReading.Value
		}
		return b.ProviderReading
	}
	effectiveValueF1 := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.ValueF1 != nil {
			return b.UserReading.ValueF1
		}
		return b.ProviderReadingF1
	}
	effectiveValueF2 := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.ValueF2 != nil {
			return b.UserReading.ValueF2
		}
		return b.ProviderReadingF2
	}
	effectiveValueF3 := func(b models.Bill) *float64 {
		if b.UserReading != nil && b.UserReading.ValueF3 != nil {
			return b.UserReading.ValueF3
		}
		return b.ProviderReadingF3
	}

	var periods []ConsumptionPeriod
	summary := &ConsumptionSummary{}

	for i := 1; i < len(sortedBills); i++ {
		bill := sortedBills[i]
		prev := sortedBills[i-1]
		billID := bill.ID
		period := ConsumptionPeriod{
			PeriodStart: bill.PeriodStart,
			PeriodEnd:   bill.PeriodEnd,
			BillID:      &billID,
		}
		if utilityType == "electricity" {
			// --- Provider consumption (consecutive absolute F1/F2/F3 readings) ---
			provTotal := 0.0
			hasProvF1 := bill.ProviderReadingF1 != nil && prev.ProviderReadingF1 != nil
			hasProvF2 := bill.ProviderReadingF2 != nil && prev.ProviderReadingF2 != nil
			hasProvF3 := bill.ProviderReadingF3 != nil && prev.ProviderReadingF3 != nil
			if hasProvF1 {
				f1 := *bill.ProviderReadingF1 - *prev.ProviderReadingF1
				period.ProviderConsumptionF1 = &f1
				summary.TotalProviderF1 += f1
				provTotal += f1
			}
			if hasProvF2 {
				f2 := *bill.ProviderReadingF2 - *prev.ProviderReadingF2
				period.ProviderConsumptionF2 = &f2
				summary.TotalProviderF2 += f2
				provTotal += f2
			}
			if hasProvF3 {
				f3 := *bill.ProviderReadingF3 - *prev.ProviderReadingF3
				period.ProviderConsumptionF3 = &f3
				summary.TotalProviderF3 += f3
				provTotal += f3
			}
			if hasProvF1 || hasProvF2 || hasProvF3 {
				period.ProviderConsumption = &provTotal
				summary.TotalProvider += provTotal
			}

			// --- User consumption (consecutive effective F1/F2/F3 values) ---
			ef1Cur, ef1Prv := effectiveValueF1(bill), effectiveValueF1(prev)
			ef2Cur, ef2Prv := effectiveValueF2(bill), effectiveValueF2(prev)
			ef3Cur, ef3Prv := effectiveValueF3(bill), effectiveValueF3(prev)
			userTotal := 0.0
			hasUser := false
			if ef1Cur != nil && ef1Prv != nil {
				f1 := *ef1Cur - *ef1Prv
				period.UserConsumptionF1 = &f1
				summary.TotalUserF1 += f1
				userTotal += f1
				hasUser = true
			}
			if ef2Cur != nil && ef2Prv != nil {
				f2 := *ef2Cur - *ef2Prv
				period.UserConsumptionF2 = &f2
				summary.TotalUserF2 += f2
				userTotal += f2
				hasUser = true
			}
			if ef3Cur != nil && ef3Prv != nil {
				f3 := *ef3Cur - *ef3Prv
				period.UserConsumptionF3 = &f3
				summary.TotalUserF3 += f3
				userTotal += f3
				hasUser = true
			}
			if hasUser {
				period.UserConsumption = &userTotal
				summary.TotalUser += userTotal
			}

			// Per-band differences
			if period.ProviderConsumptionF1 != nil && period.UserConsumptionF1 != nil {
				d := *period.ProviderConsumptionF1 - *period.UserConsumptionF1
				period.DifferenceF1 = &d
			}
			if period.ProviderConsumptionF2 != nil && period.UserConsumptionF2 != nil {
				d := *period.ProviderConsumptionF2 - *period.UserConsumptionF2
				period.DifferenceF2 = &d
			}
			if period.ProviderConsumptionF3 != nil && period.UserConsumptionF3 != nil {
				d := *period.ProviderConsumptionF3 - *period.UserConsumptionF3
				period.DifferenceF3 = &d
			}

		} else {
			// --- Gas / Water ---
			// Provider: consecutive absolute ProviderReading
			if bill.ProviderReading != nil && prev.ProviderReading != nil {
				provTotal := *bill.ProviderReading - *prev.ProviderReading
				period.ProviderConsumption = &provTotal
				summary.TotalProvider += provTotal
			}

			// User: consecutive effective reading values
			efCur, efPrv := effectiveValue(bill), effectiveValue(prev)
			if efCur != nil && efPrv != nil {
				userTotal := *efCur - *efPrv
				period.UserConsumption = &userTotal
				summary.TotalUser += userTotal
			}
		}

		// Overall difference for this period
		if period.ProviderConsumption != nil && period.UserConsumption != nil {
			diff := *period.ProviderConsumption - *period.UserConsumption
			period.Difference = &diff
		}

		periods = append(periods, period)
	}
	summary.CumulativeDifferenceF1 = summary.TotalProviderF1 - summary.TotalUserF1
	summary.CumulativeDifferenceF2 = summary.TotalProviderF2 - summary.TotalUserF2
	summary.CumulativeDifferenceF3 = summary.TotalProviderF3 - summary.TotalUserF3
	summary.CumulativeDifference = summary.TotalProvider - summary.TotalUser

	if len(sortedBills) > 0 {
		summary.FirstPeriod = sortedBills[0].PeriodStart
		summary.LastPeriod = sortedBills[len(sortedBills)-1].PeriodEnd
	}

	numPeriods := len(periods)
	if numPeriods > 0 {
		cumulativeThreshold := threshold * float64(numPeriods)
		if summary.CumulativeDifference > cumulativeThreshold*2 {
			summary.HasCumulativeAlert = true
			summary.CumulativeAlertLevel = "alert"
			summary.CumulativeMessage = fmt.Sprintf("ATTENZIONE: il fornitore ha fatturato %.1f unità IN PIÙ rispetto ai consumi effettivi rilevati dalle autoletture in %d periodi. Stai pagando più del dovuto!",
				summary.CumulativeDifference, numPeriods)
		} else if summary.CumulativeDifference > cumulativeThreshold {
			summary.HasCumulativeAlert = true
			summary.CumulativeAlertLevel = "warning"
			summary.CumulativeMessage = fmt.Sprintf("Il fornitore ha fatturato %.1f unità in più rispetto ai consumi effettivi in %d periodi. Tieni sotto controllo questa differenza.",
				summary.CumulativeDifference, numPeriods)
		} else if summary.CumulativeDifference < -cumulativeThreshold {
			summary.CumulativeMessage = fmt.Sprintf("Il fornitore ha fatturato %.1f unità in meno rispetto ai consumi rilevati in %d periodi. Potrebbe esserci un conguaglio a fine anno.",
				-summary.CumulativeDifference, numPeriods)
		}
	}

	return periods, summary
}
