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
	"sort"
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
	Field        string `json:"field"`         // bill_number, period_start, period_end, due_date, amount_total, consumption_total
	Pattern      string `json:"pattern"`       // Regex pattern with capture group (full pattern with context)
	ValuePattern string `json:"value_pattern"` // Pattern for just the value (optional, used by drag-and-drop UI)
	Prefix       string `json:"prefix"`        // Context prefix text (optional, for display/debugging)
	Suffix       string `json:"suffix"`        // Context suffix text (optional, for display/debugging)
	Format       string `json:"format"`        // Optional date format for parsing (e.g., "02/01/2006")

	// Position-based extraction (for accurate matching)
	Page         int     `json:"page"`          // Page number (0-indexed)
	X            float64 `json:"x"`             // X coordinate in PDF units
	Y            float64 `json:"y"`             // Y coordinate in PDF units
	Width        float64 `json:"width"`         // Width of the word
	Height       float64 `json:"height"`        // Height of the word
	ContextLeft  string  `json:"context_left"`  // Words to the left for context validation
	ContextAbove string  `json:"context_above"` // Words above for context validation

	// Anchor-based extraction (relative positioning, resilient to layout changes)
	AnchorText      string `json:"anchor_text"`      // Label text to find as anchor (e.g., "Importo totale da pagare")
	AnchorDirection string `json:"anchor_direction"` // "right", "below", "right_or_below"
	GlobalSearch    bool   `json:"global_search"`    // For unique fields (POD/PDR): search entire document
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

	// Gas conversion coefficient (Coefficiente di conversione C)
	ConversionCoefficient *float64 `json:"conversion_coefficient,omitempty"`

	// Estimated reading/consumption fields
	EstimatedDate                *string  `json:"estimated_date,omitempty"`
	EstimatedReading             *float64 `json:"estimated_reading,omitempty"`     // Lettura stimata (mc)
	EstimatedConsumption         *float64 `json:"estimated_consumption,omitempty"` // Consumo stimato (Smc)
	PreviousEstimatedConsumption *float64 `json:"previous_estimated_consumption,omitempty"`
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

	// Check if a specific template was requested
	var template models.BillTemplate
	templateFound := false

	if templateIDStr := c.PostForm("template_id"); templateIDStr != "" {
		// User specified a template - use it if valid
		templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
		if err == nil {
			templateFound = h.db.Where("id = ? AND user_id = ?", templateID, userID).
				First(&template).Error == nil
			if templateFound {
				log.Printf("Using user-specified template ID %d: %s", templateID, template.Name)
			} else {
				log.Printf("Requested template ID %d not found, falling back to auto-detection", templateID)
			}
		}
	}

	// If no template specified or not found, try to find a matching template by provider
	if !templateFound {
		templateFound = h.db.Where("user_id = ? AND provider = ? AND utility_type = ?",
			userID, utility.Provider, utility.Type).
			Order("is_default DESC").
			First(&template).Error == nil
	}

	var extracted ExtractedBillData
	extracted.PDFURL = pdfURL
	extracted.RawText = text

	if templateFound {
		// Extract words with positions for position-based matching
		var words []WordInfo
		textWithPos, posErr := h.extractTextWithPositions(filepath)
		if posErr == nil && textWithPos != nil && len(textWithPos.Words) > 0 {
			words = textWithPos.Words
			log.Printf("Extracted %d words with positions for template matching", len(words))
		}

		// Extract data using template rules with position data
		extracted = h.extractBillDataWithTemplateAndWords(text, template, pdfURL, words)

		// Build set of fields the template defines rules for
		templateFields := h.getTemplateFieldSet(template)

		// Only supplement fields the template does NOT define rules for.
		// When a template has a rule for a field, trust it (even if empty) -
		// wrong default values are worse than missing values.
		defaultExtracted := h.extractBillDataDefault(text, utility.Type, pdfURL)
		extracted = h.mergeExtracted(extracted, defaultExtracted, templateFields)
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

// win1252ToUTF8 maps Windows-1252 specific bytes (0x80-0x9F range) to their
// correct UTF-8 equivalents. These bytes differ from Latin-1/ISO-8859-1.
var win1252ToUTF8 = map[byte][]byte{
	0x80: []byte("€"),      // U+20AC Euro sign
	0x82: []byte("‚"),      // U+201A Single low-9 quotation mark
	0x83: []byte("ƒ"),      // U+0192 Latin small letter f with hook
	0x84: []byte("„"),      // U+201E Double low-9 quotation mark
	0x85: []byte("…"),      // U+2026 Horizontal ellipsis
	0x86: []byte("†"),      // U+2020 Dagger
	0x87: []byte("‡"),      // U+2021 Double dagger
	0x88: []byte("ˆ"),      // U+02C6 Modifier letter circumflex accent
	0x89: []byte("‰"),      // U+2030 Per mille sign
	0x8A: []byte("Š"),      // U+0160 Latin capital letter S with caron
	0x8B: []byte("‹"),      // U+2039 Single left-pointing angle quotation mark
	0x8C: []byte("Œ"),      // U+0152 Latin capital ligature OE
	0x8E: []byte("Ž"),      // U+017D Latin capital letter Z with caron
	0x91: []byte("\u2018"), // U+2018 Left single quotation mark
	0x92: []byte("\u2019"), // U+2019 Right single quotation mark
	0x93: []byte("\u201C"), // U+201C Left double quotation mark
	0x94: []byte("\u201D"), // U+201D Right double quotation mark
	0x95: []byte("•"),      // U+2022 Bullet
	0x96: []byte("\u2013"), // U+2013 En dash
	0x97: []byte("\u2014"), // U+2014 Em dash
	0x98: []byte("˜"),      // U+02DC Small tilde
	0x99: []byte("™"),      // U+2122 Trade mark sign
	0x9A: []byte("š"),      // U+0161 Latin small letter s with caron
	0x9B: []byte("›"),      // U+203A Single right-pointing angle quotation mark
	0x9C: []byte("œ"),      // U+0153 Latin small ligature oe
	0x9E: []byte("ž"),      // U+017E Latin small letter z with caron
	0x9F: []byte("Ÿ"),      // U+0178 Latin capital letter Y with diaeresis
}

// normalizeLatin1ToUTF8 converts bytes that may contain Latin-1 or Windows-1252
// encoded characters to valid UTF-8. pdftotext on Windows often outputs Latin-1
// (e.g., ° as 0xB0) or Windows-1252 (e.g., € as 0x80) instead of proper UTF-8.
func normalizeLatin1ToUTF8(data []byte) string {
	var result []byte
	for i := 0; i < len(data); i++ {
		b := data[i]
		if b < 0x80 {
			// ASCII: pass through
			result = append(result, b)
		} else if b >= 0xF0 && b <= 0xF7 && i+3 < len(data) &&
			data[i+1] >= 0x80 && data[i+1] < 0xC0 &&
			data[i+2] >= 0x80 && data[i+2] < 0xC0 &&
			data[i+3] >= 0x80 && data[i+3] < 0xC0 {
			// Valid 4-byte UTF-8 sequence (U+10000..U+10FFFF): pass through
			result = append(result, b, data[i+1], data[i+2], data[i+3])
			i += 3
		} else if b >= 0xE0 && b <= 0xEF && i+2 < len(data) &&
			data[i+1] >= 0x80 && data[i+1] < 0xC0 &&
			data[i+2] >= 0x80 && data[i+2] < 0xC0 {
			// Valid 3-byte UTF-8 sequence (U+0800..U+FFFF): pass through
			// This includes € (U+20AC = 0xE2 0x82 0xAC) and other common symbols
			result = append(result, b, data[i+1], data[i+2])
			i += 2
		} else if b >= 0xC0 && b <= 0xDF && i+1 < len(data) &&
			data[i+1] >= 0x80 && data[i+1] < 0xC0 {
			// Valid 2-byte UTF-8 sequence (U+0080..U+07FF): pass through
			result = append(result, b, data[i+1])
			i++
		} else if b >= 0x80 && b < 0xC0 {
			// Bare continuation byte: not part of a valid UTF-8 sequence
			// Check Windows-1252 specific mappings first (0x80-0x9F differ from Latin-1)
			if mapped, ok := win1252ToUTF8[b]; ok {
				result = append(result, mapped...)
			} else {
				// Bare Latin-1 byte (0xA0-0xBF): convert to UTF-8
				// In UTF-8, code points 0x80-0xBF are encoded as 0xC2 0x80-0xBF
				result = append(result, 0xC2, b)
			}
		} else if b >= 0xC0 {
			// Multi-byte start without valid continuation: treat as Latin-1
			result = append(result, 0xC3, b-0x40)
		} else {
			result = append(result, b)
		}
	}
	return string(result)
}

// extractTextFromPDF uses pdftotext (poppler-utils) to extract text
func (h *PDFHandler) extractTextFromPDF(pdfPath string) (string, error) {
	// Try pdftotext first (Linux/Mac with poppler-utils)
	cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
	output, err := cmd.Output()
	if err == nil {
		return normalizeLatin1ToUTF8(output), nil
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

// WordInfo represents a word with its position and bounding box
type WordInfo struct {
	Text      string  `json:"text"`
	LineIndex int     `json:"lineIndex"`
	WordIndex int     `json:"wordIndex"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Page      int     `json:"page"`
}

// ExtractedTextWithPositions represents text with word positions
type ExtractedTextWithPositions struct {
	RawText   string     `json:"raw_text"`
	Words     []WordInfo `json:"words"`
	HasBBox   bool       `json:"has_bbox"`
	PageCount int        `json:"page_count"`
}

// extractTextWithPositions extracts text with word positions using pdftotext -bbox
func (h *PDFHandler) extractTextWithPositions(pdfPath string) (*ExtractedTextWithPositions, error) {
	result := &ExtractedTextWithPositions{
		Words:   []WordInfo{},
		HasBBox: false,
	}

	// Try pdftotext -bbox first (poppler-utils on Linux/Docker)
	cmd := exec.Command("pdftotext", "-bbox", pdfPath, "-")
	output, err := cmd.Output()
	if err == nil {
		// Parse the HTML/XML output from -bbox (normalize encoding)
		words, pageCount, parseErr := parsePdftextBboxOutput(normalizeLatin1ToUTF8(output))
		if parseErr == nil && len(words) > 0 {
			result.Words = words
			result.HasBBox = true
			result.PageCount = pageCount
			// Also get raw text
			rawCmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
			rawOutput, _ := rawCmd.Output()
			result.RawText = normalizeLatin1ToUTF8(rawOutput)
			return result, nil
		}
		log.Printf("Warning: Could not parse -bbox output, falling back to layout: %v", parseErr)
	}

	// Fallback to -layout and calculate relative positions
	cmd = exec.Command("pdftotext", "-layout", pdfPath, "-")
	output, err = cmd.Output()
	if err != nil {
		// Try basic extraction for Windows
		file, fileErr := os.Open(pdfPath)
		if fileErr != nil {
			return nil, fileErr
		}
		defer file.Close()

		content, readErr := io.ReadAll(file)
		if readErr != nil {
			return nil, readErr
		}

		text := extractBasicPDFText(string(content))
		if text == "" {
			return nil, fmt.Errorf("could not extract text from PDF")
		}
		result.RawText = text
		result.Words = extractWordsFromLayoutText(text)
		return result, nil
	}

	normalized := normalizeLatin1ToUTF8(output)
	result.RawText = normalized
	result.Words = extractWordsFromLayoutText(normalized)
	return result, nil
}

// parsePdftextBboxOutput parses the HTML output from pdftotext -bbox
func parsePdftextBboxOutput(htmlOutput string) ([]WordInfo, int, error) {
	var words []WordInfo

	// Pattern to match page elements with their content
	pagePattern := regexp.MustCompile(`<page[^>]*>([\s\S]*?)</page>`)
	// Pattern to match word elements: <word xMin="X" yMin="Y" xMax="X2" yMax="Y2">text</word>
	wordPattern := regexp.MustCompile(`<word\s+xMin="([^"]+)"\s+yMin="([^"]+)"\s+xMax="([^"]+)"\s+yMax="([^"]+)"[^>]*>([^<]+)</word>`)

	// Find all pages
	pageMatches := pagePattern.FindAllStringSubmatch(htmlOutput, -1)
	pageCount := len(pageMatches)

	if pageCount == 0 {
		return nil, 0, fmt.Errorf("no pages found in bbox output")
	}

	// Process each page
	for pageIndex, pageMatch := range pageMatches {
		if len(pageMatch) < 2 {
			continue
		}

		pageContent := pageMatch[1]
		lineIndex := 0
		wordIndexInLine := 0
		lastY := float64(-1)

		// Find all words in this page
		wordMatches := wordPattern.FindAllStringSubmatch(pageContent, -1)
		for _, match := range wordMatches {
			if len(match) < 6 {
				continue
			}

			xMin, _ := strconv.ParseFloat(match[1], 64)
			yMin, _ := strconv.ParseFloat(match[2], 64)
			xMax, _ := strconv.ParseFloat(match[3], 64)
			yMax, _ := strconv.ParseFloat(match[4], 64)
			text := strings.TrimSpace(match[5])

			if text == "" {
				continue
			}

			// Detect new line (Y position changed significantly)
			if lastY >= 0 && (yMin-lastY > 5 || yMin < lastY-5) {
				lineIndex++
				wordIndexInLine = 0
			}
			lastY = yMin

			words = append(words, WordInfo{
				Text:      text,
				LineIndex: lineIndex,
				WordIndex: wordIndexInLine,
				X:         xMin,
				Y:         yMin,
				Width:     xMax - xMin,
				Height:    yMax - yMin,
				Page:      pageIndex,
			})
			wordIndexInLine++
		}
	}

	if len(words) == 0 {
		return nil, 0, fmt.Errorf("no words found in bbox output")
	}

	return words, pageCount, nil
}

// extractWordsFromLayoutText extracts words with relative positions from layout text
func extractWordsFromLayoutText(text string) []WordInfo {
	var words []WordInfo
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Split line into words preserving positions
		wordPattern := regexp.MustCompile(`\S+`)
		matches := wordPattern.FindAllStringIndex(line, -1)

		for wordIdx, match := range matches {
			wordText := line[match[0]:match[1]]
			if wordText == "" {
				continue
			}

			words = append(words, WordInfo{
				Text:      wordText,
				LineIndex: lineIdx,
				WordIndex: wordIdx,
				X:         float64(match[0]), // Character position as X
				Y:         float64(lineIdx),  // Line number as Y
				Width:     float64(match[1] - match[0]),
				Height:    1,
				Page:      0,
			})
		}
	}

	return words
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
	return h.extractBillDataWithTemplateAndWords(text, template, pdfURL, nil)
}

// extractBillDataWithTemplateAndWords extracts bill data using template rules with word position support
func (h *PDFHandler) extractBillDataWithTemplateAndWords(text string, template models.BillTemplate, pdfURL string, words []WordInfo) ExtractedBillData {
	var rules []BillTemplateRule
	json.Unmarshal([]byte(template.ExtractionRules), &rules)

	extracted := ExtractedBillData{
		PDFURL:   pdfURL,
		RawText:  text,
		Provider: template.Provider,
	}

	for _, rule := range rules {
		value := h.extractValueWithRuleAndWords(rule, text, words)
		if value == "" {
			log.Printf("Template extraction FAILED for field '%s' (anchor='%s', pattern='%s')",
				rule.Field, rule.AnchorText, rule.Pattern)
			continue
		}
		log.Printf("Template extraction OK for field '%s': '%s'", rule.Field, value)

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
		case "conversion_coefficient":
			coeff := h.parseAmount(value)
			extracted.ConversionCoefficient = &coeff
		case "provider_reading":
			reading := h.parseAmount(value)
			extracted.ProviderReading = &reading
		case "reading_type":
			lower := strings.ToLower(value)
			if strings.Contains(lower, "stim") || strings.Contains(lower, "presunt") {
				extracted.ReadingType = "estimated"
			} else {
				extracted.ReadingType = "actual"
			}
		case "estimated_date":
			parsed := h.parseDate(value, rule.Format)
			extracted.EstimatedDate = &parsed
		case "estimated_reading":
			val := h.parseAmount(value)
			extracted.EstimatedReading = &val
		case "estimated_consumption":
			val := h.parseAmount(value)
			extracted.EstimatedConsumption = &val
		case "previous_estimated_consumption":
			val := h.parseAmount(value)
			extracted.PreviousEstimatedConsumption = &val
		}
	}

	return extracted
}

// extractValueWithRule extracts a value using the rule's pattern(s)
// It uses position-based matching when coordinates are available
func (h *PDFHandler) extractValueWithRule(rule BillTemplateRule, text string) string {
	return h.extractValueWithRuleAndWords(rule, text, nil)
}

// extractValueWithRuleAndWords extracts a value using the best available strategy.
// Priority: 1) Global search (POD/PDR) 2) Anchor-based 3) Position-based 4) Pattern-based
func (h *PDFHandler) extractValueWithRuleAndWords(rule BillTemplateRule, text string, words []WordInfo) string {
	// 1. Global search for unique fields (POD/PDR, customer code)
	if rule.GlobalSearch && rule.ValuePattern != "" && len(words) > 0 {
		value := h.extractByGlobalSearch(rule, words)
		if value != "" {
			log.Printf("Field %s matched by global search: %s", rule.Field, value)
			return value
		}
		log.Printf("Field %s: global search failed, trying next strategy", rule.Field)
	}

	// 2. Anchor-based extraction (resilient to layout changes)
	if rule.AnchorText != "" && len(words) > 0 {
		value := h.extractByAnchor(rule, words)
		if value != "" {
			log.Printf("Field %s matched by anchor '%s': %s", rule.Field, rule.AnchorText, value)
			return value
		}
		log.Printf("Field %s: anchor-based match failed, trying position", rule.Field)
	}

	// 3. Position-based extraction (original coordinate matching)
	if rule.X > 0 && rule.Y > 0 && len(words) > 0 {
		value := h.extractByPosition(rule, words)
		if value != "" {
			log.Printf("Field %s matched by position: %s", rule.Field, value)
			return value
		}
		log.Printf("Field %s: position-based match failed, falling back to pattern", rule.Field)
	}

	// 4. Pattern-based extraction (regex on full text)
	return h.extractByPattern(rule, text)
}

// extractByPosition finds the value at the specified position with context validation
func (h *PDFHandler) extractByPosition(rule BillTemplateRule, words []WordInfo) string {
	// Filter words on the correct page
	pageWords := make([]WordInfo, 0)
	for _, w := range words {
		if w.Page == rule.Page {
			pageWords = append(pageWords, w)
		}
	}

	if len(pageWords) == 0 {
		return ""
	}

	// Find the word closest to the stored position
	var bestMatch WordInfo
	bestDistance := float64(999999)
	tolerance := 20.0 // Allow some tolerance for position matching (PDF units)

	for _, w := range pageWords {
		// Calculate distance from stored position
		dx := w.X - rule.X
		dy := w.Y - rule.Y
		distance := dx*dx + dy*dy // Squared distance (no need for sqrt)

		if distance < bestDistance && distance < tolerance*tolerance {
			bestDistance = distance
			bestMatch = w
		}
	}

	// If found by position, validate with context if available
	if bestMatch.Text != "" {
		// Validate context_left if provided
		if rule.ContextLeft != "" {
			leftContext := h.getContextLeft(bestMatch, pageWords)
			if !strings.Contains(strings.ToLower(leftContext), strings.ToLower(rule.ContextLeft)) {
				log.Printf("Context validation failed for %s: expected '%s' on left, got '%s'",
					rule.Field, rule.ContextLeft, leftContext)
				// Don't fail completely, just log warning
			}
		}

		// Validate context_above if provided
		if rule.ContextAbove != "" {
			aboveContext := h.getContextAbove(bestMatch, pageWords)
			if !strings.Contains(strings.ToLower(aboveContext), strings.ToLower(rule.ContextAbove)) {
				log.Printf("Context validation failed for %s: expected '%s' above, got '%s'",
					rule.Field, rule.ContextAbove, aboveContext)
			}
		}

		return bestMatch.Text
	}

	return ""
}

// getContextLeft returns words to the left of the target word (same line)
func (h *PDFHandler) getContextLeft(target WordInfo, words []WordInfo) string {
	var leftWords []string
	for _, w := range words {
		// Same line (within 5 units Y) and to the left
		if abs(w.Y-target.Y) < 5 && w.X < target.X {
			leftWords = append(leftWords, w.Text)
		}
	}
	// Sort by X position and return last 3 words
	sort.Slice(leftWords, func(i, j int) bool {
		return i < j // Keep original order
	})
	if len(leftWords) > 3 {
		leftWords = leftWords[len(leftWords)-3:]
	}
	return strings.Join(leftWords, " ")
}

// getContextAbove returns words above the target word (within same X range)
func (h *PDFHandler) getContextAbove(target WordInfo, words []WordInfo) string {
	var aboveWords []string
	for _, w := range words {
		// Above (smaller Y) and horizontally overlapping
		if w.Y < target.Y && w.Y > target.Y-50 {
			// Check horizontal overlap
			if w.X < target.X+target.Width && w.X+w.Width > target.X {
				aboveWords = append(aboveWords, w.Text)
			}
		}
	}
	return strings.Join(aboveWords, " ")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// AnchorMatch represents a found anchor text location in the document
type AnchorMatch struct {
	Words    []WordInfo // The words that form the anchor
	LastWord WordInfo   // Last word of the anchor (used as reference point)
	Page     int        // Page where anchor was found
}

// extractByAnchor finds the value relative to an anchor label in the document.
// It searches all pages for the anchor text, then looks for matching values
// in the specified direction (right, below, or both).
func (h *PDFHandler) extractByAnchor(rule BillTemplateRule, words []WordInfo) string {
	// Find all occurrences of the anchor text across all pages
	anchors := findAnchorOccurrences(rule.AnchorText, words)
	if len(anchors) == 0 {
		return ""
	}

	// Compile the value pattern
	if rule.ValuePattern == "" {
		return ""
	}
	valueRe, err := regexp.Compile("(?i)^" + rule.ValuePattern + "$")
	if err != nil {
		log.Printf("Invalid value pattern for anchor search field %s: %v", rule.Field, err)
		return ""
	}

	type candidate struct {
		text     string
		distance float64
		word     WordInfo
	}

	var bestCandidate *candidate

	for _, anchor := range anchors {
		// Search the region of interest based on direction
		roiWords := searchROI(anchor, rule.AnchorDirection, words)

		for _, w := range roiWords {
			// Check if this word matches the value pattern
			if valueRe.MatchString(w.Text) {
				// Calculate distance from anchor reference point
				dx := w.X - anchor.LastWord.X
				dy := w.Y - anchor.LastWord.Y
				dist := dx*dx + dy*dy

				// Historical position bonus: strongly prefer values near the
				// position where the user originally selected the value.
				// This disambiguates when the same anchor text appears
				// multiple times on the page (e.g., "Totale da pagare" in
				// both the bill summary table and a pie chart legend).
				if rule.X > 0 && rule.Y > 0 {
					histDx := w.X - rule.X
					histDy := w.Y - rule.Y
					histDist := histDx*histDx + histDy*histDy
					if histDist < 30*30 {
						dist *= 0.01 // Near-exact match to original position
					} else if histDist < 80*80 {
						dist *= 0.05 // Very close to original position
					} else if histDist < 150*150 {
						dist *= 0.2 // Reasonably close to original position
					}
					log.Printf("  Field %s: candidate '%s' at (%.1f,%.1f) histDist=%.0f anchorDist=%.0f finalScore=%.1f",
						rule.Field, w.Text, w.X, w.Y, histDist, dx*dx+dy*dy, dist)
				}

				if bestCandidate == nil || dist < bestCandidate.distance {
					bestCandidate = &candidate{text: w.Text, distance: dist, word: w}
				}
			}
		}
	}

	if bestCandidate != nil {
		return bestCandidate.text
	}
	return ""
}

// findAnchorOccurrences finds all occurrences of an anchor text (multi-word sequence)
// across all pages. Case-insensitive matching with tolerance for whitespace.
func findAnchorOccurrences(anchorText string, words []WordInfo) []AnchorMatch {
	anchorWords := strings.Fields(strings.ToLower(anchorText))
	if len(anchorWords) == 0 {
		return nil
	}

	var matches []AnchorMatch

	for i := 0; i <= len(words)-len(anchorWords); i++ {
		matched := true
		var matchedWords []WordInfo

		for j, aw := range anchorWords {
			w := words[i+j]
			wText := strings.ToLower(w.Text)

			// Must be on the same page for consecutive words
			if j > 0 && w.Page != words[i].Page {
				matched = false
				break
			}

			// Must be on the same line (Y within 10 units) for consecutive words
			if j > 0 && abs(w.Y-words[i+j-1].Y) > 10 {
				matched = false
				break
			}

			// Case-insensitive comparison, strip trailing punctuation for flexibility
			wClean := strings.TrimRight(wText, ":.,;")
			awClean := strings.TrimRight(aw, ":.,;")
			if wClean != awClean {
				matched = false
				break
			}

			matchedWords = append(matchedWords, w)
		}

		if matched && len(matchedWords) == len(anchorWords) {
			matches = append(matches, AnchorMatch{
				Words:    matchedWords,
				LastWord: matchedWords[len(matchedWords)-1],
				Page:     matchedWords[0].Page,
			})
		}
	}

	return matches
}

// searchROI returns words in the Region of Interest relative to the anchor.
// direction: "right" = same line to the right, "below" = below anchor,
// "right_or_below" = both regions combined.
func searchROI(anchor AnchorMatch, direction string, allWords []WordInfo) []WordInfo {
	ref := anchor.LastWord
	var results []WordInfo

	if direction == "" {
		direction = "right_or_below"
	}

	if direction == "right" || direction == "right_or_below" {
		// Same line (Y within 10 units), to the right, within 300 units
		for _, w := range allWords {
			if w.Page != ref.Page {
				continue
			}
			if abs(w.Y-ref.Y) < 10 && w.X > ref.X+ref.Width-5 && w.X < ref.X+300 {
				results = append(results, w)
			}
		}
	}

	if direction == "below" || direction == "right_or_below" {
		// Below anchor (Y > anchor + 5), within 150 units vertical,
		// with horizontal overlap (anchor X range extended)
		anchorXMin := anchor.Words[0].X
		anchorXMax := ref.X + ref.Width
		for _, w := range allWords {
			if w.Page != ref.Page {
				continue
			}
			if w.Y > ref.Y+5 && w.Y < ref.Y+150 {
				// Check horizontal proximity: word should be within or near anchor's X range
				if w.X < anchorXMax+100 && w.X+w.Width > anchorXMin-50 {
					results = append(results, w)
				}
			}
		}
	}

	return results
}

// extractByGlobalSearch searches the entire document for a value matching the pattern.
// Used for unique identifiers like POD/PDR codes that appear once in the document.
func (h *PDFHandler) extractByGlobalSearch(rule BillTemplateRule, words []WordInfo) string {
	if rule.ValuePattern == "" {
		return ""
	}

	valueRe, err := regexp.Compile("(?i)^" + rule.ValuePattern + "$")
	if err != nil {
		log.Printf("Invalid value pattern for global search field %s: %v", rule.Field, err)
		return ""
	}

	for _, w := range words {
		if valueRe.MatchString(w.Text) {
			return w.Text
		}
	}

	return ""
}

// extractByPattern uses regex pattern matching (original method)
func (h *PDFHandler) extractByPattern(rule BillTemplateRule, text string) string {
	hasContext := rule.Pattern != "" && rule.Pattern != rule.ValuePattern

	// 1. If the pattern has context (different from value_pattern), try it first
	if hasContext {
		re, err := regexp.Compile("(?i)" + rule.Pattern)
		if err != nil {
			log.Printf("Warning: Invalid regex pattern for field %s: %v", rule.Field, err)
		} else {
			match := re.FindStringSubmatch(text)
			if len(match) > 1 {
				return strings.TrimSpace(match[1])
			}
		}
	}

	// 2. Auto-context from prefix/suffix: when pattern has no context (pattern == value_pattern),
	// build context from the stored prefix labels. This is tried BEFORE the bare pattern
	// because bare patterns match the first occurrence which is usually wrong.
	if rule.ValuePattern != "" && rule.Prefix != "" {
		contextPrefix := extractLabelContext(rule.Prefix)
		if contextPrefix != "" {
			contextPattern := contextPrefix + `[\s:]*` + rule.ValuePattern
			re, err := regexp.Compile("(?i)" + contextPattern)
			if err == nil {
				match := re.FindStringSubmatch(text)
				if len(match) > 1 {
					log.Printf("Field %s matched with auto-context '%s': %s", rule.Field, contextPrefix, match[1])
					return strings.TrimSpace(match[1])
				}
			}
			log.Printf("Field %s: auto-context '%s' did not match", rule.Field, contextPrefix)
		}
	}

	// 3. Bare pattern as last resort (no context — matches first occurrence in document)
	if !hasContext && rule.Pattern != "" {
		re, err := regexp.Compile("(?i)" + rule.Pattern)
		if err != nil {
			log.Printf("Warning: Invalid regex pattern for field %s: %v", rule.Field, err)
		} else {
			match := re.FindStringSubmatch(text)
			if len(match) > 1 {
				log.Printf("Field %s matched with bare pattern (no context): %s", rule.Field, match[1])
				return strings.TrimSpace(match[1])
			}
		}
	}

	// 4. Fallback: value_pattern alone (only if different from pattern)
	if rule.ValuePattern != "" && rule.ValuePattern != rule.Pattern {
		re, err := regexp.Compile("(?i)" + rule.ValuePattern)
		if err != nil {
			log.Printf("Warning: Invalid value pattern for field %s: %v", rule.Field, err)
		} else {
			match := re.FindStringSubmatch(text)
			if len(match) > 1 {
				log.Printf("Field %s matched with fallback value_pattern", rule.Field)
				return strings.TrimSpace(match[1])
			} else if len(match) > 0 {
				return strings.TrimSpace(match[0])
			}
		}
	}

	return ""
}

// extractLabelContext extracts label-like words from the prefix/suffix text.
// Returns the last sequence of alphabetic words (skipping numbers and symbols),
// which are most likely structural labels that remain constant across bills.
func extractLabelContext(prefixText string) string {
	words := strings.Fields(prefixText)
	// Walk backwards to find the last run of label-like words
	var labels []string
	for i := len(words) - 1; i >= 0; i-- {
		w := words[i]
		// A label word is mostly alphabetic (allow accented chars, parentheses)
		isLabel := true
		letterCount := 0
		for _, r := range w {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= 0x00C0 && r <= 0x024F) || r == '(' || r == ')' {
				letterCount++
			}
		}
		if letterCount == 0 || float64(letterCount)/float64(len([]rune(w))) < 0.5 {
			isLabel = false
		}
		if isLabel {
			labels = append([]string{regexp.QuoteMeta(w)}, labels...)
		} else if len(labels) > 0 {
			break // Stop at first non-label word once we've collected some
		}
	}
	if len(labels) == 0 {
		return ""
	}
	// Use at most 3 words for context
	if len(labels) > 3 {
		labels = labels[len(labels)-3:]
	}
	return strings.Join(labels, `\s+`)
}

// extractBillDataDefault uses default patterns for common Italian utilities
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

	// Period patterns - try Italian month formats first (more specific)
	// Format: "01 dicembre 2025 - 31 dicembre 2025"
	monthPeriodPattern := regexp.MustCompile(`(?i)(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})\s*[-–]\s*(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
	if match := monthPeriodPattern.FindStringSubmatch(text); len(match) > 6 {
		extracted.PeriodStart = h.parseItalianDate(match[1], strings.ToLower(match[2]), match[3])
		extracted.PeriodEnd = h.parseItalianDate(match[4], strings.ToLower(match[5]), match[6])
		log.Printf("Extracted period (Italian dash): %s - %s", extracted.PeriodStart, extracted.PeriodEnd)
	}

	// Format: "dall'1 novembre 2024 al 22 dicembre 2024" (possibly across lines)
	if extracted.PeriodStart == "" {
		dallAlPattern := regexp.MustCompile(`(?i)dall['']?\s*(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})\s+al\s+(\d{1,2})\s+(gennaio|febbraio|marzo|aprile|maggio|giugno|luglio|agosto|settembre|ottobre|novembre|dicembre)\s+(\d{4})`)
		if match := dallAlPattern.FindStringSubmatch(text); len(match) > 6 {
			extracted.PeriodStart = h.parseItalianDate(match[1], strings.ToLower(match[2]), match[3])
			extracted.PeriodEnd = h.parseItalianDate(match[4], strings.ToLower(match[5]), match[6])
			log.Printf("Extracted period (dall/al): %s - %s", extracted.PeriodStart, extracted.PeriodEnd)
		}
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

	// Extract reading period from "Da DD/MM/YYYY a DD/MM/YYYY" pattern
	readingPeriodPattern := regexp.MustCompile(`Da\s+(\d{1,2}/\d{1,2}/\d{4})\s*[^\d]*a\s+(\d{1,2}/\d{1,2}/\d{4})`)
	if match := readingPeriodPattern.FindStringSubmatch(text); len(match) > 2 {
		// Use the end date of the reading period as the provider reading date
		extracted.ProviderReadingDate = h.parseDate(match[2], "")
		log.Printf("Extracted reading period: %s - %s", match[1], match[2])
	}

	switch utilityType {
	case "electricity":
		// Electricity readings pattern: final_reading - initial_reading = consumption
		// Repeated for F1, F2, F3 bands. We extract the FINAL readings.

		log.Printf("Extracting electricity readings from text (length: %d)", len(text))

		// Try to find the "Letture e consumi" section first
		lettureSection := text
		if idx := strings.Index(strings.ToLower(text), "letture e consumi"); idx != -1 {
			lettureSection = text[idx:]
			// Limit to next major section (roughly 2000 chars should cover the readings table)
			if len(lettureSection) > 2000 {
				lettureSection = lettureSection[:2000]
			}
			log.Printf("Found 'Letture e consumi' section at index %d", idx)
		}

		// Pattern for "final - initial = consumption" format
		// Matches: 1.936,56 - 1.899,18 = 37,38
		electricPattern := regexp.MustCompile(`(\d{1,3}(?:[.\s]\d{3})*,\d+)\s*-\s*(\d{1,3}(?:[.\s]\d{3})*,\d+)\s*=\s*(\d{1,3}(?:[.\s]\d{3})*,\d+)`)
		matches := electricPattern.FindAllStringSubmatch(lettureSection, -1)

		log.Printf("Found %d reading matches (pattern 1) in electricity bill", len(matches))

		if len(matches) >= 3 {
			// F1, F2, F3 final readings (first value in each match is the final reading)
			f1 := h.parseAmount(matches[0][1])
			f2 := h.parseAmount(matches[1][1])
			f3 := h.parseAmount(matches[2][1])
			extracted.ProviderReadingF1 = &f1
			extracted.ProviderReadingF2 = &f2
			extracted.ProviderReadingF3 = &f3
			log.Printf("Extracted electricity readings (pattern 1): F1=%.2f, F2=%.2f, F3=%.2f", f1, f2, f3)
		} else {
			// Fallback: Try simpler pattern without thousand separators
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

		// If still not found, try looking for values near FASCIA labels
		if extracted.ProviderReadingF1 == nil {
			log.Printf("Trying FASCIA label patterns...")
			// Look for patterns like "FASCIA 1" followed by numbers in table format
			// The table might have: Lettura finale | Lettura iniziale | Consumo
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
			// Log a sample of the text for debugging
			sample := lettureSection
			if len(sample) > 500 {
				sample = sample[:500]
			}
			log.Printf("Sample text: %s", sample)
		}

	case "gas":
		// Scope to readings section for more accurate extraction
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

		// Pattern: final_reading - initial_reading = (consumption may be on next line)
		gasPattern := regexp.MustCompile(`(\d[\d.,]+)\s*-\s*(\d[\d.,]+)\s*=`)
		matches := gasPattern.FindAllStringSubmatch(gasReadingsSection, -1)

		log.Printf("Found %d reading matches in gas bill", len(matches))

		if len(matches) > 0 {
			// Use last match: most recent reading period's final reading
			lastMatch := matches[len(matches)-1]
			reading := h.parseAmount(lastMatch[1])
			extracted.ProviderReading = &reading
			log.Printf("Extracted gas reading: %.2f mc (last of %d matches)", reading, len(matches))
		}

	case "water":
		// Water readings - look for "Lettura attuale" or similar
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

// getTemplateFieldSet returns the set of field names that a template defines rules for.
func (h *PDFHandler) getTemplateFieldSet(template models.BillTemplate) map[string]bool {
	fields := make(map[string]bool)
	var rules []BillTemplateRule
	json.Unmarshal([]byte(template.ExtractionRules), &rules)
	for _, rule := range rules {
		fields[rule.Field] = true
	}
	fieldNames := make([]string, 0, len(fields))
	for f := range fields {
		fieldNames = append(fieldNames, f)
	}
	log.Printf("Template '%s' defines rules for %d fields: %v", template.Name, len(fields), fieldNames)
	return fields
}

// mergeExtracted fills missing fields in primary with values from fallback.
// templateFields contains field names the template defines rules for - these are
// NOT overridden by default extraction even if empty, because wrong values are
// worse than missing values. The user can always fill them in manually.
func (h *PDFHandler) mergeExtracted(primary, fallback ExtractedBillData, templateFields map[string]bool) ExtractedBillData {
	if primary.BillNumber == "" && !templateFields["bill_number"] {
		primary.BillNumber = fallback.BillNumber
	}
	if primary.IssueDate == "" && !templateFields["issue_date"] {
		primary.IssueDate = fallback.IssueDate
	}
	if primary.PeriodStart == "" && !templateFields["period_start"] {
		primary.PeriodStart = fallback.PeriodStart
	}
	if primary.PeriodEnd == "" && !templateFields["period_end"] {
		primary.PeriodEnd = fallback.PeriodEnd
	}
	if primary.DueDate == "" && !templateFields["due_date"] {
		primary.DueDate = fallback.DueDate
	}
	if primary.AmountTotal == 0 && !templateFields["amount_total"] {
		primary.AmountTotal = fallback.AmountTotal
	}
	if primary.ConsumptionTotal == 0 && !templateFields["consumption_total"] {
		primary.ConsumptionTotal = fallback.ConsumptionTotal
	}
	// Provider is always merged (not a template field, detected by keyword)
	if primary.Provider == "" {
		primary.Provider = fallback.Provider
	}
	if primary.ServiceCode == "" && !templateFields["service_code"] {
		primary.ServiceCode = fallback.ServiceCode
	}
	if primary.CustomerCode == "" && !templateFields["customer_code"] {
		primary.CustomerCode = fallback.CustomerCode
	}
	if primary.ProviderReadingDate == "" && !templateFields["provider_reading_date"] {
		primary.ProviderReadingDate = fallback.ProviderReadingDate
	}
	if primary.ReadingType == "" && !templateFields["reading_type"] {
		primary.ReadingType = fallback.ReadingType
	}
	if primary.ProviderReadingF1 == nil && !templateFields["provider_reading_f1"] {
		primary.ProviderReadingF1 = fallback.ProviderReadingF1
	}
	if primary.ProviderReadingF2 == nil && !templateFields["provider_reading_f2"] {
		primary.ProviderReadingF2 = fallback.ProviderReadingF2
	}
	if primary.ProviderReadingF3 == nil && !templateFields["provider_reading_f3"] {
		primary.ProviderReadingF3 = fallback.ProviderReadingF3
	}
	if primary.ProviderReading == nil && !templateFields["provider_reading"] {
		primary.ProviderReading = fallback.ProviderReading
	}
	if primary.ConversionCoefficient == nil && !templateFields["conversion_coefficient"] {
		primary.ConversionCoefficient = fallback.ConversionCoefficient
	}
	if primary.EstimatedDate == nil && !templateFields["estimated_date"] {
		primary.EstimatedDate = fallback.EstimatedDate
	}
	if primary.EstimatedReading == nil && !templateFields["estimated_reading"] {
		primary.EstimatedReading = fallback.EstimatedReading
	}
	if primary.EstimatedConsumption == nil && !templateFields["estimated_consumption"] {
		primary.EstimatedConsumption = fallback.EstimatedConsumption
	}
	if primary.PreviousEstimatedConsumption == nil && !templateFields["previous_estimated_consumption"] {
		primary.PreviousEstimatedConsumption = fallback.PreviousEstimatedConsumption
	}
	return primary
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
		Name            string             `json:"name" binding:"required"`
		Provider        string             `json:"provider" binding:"required"`
		UtilityType     string             `json:"utility_type" binding:"required,oneof=electricity gas water waste"`
		IsDefault       bool               `json:"is_default"`
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
		Name            string             `json:"name"`
		Provider        string             `json:"provider"`
		UtilityType     string             `json:"utility_type"`
		IsDefault       bool               `json:"is_default"`
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

	result := h.db.Unscoped().Where("id = ? AND user_id = ?", templateID, userID).Delete(&models.BillTemplate{})
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

// PDFPageImage represents a page converted to image with word positions
type PDFPageImage struct {
	PageNumber  int        `json:"page_number"`
	ImageURL    string     `json:"image_url"`
	ImageWidth  int        `json:"image_width"`
	ImageHeight int        `json:"image_height"`
	Words       []WordInfo `json:"words"`
}

// PDFAnalysisResult contains the full PDF analysis for template wizard
type PDFAnalysisResult struct {
	Pages     []PDFPageImage `json:"pages"`
	PageCount int            `json:"page_count"`
	RawText   string         `json:"raw_text"`
}

// AnalyzePDFForTemplate - POST /api/v1/pdf/analyze
// Converts PDF to images and extracts word positions for Textract-like UI
func (h *PDFHandler) AnalyzePDFForTemplate(c *gin.Context) {
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

	// Save PDF temporarily
	timestamp := time.Now().UnixNano()
	pdfFile := filepath.Join(h.uploadsDir, fmt.Sprintf("analyze_%d.pdf", timestamp))
	if err := c.SaveUploadedFile(file, pdfFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer os.Remove(pdfFile)

	// Convert PDF pages to images using pdftoppm
	imagePrefix := filepath.Join(h.uploadsDir, fmt.Sprintf("page_%d", timestamp))
	cmd := exec.Command("pdftoppm", "-png", "-r", "150", pdfFile, imagePrefix)
	if err := cmd.Run(); err != nil {
		log.Printf("pdftoppm failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert PDF to images. Make sure poppler-utils is installed."})
		return
	}

	// Find generated images
	pattern := fmt.Sprintf("page_%d-*.png", timestamp)
	matches, _ := filepath.Glob(filepath.Join(h.uploadsDir, pattern))
	if len(matches) == 0 {
		// Try single page format (no suffix for single page PDFs)
		singlePage := imagePrefix + ".png"
		if _, err := os.Stat(singlePage); err == nil {
			matches = []string{singlePage}
		}
	}

	if len(matches) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No images generated from PDF"})
		return
	}

	// Sort matches to ensure correct page order
	sort.Strings(matches)

	// Extract text with bbox for word positions
	extracted, err := h.extractTextWithPositions(pdfFile)
	if err != nil {
		log.Printf("Text extraction warning: %v", err)
	}

	// Build result
	result := PDFAnalysisResult{
		Pages:     make([]PDFPageImage, 0, len(matches)),
		PageCount: len(matches),
		RawText:   "",
	}
	if extracted != nil {
		result.RawText = extracted.RawText
	}

	// Process each page
	for i, imgPath := range matches {
		// Get image dimensions
		width, height := getImageDimensions(imgPath)

		// Move image to permanent location with predictable name
		newName := fmt.Sprintf("template_page_%d_%d.png", timestamp, i+1)
		newPath := filepath.Join(h.uploadsDir, newName)
		os.Rename(imgPath, newPath)

		// Filter words for this page
		pageWords := []WordInfo{}
		if extracted != nil && extracted.HasBBox {
			for _, w := range extracted.Words {
				if w.Page == i {
					pageWords = append(pageWords, w)
				}
			}
		}

		result.Pages = append(result.Pages, PDFPageImage{
			PageNumber:  i + 1,
			ImageURL:    "/uploads/" + newName,
			ImageWidth:  width,
			ImageHeight: height,
			Words:       pageWords,
		})
	}

	// If no bbox data, distribute words across pages based on line index
	if extracted != nil && !extracted.HasBBox && len(extracted.Words) > 0 {
		wordsPerPage := len(extracted.Words) / len(result.Pages)
		if wordsPerPage == 0 {
			wordsPerPage = len(extracted.Words)
		}

		for i := range result.Pages {
			start := i * wordsPerPage
			end := start + wordsPerPage
			if i == len(result.Pages)-1 {
				end = len(extracted.Words)
			}
			if start < len(extracted.Words) {
				if end > len(extracted.Words) {
					end = len(extracted.Words)
				}
				result.Pages[i].Words = extracted.Words[start:end]
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// getImageDimensions reads PNG dimensions from file header
func getImageDimensions(path string) (int, int) {
	file, err := os.Open(path)
	if err != nil {
		return 800, 1100 // Default A4-ish dimensions
	}
	defer file.Close()

	// Read PNG header (8 bytes magic + IHDR chunk)
	header := make([]byte, 24)
	if _, err := file.Read(header); err != nil {
		return 800, 1100
	}

	// PNG dimensions are at bytes 16-23 in big-endian
	if len(header) >= 24 {
		width := int(header[16])<<24 | int(header[17])<<16 | int(header[18])<<8 | int(header[19])
		height := int(header[20])<<24 | int(header[21])<<16 | int(header[22])<<8 | int(header[23])
		if width > 0 && height > 0 {
			return width, height
		}
	}

	return 800, 1100
}

// CleanupTemplateImages - DELETE /api/v1/pdf/cleanup/:timestamp
// Cleans up temporary template images
func (h *PDFHandler) CleanupTemplateImages(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	timestamp := c.Param("timestamp")
	pattern := filepath.Join(h.uploadsDir, fmt.Sprintf("template_page_%s_*.png", timestamp))
	matches, _ := filepath.Glob(pattern)

	for _, match := range matches {
		os.Remove(match)
	}

	c.JSON(http.StatusOK, gin.H{"deleted": len(matches)})
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

	// Check if caller wants positions (for template wizard)
	withPositions := c.Query("with_positions") == "true" || c.PostForm("with_positions") == "true"

	if withPositions {
		result, err := h.extractTextWithPositions(tempFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract text: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// Standard extraction without positions
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
