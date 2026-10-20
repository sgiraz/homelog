package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// analysisWordsPath is where an analysis keeps the sample bill's words for the
// wizard's rule suggestions, per user: the tag alone does not grant access.
func (h *PDFHandler) analysisWordsPath(tag string, userID uint) string {
	return filepath.Join(h.uploadsDir, fmt.Sprintf("analyze_%s_u%d.json", tag, userID))
}

// SuggestBillRule - POST /api/v1/pdf/suggest-rule
// Proposes the rules that read the picked value back from the analysed bill.
// An empty list means no rule reproduces it: the wizard does not map it.
func (h *PDFHandler) SuggestBillRule(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var input struct {
		Tag   string   `json:"tag" binding:"required"`
		Field string   `json:"field" binding:"required"`
		Word  WordInfo `json:"word" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	raw, err := os.ReadFile(h.analysisWordsPath(input.Tag, userID))
	if !analysisTagRe.MatchString(input.Tag) || err != nil {
		apierr.Fail(c, http.StatusNotFound, "pdf_analysis_expired", "The analysed PDF is gone: upload it again")
		return
	}
	var words []WordInfo
	if err := json.Unmarshal(raw, &words); err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to read the analysed PDF")
		return
	}

	suggestions, cell := h.suggestRules(input.Field, input.Word, words)
	if suggestions == nil {
		suggestions = []RuleSuggestion{}
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions, "cell": cell})
}

// TryBillTemplate - POST /api/v1/pdf/try-template
// Runs unsaved rules on another bill, so the wizard can check a template on a
// second PDF before saving it. Nothing is stored.
func (h *PDFHandler) TryBillTemplate(c *gin.Context) {
	if _, exists := middleware.GetUserID(c); !exists {
		apierr.Fail(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	file, ok := receivePDFUpload(c)
	if !ok {
		return
	}
	var rules []BillTemplateRule
	if err := json.Unmarshal([]byte(c.PostForm("extraction_rules")), &rules); err != nil {
		apierr.Fail(c, http.StatusBadRequest, "invalid_request", "Invalid extraction rules")
		return
	}

	path := filepath.Join(h.uploadsDir, fmt.Sprintf("temp_try_%d_%s.pdf", time.Now().UnixNano(), randomSuffix(8)))
	if err := c.SaveUploadedFile(file, path); err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to save file")
		return
	}
	defer os.Remove(path)

	extracted, err := h.extractTextWithPositions(path)
	if err != nil {
		apierr.Fail(c, http.StatusUnprocessableEntity, "pdf_render_failed", "The text of this PDF could not be read")
		return
	}
	raw, _ := json.Marshal(rules)
	template := models.BillTemplate{ExtractionRules: string(raw)}
	result := h.extractBillDataWithTemplateAndWords(extracted.RawText, template, "", extracted.Words)
	result.RawText = ""
	result.NotFound = notFoundFields(result, template)
	c.JSON(http.StatusOK, result)
}

// layoutMismatch tells whether a bill does not look like the template's
// layout: half or more of the fields it maps were not found. It then names
// the property's template of the same type that reads the bill best, if any
// does better. The upload only warns: it never switches template by itself.
func (h *PDFHandler) layoutMismatch(extracted ExtractedBillData, template models.BillTemplate, text string, words []WordInfo) (bool, *uint) {
	mapped := len(templateFields(template))
	if mapped == 0 || 2*len(extracted.NotFound) < mapped {
		return false, nil
	}
	var others []models.BillTemplate
	h.db.Where("property_id = ? AND utility_type = ? AND id != ?", template.PropertyID, template.UtilityType, template.ID).Find(&others)

	var best *uint
	bestMissing := float64(len(extracted.NotFound)) / float64(mapped)
	for i := range others {
		n := len(templateFields(others[i]))
		if n == 0 {
			continue
		}
		missing := float64(len(notFoundFields(h.extractBillDataWithTemplateAndWords(text, others[i], "", words), others[i]))) / float64(n)
		if missing < bestMissing && 2*missing < 1 {
			best, bestMissing = &others[i].ID, missing
		}
	}
	return true, best
}

func templateFields(template models.BillTemplate) map[string]bool {
	var rules []BillTemplateRule
	json.Unmarshal([]byte(template.ExtractionRules), &rules)
	fields := map[string]bool{}
	for _, r := range rules {
		fields[r.Field] = true
	}
	return fields
}
