package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// extractBillDataDefault runs the fallback regex-based extraction used when no
// template matches. It recognises common Italian utility-bill patterns.
func (h *PDFHandler) extractBillDataDefault(text string, utilityType string, pdfURL string) ExtractedBillData {
	extracted := ExtractedBillData{
		PDFURL:  pdfURL,
		RawText: text,
	}

	// Bill number patterns (common Italian utility bill formats)
	billPatterns := []string{
		`[Bb]olletta\s+n[^\d\s]?\s*(\d{6,})`,
		`[Nn][^\d\s]?\s*[Bb]olletta\s+(\d{6,})`,
		`[Nn][^\d\s]?\s+(\d{6,})\s+emessa`,
		`[Ff]attura\s+n[^\d\s]?\s*(\d{6,})`,
		`[Nn]umero\s+[Ff]attura[:\s]*(\d{6,})`,
		`[Nn][^\d\s]?\s*(\d{6,})`,
	}
	for _, pattern := range billPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			extracted.BillNumber = match[1]
			break
		}
	}

	// Amount patterns (common Italian bill formats, ordered by specificity)
	// NOTE: Only use patterns with clear context labels. Avoid bare number+€ patterns
	// that match any euro amount in the document — templates handle provider-specific formats.
	amountPatterns := []string{
		`[Ii]mporto\s+totale\s+da\s+pagare[^0-9]*(\d+[.,]\d{2})`,
		`[Ii]mporto\s+totale\s+da\s+(\d+[.,]\d{2})`,
		`[Tt]otale\s+[Ss]pesa\s+(\d+[.,]\d{2})`,
		`[Tt]otale\s+da\s+pagare[^0-9]*(\d+[.,]\d{2})`,
		`[Tt]otale\s+[Bb]olletta\s+(\d+[.,]\d{2})`,
	}
	for _, pattern := range amountPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			extracted.AmountTotal = h.parseAmount(match[1])
			break
		}
	}

	// Consumption patterns (multiple patterns per type, ordered by specificity)
	var consumptionPatterns []string
	switch utilityType {
	case "electricity":
		consumptionPatterns = []string{
			`[Cc]onsumo\s+totale[^0-9]*(\d+[.,]?\d*)\s*kWh`,
		}
	case "gas":
		consumptionPatterns = []string{
			`[Cc]onsumi\s+fatturati[:\s]*(\d+[.,]?\d*)\s*Smc`,
			`(\d+[.,]\d+)\s+Smc.*\n\s*[Cc]onsumo\s+totale\s+fatturato`,
			`[Cc]onsumo\s+totale[^0-9]*(\d+[.,]?\d*)\s*Smc`,
		}
	case "water":
		consumptionPatterns = []string{
			`[Cc]onsumo\s+totale[^0-9]*(\d+[.,]?\d*)\s*mc`,
		}
	}
	for _, pattern := range consumptionPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			extracted.ConsumptionTotal = h.parseAmount(match[1])
			break
		}
	}

	// Issue date patterns (Italian month names or numeric)
	issueDatePatterns := []string{
		`emessa\s+il\s+(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`,
		`[Bb]olletta\s+del\s+(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`,
		`[Dd]ata\s+emissione[:\s]*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
	}
	for _, pattern := range issueDatePatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			if len(match) == 4 {
				extracted.IssueDate = h.parseItalianDate(match[1], match[2], match[3])
			} else {
				extracted.IssueDate = h.parseDate(match[1], "")
			}
			log.Printf("Extracted issue date: %s", extracted.IssueDate)
			break
		}
	}

	// Due date patterns - try specific patterns first
	dueDatePatterns := []string{
		`[Ss]cadenza[^0-9]*(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`,
		`[Ss]cadenza[:\s]*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
		`[Dd]ata\s+di\s+scadenza[:\s]*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
		`entro\s+il\s+(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
	}
	for _, pattern := range dueDatePatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			if len(match) == 4 {
				extracted.DueDate = h.parseItalianDate(match[1], strings.ToLower(match[2]), match[3])
				log.Printf("Extracted due date (Italian): %s", extracted.DueDate)
			} else {
				extracted.DueDate = h.parseDate(match[1], "")
				log.Printf("Extracted due date (numeric): %s", extracted.DueDate)
			}
			break
		}
	}

	// Period patterns - try Italian month formats first (more specific)
	monthPeriodPattern := regexp.MustCompile(`(?i)(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})\s*[-–]\s*(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
	if match := monthPeriodPattern.FindStringSubmatch(text); len(match) > 6 {
		extracted.PeriodStart = h.parseItalianDate(match[1], strings.ToLower(match[2]), match[3])
		extracted.PeriodEnd = h.parseItalianDate(match[4], strings.ToLower(match[5]), match[6])
		log.Printf("Extracted period (Italian dash): %s - %s", extracted.PeriodStart, extracted.PeriodEnd)
	}

	if extracted.PeriodStart == "" {
		dallAlPattern := regexp.MustCompile(`(?i)dall['']?\s*(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})\s+al\s+(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
		if match := dallAlPattern.FindStringSubmatch(text); len(match) > 6 {
			extracted.PeriodStart = h.parseItalianDate(match[1], strings.ToLower(match[2]), match[3])
			extracted.PeriodEnd = h.parseItalianDate(match[4], strings.ToLower(match[5]), match[6])
			log.Printf("Extracted period (dall/al): %s - %s", extracted.PeriodStart, extracted.PeriodEnd)
		}
	}

	if extracted.PeriodStart == "" {
		periodPattern := regexp.MustCompile(`(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})\s*[-–]\s*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`)
		if match := periodPattern.FindStringSubmatch(text); len(match) > 2 {
			extracted.PeriodStart = h.parseDate(match[1], "")
			extracted.PeriodEnd = h.parseDate(match[2], "")
			log.Printf("Extracted period (numeric): %s - %s", extracted.PeriodStart, extracted.PeriodEnd)
		}
	}

	// Gas conversion coefficient (Coefficiente di conversione C)
	if utilityType == "gas" {
		coeffPatterns := []string{
			`[Cc]oefficiente\s+di\s+conversione\s*\(?C?\)?\s*[:\s]*(\d+[,\.]\d+)`,
			`[Cc]oeff\.?\s*(?:di\s+)?conv\.?\s*\(?C?\)?\s*[:\s]*(\d+[,\.]\d+)`,
		}
		for _, pattern := range coeffPatterns {
			re := regexp.MustCompile(pattern)
			if match := re.FindStringSubmatch(text); len(match) > 1 {
				coeff := h.parseAmount(match[1])
				extracted.ConversionCoefficient = &coeff
				log.Printf("Extracted gas conversion coefficient: %f", coeff)
				break
			}
		}
	}

	// POD/PDR code
	if utilityType == "electricity" {
		podPattern := regexp.MustCompile(`[Cc]odice\s+POD[:\s]*(IT\d{3}E\d+)`)
		if match := podPattern.FindStringSubmatch(text); len(match) > 1 {
			extracted.ServiceCode = match[1]
		}
	} else if utilityType == "gas" {
		pdrPatterns := []string{
			`[Cc]odice\s+PDR[\s:]*(\d{10,})`,
			`PDR[^0-9]*(\d{10,})`,
		}
		for _, pattern := range pdrPatterns {
			re := regexp.MustCompile(pattern)
			if match := re.FindStringSubmatch(text); len(match) > 1 {
				extracted.ServiceCode = match[1]
				break
			}
		}
	}

	// Customer code
	customerPattern := regexp.MustCompile(`[Cc]odice\s+[Cc]liente[:\s]*([A-Z0-9]+)`)
	if match := customerPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.CustomerCode = match[1]
	}

	h.extractProviderReadings(&extracted, text, utilityType)

	return extracted
}

