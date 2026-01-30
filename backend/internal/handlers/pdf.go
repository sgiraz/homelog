package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"gorm.io/gorm"
)

type PDFHandler struct {
	db         *gorm.DB
	uploadsDir string
}

func NewPDFHandler(db *gorm.DB) *PDFHandler {
	uploadsDir := "./data/uploads"
	// Ensure uploads directory exists
	os.MkdirAll(uploadsDir, 0755)
	return &PDFHandler{db: db, uploadsDir: uploadsDir}
}

// BillTemplateRule defines extraction patterns for bill fields
type BillTemplateRule struct {
	Field   string `json:"field"`   // bill_number, period_start, period_end, due_date, amount_total, consumption_total
	Pattern string `json:"pattern"` // Regex pattern with capture group
	Format  string `json:"format"`  // Optional date format for parsing (e.g., "02/01/2006")
}

// ExtractedBillData represents data extracted from a PDF bill
type ExtractedBillData struct {
	BillNumber       string  `json:"bill_number,omitempty"`
	IssueDate        string  `json:"issue_date,omitempty"`
	PeriodStart      string  `json:"period_start,omitempty"`
	PeriodEnd        string  `json:"period_end,omitempty"`
	DueDate          string  `json:"due_date,omitempty"`
	AmountTotal      float64 `json:"amount_total,omitempty"`
	ConsumptionTotal float64 `json:"consumption_total,omitempty"`
	Provider         string  `json:"provider,omitempty"`
	ServiceCode      string  `json:"service_code,omitempty"` // POD or PDR
	CustomerCode     string  `json:"customer_code,omitempty"`
	PDFURL           string  `json:"pdf_url,omitempty"`
	RawText          string  `json:"raw_text,omitempty"` // For debugging/template creation

	// Provider meter readings (letture rilevate dal fornitore)
	ProviderReadingDate string   `json:"provider_reading_date,omitempty"` // Data lettura
	ProviderReadingF1   *float64 `json:"provider_reading_f1,omitempty"`   // Lettura F1 (electricity peak)
	ProviderReadingF2   *float64 `json:"provider_reading_f2,omitempty"`   // Lettura F2 (electricity mid)
	ProviderReadingF3   *float64 `json:"provider_reading_f3,omitempty"`   // Lettura F3 (electricity off-peak)
	ProviderReading     *float64 `json:"provider_reading,omitempty"`      // Lettura singola (gas/water mc)
	ReadingType         string   `json:"reading_type,omitempty"`          // actual (rilevata), estimated (stimata)
}

// ExtractedContractData represents data extracted from a contract PDF
type ExtractedContractData struct {
	Provider      string `json:"provider,omitempty"`
	ServiceCode   string `json:"service_code,omitempty"` // POD or PDR
	CustomerCode  string `json:"customer_code,omitempty"`
	Address       string `json:"address,omitempty"`
	PowerCapacity string `json:"power_capacity,omitempty"`
	PDFURL        string `json:"pdf_url,omitempty"`
	RawText       string `json:"raw_text,omitempty"`
}

// UploadBillPDF - POST /api/v1/utilities/:id/bills/upload
// Uploads a PDF and extracts bill data using templates
func (h *PDFHandler) UploadBillPDF(c *gin.Context) {
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

	// Get the uploaded file
	file, err := c.FormFile("pdf_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No PDF file uploaded"})
		return
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".pdf") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are allowed"})
		return
	}

	// Save the file
	filename := fmt.Sprintf("bill_%d_%d.pdf", utilityID, time.Now().Unix())
	filepath := filepath.Join(h.uploadsDir, filename)

	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	pdfURL := "/uploads/" + filename

	// Extract text from PDF
	text, err := h.extractTextFromPDF(filepath)
	if err != nil {
		log.Printf("Warning: Could not extract text from PDF: %v", err)
		// Continue anyway, return the PDF URL
		c.JSON(http.StatusOK, gin.H{
			"pdf_url":  pdfURL,
			"raw_text": "",
			"message":  "PDF uploaded but text extraction failed",
		})
		return
	}

	// Try to find a matching template
	var template models.BillTemplate
	templateFound := h.db.Where("user_id = ? AND provider = ? AND utility_type = ?",
		userID, utility.Provider, utility.Type).
		Order("is_default DESC").
		First(&template).Error == nil

	var extracted ExtractedBillData
	extracted.PDFURL = pdfURL
	extracted.RawText = text

	if templateFound {
		// Extract data using template rules
		extracted = h.extractBillDataWithTemplate(text, template, pdfURL)
	} else {
		// Use default extraction patterns
		extracted = h.extractBillDataDefault(text, utility.Type, pdfURL)
	}

	log.Printf("Extracted bill data: %+v", extracted)
	c.JSON(http.StatusOK, extracted)
}

