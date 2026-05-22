package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// Maximum accepted size for an uploaded PDF (bill or contract). Keeping this
// modest matches the Raspberry Pi 3B+ memory constraint and slows storage-fill
// attacks against the public auth'd endpoints.
const maxPDFUploadSize = 10 << 20 // 10 MiB

// pdfProcessTimeout bounds how long pdftotext/pdftoppm may run before being
// killed. Without this, a crafted PDF could hang the process indefinitely.
// Generous enough for slow ARM hardware (Raspberry Pi 3B+) to rasterize a
// multi-page utility bill at renderDPI without hitting a SIGKILL.
const pdfProcessTimeout = 60 * time.Second

// renderDPI is the resolution pdftoppm uses to rasterize PDF pages for the
// template wizard preview. Kept modest so rendering stays within the Pi 3B+
// time budget (see pdfProcessTimeout). The frontend reads it back via
// PDFPageImage.RenderDPI to scale word overlays onto the image, so the
// backend render resolution and the frontend coordinate scale never drift.
const renderDPI = 100

// randomSuffix returns n bytes of cryptographic randomness as hex (2n chars).
// Used to make uploaded file paths unguessable so /uploads/<name> cannot be
// enumerated via predictable (utilityID, timestamp) tuples.
func randomSuffix(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read only fails on catastrophic OS failure; fall back to
		// the nanosecond timestamp — still collision-resistant within one host.
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

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
	MultiLine       bool   `json:"multi_line"`       // For text blocks: collect all text below anchor until next section
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

	// Communication text (extracted from bill)
	CommunicationText string `json:"communication_text,omitempty"`
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

	// Reject oversized uploads before writing to disk.
	if file.Size > maxPDFUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "PDF troppo grande (max 10 MB)"})
		return
	}

	// Save the file with an unguessable suffix so /uploads/<name> cannot be
	// enumerated by guessing (utilityID, timestamp).
	filename := fmt.Sprintf("bill_%d_%d_%s.pdf", utilityID, time.Now().Unix(), randomSuffix(16))
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

	if file.Size > maxPDFUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "PDF troppo grande (max 10 MB)"})
		return
	}

	// Save the file with an unguessable suffix.
	filename := fmt.Sprintf("contract_%d_%d_%s.pdf", userID, time.Now().Unix(), randomSuffix(16))
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
		UtilityType     string             `json:"utility_type" binding:"required,oneof=electricity gas water waste internet insurance affitto mutuo"`
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
	PageNumber  int    `json:"page_number"`
	ImageURL    string `json:"image_url"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	// RenderDPI echoes the resolution pdftoppm used so the frontend can scale
	// the 72-DPI bbox word coordinates onto the rendered image without
	// hardcoding (and drifting from) the backend value.
	RenderDPI int        `json:"render_dpi"`
	Words     []WordInfo `json:"words"`
}

// PDFAnalysisResult contains the full PDF analysis for template wizard.
// Tag is an opaque identifier (timestamp + random suffix) used by the frontend
// to request cleanup of the generated preview images.
type PDFAnalysisResult struct {
	Tag       string         `json:"tag"`
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

	if file.Size > maxPDFUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "PDF troppo grande (max 10 MB)"})
		return
	}

	// Save PDF temporarily. The "timestamp" here doubles as the tag passed to
	// CleanupTemplateImages, so we keep it as part of the filename and append
	// a random suffix for unguessability.
	timestamp := time.Now().UnixNano()
	suffix := randomSuffix(8)
	tag := fmt.Sprintf("%d_%s", timestamp, suffix)
	pdfFile := filepath.Join(h.uploadsDir, fmt.Sprintf("analyze_%s.pdf", tag))
	if err := c.SaveUploadedFile(file, pdfFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer os.Remove(pdfFile)

	// Convert PDF pages to images using pdftoppm (bounded runtime).
	imagePrefix := filepath.Join(h.uploadsDir, fmt.Sprintf("page_%s", tag))
	ppmCtx, cancelPpm := context.WithTimeout(c.Request.Context(), pdfProcessTimeout)
	defer cancelPpm()
	cmd := exec.CommandContext(ppmCtx, "pdftoppm", "-png", "-r", fmt.Sprintf("%d", renderDPI), pdfFile, imagePrefix)
	var ppmStderr bytes.Buffer
	cmd.Stderr = &ppmStderr
	if err := cmd.Run(); err != nil {
		log.Printf("pdftoppm failed: %v (stderr: %s)", err, strings.TrimSpace(ppmStderr.String()))
		// Distinguish the failure modes so the client gets an actionable message
		// instead of a blanket "install poppler-utils" (which misleads when the
		// real cause is a timeout on slow hardware or an unreadable PDF).
		switch {
		case ppmCtx.Err() == context.DeadlineExceeded:
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Conversione del PDF troppo lenta: il file è troppo complesso o ha troppe pagine. Riprova con un PDF più leggero."})
		case errors.Is(err, exec.ErrNotFound):
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Conversione PDF non disponibile sul server: poppler-utils non è installato."})
		default:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Impossibile convertire il PDF in immagini: il file potrebbe essere corrotto o non supportato."})
		}
		return
	}

	// Find generated images
	pattern := fmt.Sprintf("page_%s-*.png", tag)
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
		Tag:       tag,
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

		// Move image to a location keyed by the (timestamp, suffix) tag — the
		// suffix keeps the URL unguessable across users.
		newName := fmt.Sprintf("template_page_%s_%d.png", tag, i+1)
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
			RenderDPI:   renderDPI,
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

// CleanupTemplateImages - DELETE /api/v1/pdf/cleanup/:timestamp
// Cleans up temporary template images produced by AnalyzePDFForTemplate.
// The :timestamp path segment is the opaque "tag" returned from analyze; it
// must match the expected shape (digits, an underscore, and a hex suffix)
// to prevent path-traversal or glob injection into the uploads dir.
var analysisTagRe = regexp.MustCompile(`^\d+(?:_[a-f0-9]+)?$`)

func (h *PDFHandler) CleanupTemplateImages(c *gin.Context) {
	_, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tag := c.Param("timestamp")
	if !analysisTagRe.MatchString(tag) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cleanup tag"})
		return
	}
	pattern := filepath.Join(h.uploadsDir, fmt.Sprintf("template_page_%s_*.png", tag))
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

	if file.Size > maxPDFUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "PDF troppo grande (max 10 MB)"})
		return
	}

	// Save temporarily with an unguessable name, even though the file is
	// deleted on return — a predictable name lets a concurrent attacker race
	// the defer and read another user's upload in the same window.
	tempFile := filepath.Join(h.uploadsDir, fmt.Sprintf("temp_%d_%s.pdf", time.Now().UnixNano(), randomSuffix(8)))
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