// extractProviderReadings extracts meter readings from the bill.
// Electricity: F1/F2/F3 values (kWh). Gas/water: single meter reading (mc/Smc).
func (h *PDFHandler) extractProviderReadings(extracted *ExtractedBillData, text string, utilityType string) {
	if strings.Contains(strings.ToLower(text), "effettiva") || strings.Contains(strings.ToLower(text), "rilevata") {
		extracted.ReadingType = "actual"
	} else if strings.Contains(strings.ToLower(text), "stimata") || strings.Contains(strings.ToLower(text), "stima") {
		extracted.ReadingType = "estimated"
	}

	readingPeriodPattern := regexp.MustCompile(`Da\s+(\d{1,2}/\d{1,2}/\d{4})\s*[^\d]*a\s+(\d{1,2}/\d{1,2}/\d{4})`)
	if match := readingPeriodPattern.FindStringSubmatch(text); len(match) > 2 {
		extracted.ProviderReadingDate = h.parseDate(match[2], "")
		log.Printf("Extracted reading period: %s - %s", match[1], match[2])
	}

	switch utilityType {
	case "electricity":
		log.Printf("Extracting electricity readings from text (length: %d)", len(text))

		lettureSection := text
		if idx := strings.Index(strings.ToLower(text), "letture e consumi"); idx != -1 {
			lettureSection = text[idx:]
			if len(lettureSection) > 2000 {
				lettureSection = lettureSection[:2000]
			}
			log.Printf("Found 'Letture e consumi' section at index %d", idx)
		}

		// Pattern for "final - initial = consumption" format
		electricPattern := regexp.MustCompile(`(\d{1,3}(?:[.\s]\d{3})*,\d+)\s*-\s*(\d{1,3}(?:[.\s]\d{3})*,\d+)\s*=\s*(\d{1,3}(?:[.\s]\d{3})*,\d+)`)
		matches := electricPattern.FindAllStringSubmatch(lettureSection, -1)

		log.Printf("Found %d reading matches (pattern 1) in electricity bill", len(matches))

		if len(matches) >= 3 {
			f1 := h.parseAmount(matches[0][1])
			f2 := h.parseAmount(matches[1][1])
			f3 := h.parseAmount(matches[2][1])
			extracted.ProviderReadingF1 = &f1
			extracted.ProviderReadingF2 = &f2
			extracted.ProviderReadingF3 = &f3
			log.Printf("Extracted electricity readings (pattern 1): F1=%.2f, F2=%.2f, F3=%.2f", f1, f2, f3)
		} else {
			simplePattern := regexp.MustCompile(`(\d+[,\.]\d+)\s*-\s*(\d+[,\.]\d+)\s*=\s*(\d+[,\.]\d+)`)
			matches = simplePattern.FindAllStringSubmatch(lettureSection, -1)
			log.Printf("Found %d reading matches (pattern 2 - simple) in electricity bill", len(matches))

			if len(matches) >= 3 {
				f1 := h.parseAmount(matches[0][1])
				f2 := h.parseAmount(matches[1][1])
				f3 := h.parseAmount(matches[2][1])
				extracted.ProviderReadingF1 = &f1
				extracted.ProviderReadingF2 = &f2
				extracted.ProviderReadingF3 = &f3
				log.Printf("Extracted electricity readings (pattern 2): F1=%.2f, F2=%.2f, F3=%.2f", f1, f2, f3)
			}
		}

		if extracted.ProviderReadingF1 == nil {
			log.Printf("Trying FASCIA label patterns...")
			fasciaPattern := regexp.MustCompile(`(?i)FASCIA\s*[123].*?(\d{1,3}(?:[.\s]\d{3})*,\d+)`)
			fasciaMatches := fasciaPattern.FindAllStringSubmatch(lettureSection, 3)
			log.Printf("Found %d FASCIA matches", len(fasciaMatches))
			for i, m := range fasciaMatches {
				log.Printf("FASCIA match %d: %v", i, m)
			}
		}

		if extracted.ProviderReadingF1 != nil {
			log.Printf("Final electricity readings: F1=%.2f, F2=%.2f, F3=%.2f",
				*extracted.ProviderReadingF1, *extracted.ProviderReadingF2, *extracted.ProviderReadingF3)
		} else {
			log.Printf("Warning: Could not extract electricity readings")
			sample := lettureSection
			if len(sample) > 500 {
				sample = sample[:500]
			}
			log.Printf("Sample text: %s", sample)
		}

	case "gas":
		gasReadingsSection := text
		if idx := strings.Index(strings.ToLower(text), "letture e consumi"); idx != -1 {
			gasReadingsSection = text[idx:]
			if len(gasReadingsSection) > 2000 {
				gasReadingsSection = gasReadingsSection[:2000]
			}
		} else if idx := strings.Index(strings.ToLower(text), "dettaglio letture"); idx != -1 {
			gasReadingsSection = text[idx:]
			if len(gasReadingsSection) > 2000 {
				gasReadingsSection = gasReadingsSection[:2000]
			}
		}

		gasPattern := regexp.MustCompile(`(\d[\d.,]+)\s*-\s*(\d[\d.,]+)\s*=`)
		matches := gasPattern.FindAllStringSubmatch(gasReadingsSection, -1)

		log.Printf("Found %d reading matches in gas bill", len(matches))

		if len(matches) > 0 {
			lastMatch := matches[len(matches)-1]
			reading := h.parseAmount(lastMatch[1])
			extracted.ProviderReading = &reading
			log.Printf("Extracted gas reading: %.2f mc (last of %d matches)", reading, len(matches))
		}

	case "water":
		waterPatterns := []string{
			`[Ll]ettura\s+(?:attuale|finale)[:\s]*(\d+[\.,]?\d*)`,
			`(\d+[\.,]\d+)\s*-\s*(\d+[\.,]\d+)\s*=\s*(\d+[\.,]\d+)`,
		}
		for _, pattern := range waterPatterns {
			re := regexp.MustCompile(pattern)
			if match := re.FindStringSubmatch(text); len(match) > 1 {
				reading := h.parseAmount(match[1])
				extracted.ProviderReading = &reading
				log.Printf("Extracted water reading: %.2f mc", reading)
				break
			}
		}
	}
}