// UploadContractPDF - POST /api/v1/utilities/contract/upload
// Uploads a contract PDF and extracts utility data
func (h *PDFHandler) UploadContractPDF(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get the uploaded file
	file, err := c.FormFile("pdf_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No PDF file uploaded"})
		return
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".pdf") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are allowed"})
		return
	}

	// Save the file
	filename := fmt.Sprintf("contract_%d_%d.pdf", userID, time.Now().Unix())
	filepath := filepath.Join(h.uploadsDir, filename)

	if err := c.SaveUploadedFile(file, filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	pdfURL := "/uploads/" + filename

	// Extract text from PDF
	text, err := h.extractTextFromPDF(filepath)
	if err != nil {
		log.Printf("Warning: Could not extract text from PDF: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"pdf_url":  pdfURL,
			"raw_text": "",
			"message":  "PDF uploaded but text extraction failed",
		})
		return
	}

	// Extract contract data using default patterns
	extracted := h.extractContractDataDefault(text, pdfURL)

	log.Printf("Extracted contract data: %+v", extracted)
	c.JSON(http.StatusOK, extracted)
}

// extractTextFromPDF uses pdftotext (poppler-utils) to extract text
func (h *PDFHandler) extractTextFromPDF(pdfPath string) (string, error) {
	// Try pdftotext first (Linux/Mac with poppler-utils)
	cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err == nil {
		return string(output), nil
	}

	// On Windows, try reading the PDF directly (basic extraction)
	// This is a fallback that works for simple text PDFs
	file, err := os.Open(pdfPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// Extract visible text between stream/endstream (very basic)
	text := extractBasicPDFText(string(content))
	if text != "" {
		return text, nil
	}

	return "", fmt.Errorf("could not extract text from PDF")
}

// extractBasicPDFText does basic text extraction from PDF content
func extractBasicPDFText(content string) string {
	var texts []string

	// Find text between BT and ET markers
	btPattern := regexp.MustCompile(`BT\s*(.*?)\s*ET`)
	matches := btPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			// Extract text from Tj and TJ operators
			tjPattern := regexp.MustCompile(`\((.*?)\)\s*Tj`)
			tjMatches := tjPattern.FindAllStringSubmatch(match[1], -1)
			for _, tj := range tjMatches {
				if len(tj) > 1 {
					texts = append(texts, tj[1])
				}
			}
		}
	}

	return strings.Join(texts, " ")
}

// extractBillDataWithTemplate extracts bill data using template rules
func (h *PDFHandler) extractBillDataWithTemplate(text string, template models.BillTemplate, pdfURL string) ExtractedBillData {
	var rules []BillTemplateRule
	json.Unmarshal([]byte(template.ExtractionRules), &rules)

	extracted := ExtractedBillData{
		PDFURL:   pdfURL,
		RawText:  text,
		Provider: template.Provider,
	}

	for _, rule := range rules {
		re := regexp.MustCompile(rule.Pattern)
		match := re.FindStringSubmatch(text)
		if len(match) > 1 {
			value := strings.TrimSpace(match[1])
			switch rule.Field {
			case "bill_number":
				extracted.BillNumber = value
			case "issue_date":
				extracted.IssueDate = h.parseDate(value, rule.Format)
			case "period_start":
				extracted.PeriodStart = h.parseDate(value, rule.Format)
			case "period_end":
				extracted.PeriodEnd = h.parseDate(value, rule.Format)
			case "due_date":
				extracted.DueDate = h.parseDate(value, rule.Format)
			case "amount_total":
				extracted.AmountTotal = h.parseAmount(value)
			case "consumption_total":
				extracted.ConsumptionTotal = h.parseAmount(value)
			case "service_code":
				extracted.ServiceCode = value
			case "customer_code":
				extracted.CustomerCode = value
			}
		}
	}

	return extracted
}

