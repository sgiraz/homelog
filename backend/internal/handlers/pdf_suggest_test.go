package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

func readingsPage() []WordInfo {
	return page(
		line(10, 40, "Numero Fattura:"), line(10, 52, "2026 / 249743"),
		line(10, 70, "Matricola contatore:"), line(120, 70, "3452327545"),
		line(10, 100, "Tipo lettura"), line(100, 100, "Data lettura"), line(200, 100, "Lettura"),
		line(10, 112, "LG - Rilevata"), line(100, 112, "03/01/2026"), line(200, 112, "89"),
		line(10, 124, "LG - Autolettura"), line(100, 124, "28/02/2026"), line(200, 124, "102"),
		line(10, 136, "MA - Stimata"), line(100, 136, "16/06/2026"), line(200, 136, "122"),
		line(10, 160, "Consumi del periodo"),
	)
}

func pick(t *testing.T, words []WordInfo, text string) WordInfo {
	t.Helper()
	for _, w := range words {
		if w.Text == text {
			return w
		}
	}
	t.Fatalf("no word %q", text)
	return WordInfo{}
}

func TestSuggestRules(t *testing.T) {
	h := &PDFHandler{}
	words := readingsPage()

	// Like values stacked above: a table column, the last cell.
	got, _ := h.suggestRules("provider_reading", pick(t, words, "122"), words)
	if len(got) == 0 || got[0].Rule.Table == nil || got[0].Rule.Table.Column != "Lettura" || got[0].Rule.Table.Cell != "last" || got[0].Value != "122" {
		t.Fatalf("reading suggestion = %+v, want the last cell of column Lettura", got)
	}

	// A label to the left: an anchor.
	got, _ = h.suggestRules("meter_serial", pick(t, words, "3452327545"), words)
	if len(got) == 0 || got[0].Rule.AnchorText != "Matricola contatore:" || got[0].Rule.AnchorDirection != "right" {
		t.Fatalf("serial suggestion = %+v, want the label on its left", got)
	}

	// Only its cell singles the value out: the year before it is in the pattern.
	got, _ = h.suggestRules("bill_number", pick(t, words, "249743"), words)
	if len(got) == 0 || got[0].Value != "249743" {
		t.Fatalf("bill number suggestion = %+v, want 249743", got)
	}

	// The reading type keeps the whole cell, for the estimate words.
	got, cell := h.suggestRules("reading_type", pick(t, words, "Stimata"), words)
	if len(got) == 0 || cell != "MA - Stimata" {
		t.Fatalf("reading type suggestion = %+v (cell %q), want the type column", got, cell)
	}

	// Every suggestion reads the picked value back.
	for _, s := range got {
		if v := h.extractValueWithRuleAndWords(s.Rule, "", words); normalizeLabel(v) != normalizeLabel(s.Value) {
			t.Errorf("rule %+v extracts %q, suggested for %q", s.Rule, v, s.Value)
		}
	}
}