// extractContractDataDefault extracts contract data using default patterns.
func (h *PDFHandler) extractContractDataDefault(text string, pdfURL string) ExtractedContractData {
	extracted := ExtractedContractData{
		PDFURL:  pdfURL,
		RawText: text,
	}

	podPattern := regexp.MustCompile(`[Cc]odice\s+POD[:\s]*(IT\d{3}E\d+)`)
	if match := podPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.ServiceCode = match[1]
	}

	pdrPattern := regexp.MustCompile(`[Cc]odice\s+PDR[:\s]*(\d+)`)
	if match := pdrPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.ServiceCode = match[1]
	}

	customerPattern := regexp.MustCompile(`[Cc]odice\s+[Cc]liente[:\s]*([A-Z0-9]+)`)
	if match := customerPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.CustomerCode = match[1]
	}

	powerPattern := regexp.MustCompile(`[Pp]otenza\s+(?:impegnata|disponibile)[:\s]*(\d+[.,]?\d*)\s*kW`)
	if match := powerPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.PowerCapacity = match[1]
	}

	return extracted
}

// parseAmount converts Italian number format (1.234,56) to float.
func (h *PDFHandler) parseAmount(value string) float64 {
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", ".")
	f, _ := strconv.ParseFloat(value, 64)
	return f
}

// parseDate parses various date formats to ISO (YYYY-MM-DD).
func (h *PDFHandler) parseDate(value string, format string) string {
	formats := []string{
		"02/01/2006",
		"2/1/2006",
		"02-01-2006",
		"02.01.2006",
		"02/01/06",
	}

	if format != "" {
		formats = append([]string{format}, formats...)
	}

	for _, f := range formats {
		if t, err := time.Parse(f, value); err == nil {
			return t.Format("2006-01-02")
		}
	}

	return value
}

// parseItalianDate builds an ISO date from day + Italian month name + year.
func (h *PDFHandler) parseItalianDate(day, month, year string) string {
	months := map[string]int{
		"gennaio": 1, "febbraio": 2, "marzo": 3, "aprile": 4,
		"maggio": 5, "giugno": 6, "luglio": 7, "agosto": 8,
		"settembre": 9, "ottobre": 10, "novembre": 11, "dicembre": 12,
	}

	d, _ := strconv.Atoi(day)
	y, _ := strconv.Atoi(year)
	m := months[strings.ToLower(month)]

	if m > 0 && d > 0 && y > 0 {
		return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
	}
	return ""
}