// extractBillDataDefault uses default patterns for common Italian utilities
func (h *PDFHandler) extractBillDataDefault(text string, utilityType string, pdfURL string) ExtractedBillData {
	extracted := ExtractedBillData{
		PDFURL:  pdfURL,
		RawText: text,
	}

	// Provider detection
	providers := []string{"E.ON", "Enel", "ETRA", "Eni", "Edison", "Acea", "A2A", "Iren", "Hera"}
	for _, p := range providers {
		if strings.Contains(strings.ToUpper(text), strings.ToUpper(p)) {
			extracted.Provider = p
			break
		}
	}

	// Bill number patterns
	billPatterns := []string{
		`n[°º]\s*(\d+)`,
		`[Ff]attura\s+n[.°]?\s*(\d+)`,
		`[Nn]umero\s+[Ff]attura[:\s]*(\d+)`,
	}
	for _, pattern := range billPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			extracted.BillNumber = match[1]
			break
		}
	}

	// Amount patterns (Euro)
	amountPatterns := []string{
		`[Tt]otale\s+da\s+pagare\s*[:\s]*(\d+[.,]\d{2})\s*[€E]`,
		`[Ii]mporto\s+totale\s+da\s+pagare\s*[:\s]*(\d+[.,]\d{2})\s*[€E]`,
		`(\d+[.,]\d{2})\s*[€E]\s*[Tt]otale`,
		`[€E]\s*(\d+[.,]\d{2})`,
	}
	for _, pattern := range amountPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			extracted.AmountTotal = h.parseAmount(match[1])
			break
		}
	}

	// Consumption patterns
	var consumptionPattern string
	switch utilityType {
	case "electricity":
		consumptionPattern = `[Cc]onsumo\s+totale[^0-9]*(\d+[.,]?\d*)\s*kWh`
	case "gas":
		consumptionPattern = `[Cc]onsumo\s+totale[^0-9]*(\d+[.,]?\d*)\s*[Ss]mc`
	case "water":
		consumptionPattern = `[Cc]onsumo\s+totale[^0-9]*(\d+[.,]?\d*)\s*mc`
	}
	if consumptionPattern != "" {
		re := regexp.MustCompile(consumptionPattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			extracted.ConsumptionTotal = h.parseAmount(match[1])
		}
	}

	// Issue date patterns (e.g., "emessa il 19 gennaio 2026")
	issueDatePatterns := []string{
		`emessa\s+il\s+(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`,
		`[Dd]ata\s+emissione[:\s]*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
	}
	for _, pattern := range issueDatePatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			if len(match) == 4 {
				// Italian month format: day, month name, year
				extracted.IssueDate = h.parseItalianDate(match[1], match[2], match[3])
			} else {
				extracted.IssueDate = h.parseDate(match[1], "")
			}
			log.Printf("Extracted issue date: %s", extracted.IssueDate)
			break
		}
	}

	// Due date patterns - try specific patterns first
	// E.ON format: "Data di scadenza ... 13 febbraio 2026" or "scadenza 13 febbraio 2026"
	dueDatePatterns := []string{
		// Italian month format near "scadenza"
		`[Ss]cadenza[^0-9]*(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`,
		// Numeric format
		`[Ss]cadenza[:\s]*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
		`[Dd]ata\s+di\s+scadenza[:\s]*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
		`entro\s+il\s+(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`,
	}
	for _, pattern := range dueDatePatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			if len(match) == 4 {
				// Italian month format: day, month name, year
				extracted.DueDate = h.parseItalianDate(match[1], strings.ToLower(match[2]), match[3])
				log.Printf("Extracted due date (Italian): %s", extracted.DueDate)
			} else {
				extracted.DueDate = h.parseDate(match[1], "")
				log.Printf("Extracted due date (numeric): %s", extracted.DueDate)
			}
			break
		}
	}

	// Period patterns - try Italian month format first (more specific)
	// E.ON format: "01 dicembre 2025 - 31 dicembre 2025"
	monthPeriodPattern := regexp.MustCompile(`(?i)(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})\s*[-–]\s*(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
	if match := monthPeriodPattern.FindStringSubmatch(text); len(match) > 6 {
		extracted.PeriodStart = h.parseItalianDate(match[1], strings.ToLower(match[2]), match[3])
		extracted.PeriodEnd = h.parseItalianDate(match[4], strings.ToLower(match[5]), match[6])
		log.Printf("Extracted period (Italian): %s - %s", extracted.PeriodStart, extracted.PeriodEnd)
	}

	// Fallback: numeric date format
	if extracted.PeriodStart == "" {
		periodPattern := regexp.MustCompile(`(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})\s*[-–]\s*(\d{1,2}[/.-]\d{1,2}[/.-]\d{2,4})`)
		if match := periodPattern.FindStringSubmatch(text); len(match) > 2 {
			extracted.PeriodStart = h.parseDate(match[1], "")
			extracted.PeriodEnd = h.parseDate(match[2], "")
			log.Printf("Extracted period (numeric): %s - %s", extracted.PeriodStart, extracted.PeriodEnd)
		}
	}

	// POD/PDR code
	if utilityType == "electricity" {
		podPattern := regexp.MustCompile(`[Cc]odice\s+POD[:\s]*(IT\d{3}E\d+)`)
		if match := podPattern.FindStringSubmatch(text); len(match) > 1 {
			extracted.ServiceCode = match[1]
		}
	} else if utilityType == "gas" {
		pdrPattern := regexp.MustCompile(`[Cc]odice\s+PDR[:\s]*(\d+)`)
		if match := pdrPattern.FindStringSubmatch(text); len(match) > 1 {
			extracted.ServiceCode = match[1]
		}
	}

	// Customer code
	customerPattern := regexp.MustCompile(`[Cc]odice\s+[Cc]liente[:\s]*([A-Z0-9]+)`)
	if match := customerPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.CustomerCode = match[1]
	}

	// Extract provider meter readings (letture rilevate)
	h.extractProviderReadings(&extracted, text, utilityType)

	return extracted
}

// extractProviderReadings extracts meter readings from the bill
// For electricity: F1, F2, F3 values (kWh)
// For gas/water: single meter reading (mc/Smc)
func (h *PDFHandler) extractProviderReadings(extracted *ExtractedBillData, text string, utilityType string) {
	// Detect reading type (actual vs estimated)
	if strings.Contains(strings.ToLower(text), "effettiva") || strings.Contains(strings.ToLower(text), "rilevata") {
		extracted.ReadingType = "actual"
	} else if strings.Contains(strings.ToLower(text), "stimata") || strings.Contains(strings.ToLower(text), "stima") {
		extracted.ReadingType = "estimated"
	}

	// Extract reading period from "Da DD/MM/YYYY a DD/MM/YYYY" pattern (E.ON format)
	// This is in the "Letture e consumi" section
	readingPeriodPattern := regexp.MustCompile(`Da\s+(\d{1,2}/\d{1,2}/\d{4})\s*[^\d]*a\s+(\d{1,2}/\d{1,2}/\d{4})`)
	if match := readingPeriodPattern.FindStringSubmatch(text); len(match) > 2 {
		// Use the end date of the reading period as the provider reading date
		extracted.ProviderReadingDate = h.parseDate(match[2], "")
		log.Printf("Extracted reading period: %s - %s", match[1], match[2])
	}

	switch utilityType {
	case "electricity":
		// E.ON format for electricity readings (from "Letture e consumi" section):
		// "1.936,56 - 1.899,18 = 37,38    2.005,19 - 1.965,35 = 39,84    2.519,09 - 2.453,21 = 65,88"
		// Pattern: final_reading - initial_reading = consumption (repeated 3 times for F1, F2, F3)
		// We want the FINAL readings (lettura finale)

		// Multiple patterns to try
		// Pattern 1: Standard format "X,XX - Y,YY = Z,ZZ"
		electricPattern := regexp.MustCompile(`(\d+[\.,]\d+)\s*-\s*(\d+[\.,]\d+)\s*=\s*(\d+[\.,]\d+)`)
		matches := electricPattern.FindAllStringSubmatch(text, -1)

		log.Printf("Found %d reading matches in electricity bill", len(matches))

		if len(matches) >= 3 {
			// F1, F2, F3 final readings (first value in each match is the final reading)
			f1 := h.parseAmount(matches[0][1])
			f2 := h.parseAmount(matches[1][1])
			f3 := h.parseAmount(matches[2][1])
			extracted.ProviderReadingF1 = &f1
			extracted.ProviderReadingF2 = &f2
			extracted.ProviderReadingF3 = &f3
			log.Printf("Extracted electricity readings: F1=%.2f, F2=%.2f, F3=%.2f", f1, f2, f3)
		}

	case "gas":
		// E.ON format for gas readings:
		// "6.299,00 - 6.180,00 = 119,00"
		// Pattern: final_reading - initial_reading = consumption
		gasPattern := regexp.MustCompile(`(\d+[\.,]\d+)\s*-\s*(\d+[\.,]\d+)\s*=\s*(\d+[\.,]\d+)`)
		matches := gasPattern.FindAllStringSubmatch(text, -1)

		log.Printf("Found %d reading matches in gas bill", len(matches))

		if len(matches) > 0 {
			// Get the first match (there should be only one for gas)
			reading := h.parseAmount(matches[0][1])
			extracted.ProviderReading = &reading
			log.Printf("Extracted gas reading: %.2f mc", reading)
		}

	case "water":
		// ETRA format for water readings - look for "Lettura attuale" or similar
		// Pattern similar to gas
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

// extractContractDataDefault extracts contract data using default patterns
func (h *PDFHandler) extractContractDataDefault(text string, pdfURL string) ExtractedContractData {
	extracted := ExtractedContractData{
		PDFURL:  pdfURL,
		RawText: text,
	}

	// Provider detection
	providers := []string{"E.ON", "Enel", "ETRA", "Eni", "Edison", "Acea", "A2A", "Iren", "Hera"}
	for _, p := range providers {
		if strings.Contains(strings.ToUpper(text), strings.ToUpper(p)) {
			extracted.Provider = p
			break
		}
	}

	// POD code (electricity)
	podPattern := regexp.MustCompile(`[Cc]odice\s+POD[:\s]*(IT\d{3}E\d+)`)
	if match := podPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.ServiceCode = match[1]
	}

	// PDR code (gas)
	pdrPattern := regexp.MustCompile(`[Cc]odice\s+PDR[:\s]*(\d+)`)
	if match := pdrPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.ServiceCode = match[1]
	}

	// Customer code
	customerPattern := regexp.MustCompile(`[Cc]odice\s+[Cc]liente[:\s]*([A-Z0-9]+)`)
	if match := customerPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.CustomerCode = match[1]
	}

	// Power capacity (kW)
	powerPattern := regexp.MustCompile(`[Pp]otenza\s+(?:impegnata|disponibile)[:\s]*(\d+[.,]?\d*)\s*kW`)
	if match := powerPattern.FindStringSubmatch(text); len(match) > 1 {
		extracted.PowerCapacity = match[1]
	}

	return extracted
}

