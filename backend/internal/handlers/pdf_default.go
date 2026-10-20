package handlers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

// italianTextDateRe matches the extended Italian date form "01 aprile 2026"
// (day, month name, year). Newer bills (e.g. Eon) write the billing period this
// way instead of the numeric dd/mm/yyyy form.
var italianTextDateRe = regexp.MustCompile(`(?i)^(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})$`)

// parseDate parses various date formats to ISO (YYYY-MM-DD), accepting both the
// numeric dd/mm/yyyy form and the extended "01 aprile 2026" Italian form.
func (h *PDFHandler) parseDate(value string, format string) string {
	value = strings.TrimSpace(value)

	// Extended Italian form ("01 aprile 2026"): reuse the month-name parser.
	if m := italianTextDateRe.FindStringSubmatch(value); m != nil {
		if iso := h.parseItalianDate(m[1], m[2], m[3]); iso != "" {
			return iso
		}
	}

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