// The analysed bill's words belong to whoever uploaded it.
func TestSuggestBillRuleIsPerUser(t *testing.T) {
	f := setupMoneyFixture(t)
	h := &PDFHandler{db: f.db, uploadsDir: t.TempDir()}
	raw, _ := json.Marshal(readingsPage())
	if err := os.WriteFile(h.analysisWordsPath("123_abcd", f.alice.ID), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.AuthRequired())
	r.POST("/pdf/suggest-rule", h.SuggestBillRule)
	body := map[string]any{"tag": "123_abcd", "field": "provider_reading", "word": pick(t, readingsPage(), "122")}

	rec := doJSON(t, r, http.MethodPost, "/pdf/suggest-rule", f.aliceTok, body)
	var out struct {
		Suggestions []RuleSuggestion `json:"suggestions"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &out) != nil || len(out.Suggestions) == 0 {
		t.Fatalf("owner: status %d, body %s", rec.Code, rec.Body.String())
	}

	eve := &models.User{Email: "eve@example.com", PasswordHash: "x", Name: "Eve", Role: "user", IsActive: true}
	mustCreate(t, f.db, eve)
	if rec := doJSON(t, r, http.MethodPost, "/pdf/suggest-rule", testutil.SignToken(t, eve), body); rec.Code != http.StatusNotFound {
		t.Errorf("another user got status %d, want 404", rec.Code)
	}
}

// A bill that misses most of a template's labels warns, and names the home's
// template that reads it.
func TestLayoutMismatch(t *testing.T) {
	f := setupMoneyFixture(t)
	h := &PDFHandler{db: f.db}
	rules := func(anchors ...string) string {
		var rs []BillTemplateRule
		for i, a := range anchors {
			rs = append(rs, BillTemplateRule{Field: []string{"bill_number", "amount_total"}[i], AnchorText: a, AnchorDirection: "right", ValuePattern: `(\d+(?:,\d{2})?)`})
		}
		raw, _ := json.Marshal(rs)
		return string(raw)
	}
	old := models.BillTemplate{PropertyID: f.prop.ID, Name: "Old", Provider: "Eon", UtilityType: "gas", ExtractionRules: rules("Bolletta", "Importo")}
	current := models.BillTemplate{PropertyID: f.prop.ID, Name: "New", Provider: "Eon", UtilityType: "gas", ExtractionRules: rules("n°", "Totale")}
	mustCreate(t, f.db, &old)
	mustCreate(t, f.db, &current)
	words := page(line(10, 10, "n° 98142292"), line(10, 30, "Totale"), line(100, 30, "29,00"))

	extracted := h.extractBillDataWithTemplateAndWords("", old, "", words)
	extracted.NotFound = notFoundFields(extracted, old)
	mismatch, suggested := h.layoutMismatch(extracted, old, "", words)
	if !mismatch || suggested == nil || *suggested != current.ID {
		t.Errorf("mismatch %v, suggested %v; want a warning naming template %d", mismatch, suggested, current.ID)
	}

	extracted = h.extractBillDataWithTemplateAndWords("", current, "", words)
	extracted.NotFound = notFoundFields(extracted, current)
	if mismatch, _ := h.layoutMismatch(extracted, current, "", words); mismatch {
		t.Error("the matching template must not warn")
	}
}

// A request the wizard cannot have sent, or an analysis already cleaned up,
// is refused with a code the client can word.
func TestSuggestAndTryRejectBadInput(t *testing.T) {
	f := setupMoneyFixture(t)
	h := &PDFHandler{db: f.db, uploadsDir: t.TempDir()}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.AuthRequired())
	r.POST("/pdf/suggest-rule", h.SuggestBillRule)
	r.POST("/pdf/try-template", h.TryBillTemplate)

	word := pick(t, readingsPage(), "122")
	for name, tc := range map[string]struct {
		body any
		code string
	}{
		"no field":       {map[string]any{"tag": "123_abcd", "word": word}, "invalid_request"},
		"unknown tag":    {map[string]any{"tag": "999_ffff", "field": "provider_reading", "word": word}, "pdf_analysis_expired"},
		"not a tag path": {map[string]any{"tag": "../etc", "field": "provider_reading", "word": word}, "pdf_analysis_expired"},
	} {
		rec := doJSON(t, r, http.MethodPost, "/pdf/suggest-rule", f.aliceTok, tc.body)
		var out struct {
			Code string `json:"error_code"`
		}
		json.Unmarshal(rec.Body.Bytes(), &out)
		if out.Code != tc.code {
			t.Errorf("%s: status %d, code %q, want %q", name, rec.Code, out.Code, tc.code)
		}
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, _ := form.CreateFormFile("pdf_file", "bill.pdf")
	part.Write([]byte("%PDF-1.4 not really"))
	form.WriteField("extraction_rules", "not json")
	form.Close()
	req := httptest.NewRequest(http.MethodPost, "/pdf/try-template", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+f.aliceTok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_request") {
		t.Errorf("rules that are not JSON: status %d, body %s", rec.Code, rec.Body.String())
	}
}