// parseAmount converts Italian number format to float
func (h *PDFHandler) parseAmount(value string) float64 {
	// Replace Italian decimal separator
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", ".")
	f, _ := strconv.ParseFloat(value, 64)
	return f
}

// parseDate parses various date formats to ISO format
func (h *PDFHandler) parseDate(value string, format string) string {
	// Try various formats
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

// parseItalianDate parses Italian month names
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

// === Template CRUD Operations ===

// ListBillTemplates - GET /api/v1/templates/bills
func (h *PDFHandler) ListBillTemplates(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var templates []models.BillTemplate
	query := h.db.Where("user_id = ?", userID)

	if provider := c.Query("provider"); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if utilityType := c.Query("utility_type"); utilityType != "" {
		query = query.Where("utility_type = ?", utilityType)
	}

	if err := query.Order("provider ASC, utility_type ASC").Find(&templates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// CreateBillTemplate - POST /api/v1/templates/bills
func (h *PDFHandler) CreateBillTemplate(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		Name            string                   `json:"name" binding:"required"`
		Provider        string                   `json:"provider" binding:"required"`
		UtilityType     string                   `json:"utility_type" binding:"required,oneof=electricity gas water waste"`
		IsDefault       bool                     `json:"is_default"`
		ExtractionRules []BillTemplateRule `json:"extraction_rules"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert rules to JSON
	rulesJSON, _ := json.Marshal(input.ExtractionRules)

	// If setting as default, unset other defaults for same provider+type
	if input.IsDefault {
		h.db.Model(&models.BillTemplate{}).
			Where("user_id = ? AND provider = ? AND utility_type = ?", userID, input.Provider, input.UtilityType).
			Update("is_default", false)
	}

	template := models.BillTemplate{
		UserID:          userID,
		Name:            input.Name,
		Provider:        input.Provider,
		UtilityType:     input.UtilityType,
		IsDefault:       input.IsDefault,
		ExtractionRules: string(rulesJSON),
	}

	if err := h.db.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// UpdateBillTemplate - PUT /api/v1/templates/bills/:id
func (h *PDFHandler) UpdateBillTemplate(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var template models.BillTemplate
	if err := h.db.Where("id = ? AND user_id = ?", templateID, userID).First(&template).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	var input struct {
		Name            string                   `json:"name"`
		Provider        string                   `json:"provider"`
		UtilityType     string                   `json:"utility_type"`
		IsDefault       bool                     `json:"is_default"`
		ExtractionRules []BillTemplateRule `json:"extraction_rules"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != "" {
		template.Name = input.Name
	}
	if input.Provider != "" {
		template.Provider = input.Provider
	}
	if input.UtilityType != "" {
		template.UtilityType = input.UtilityType
	}
	if input.ExtractionRules != nil {
		rulesJSON, _ := json.Marshal(input.ExtractionRules)
		template.ExtractionRules = string(rulesJSON)
	}

	// Handle default flag
	if input.IsDefault && !template.IsDefault {
		h.db.Model(&models.BillTemplate{}).
			Where("user_id = ? AND provider = ? AND utility_type = ? AND id != ?",
				userID, template.Provider, template.UtilityType, template.ID).
			Update("is_default", false)
	}
	template.IsDefault = input.IsDefault

	if err := h.db.Save(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update template"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// DeleteBillTemplate - DELETE /api/v1/templates/bills/:id
func (h *PDFHandler) DeleteBillTemplate(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	templateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	result := h.db.Where("id = ? AND user_id = ?", templateID, userID).Delete(&models.BillTemplate{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete template"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Template deleted successfully"})
}

// GetPDFRawText - POST /api/v1/pdf/extract-text
// Helper endpoint to extract raw text from a PDF for template creation
func (h *PDFHandler) GetPDFRawText(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	file, err := c.FormFile("pdf_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No PDF file uploaded"})
		return
	}

	// Save temporarily
	tempFile := filepath.Join(h.uploadsDir, fmt.Sprintf("temp_%d.pdf", time.Now().UnixNano()))
	defer os.Remove(tempFile)

	if err := c.SaveUploadedFile(file, tempFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	text, err := h.extractTextFromPDF(tempFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract text: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"raw_text": text,
		"length":   len(text),
	})
}
