package handlers

import (
	"fmt"
	"testing"

	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// The template wizard shows, for a template being edited, what each rule
// extracts from the uploaded PDF — and only for a template of the caller's household.
func TestTemplateValuesForAnalysis(t *testing.T) {
	db := testutil.NewDB(t)
	h := &PDFHandler{db: db}

	owner := models.User{Email: "owner@example.com", PasswordHash: "x", Name: "Owner", IsActive: true}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	prop := models.Property{UserID: owner.ID, Name: "Casa", IsCurrent: true}
	if err := db.Create(&prop).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.HouseholdMember{PropertyID: prop.ID, UserID: &owner.ID, Name: "Owner"}).Error; err != nil {
		t.Fatal(err)
	}
	tpl := models.BillTemplate{
		PropertyID: prop.ID, Name: "Etra", Provider: "Etra", UtilityType: "water",
		ExtractionRules: `[{"field":"consumption_total","anchor_text":"Consumo fatturato","anchor_direction":"right","value_pattern":"(\\d{1,3}(?:[.\\s]\\d{3})*(?:,\\d+)?)"},
			{"field":"bill_number","pattern":"Bolletta\\s+n\\.\\s+(\\d+)"},
			{"field":"due_date","anchor_text":"Scadenza","value_pattern":"(\\d{4}-\\d{2}-\\d{2})"}]`,
	}
	if err := db.Create(&tpl).Error; err != nil {
		t.Fatal(err)
	}

	pdf := &ExtractedTextWithPositions{
		RawText: "Bolletta n. 1079599\nConsumo fatturato dal 29/05/2026 al 07/10/2026   27 mc\n",
		Words: []WordInfo{
			{Text: "Consumo", X: 100, Y: 400, Width: 30},
			{Text: "fatturato", X: 132, Y: 400, Width: 30},
			{Text: "27", X: 300, Y: 400, Width: 8},
			{Text: "mc", X: 310, Y: 400, Width: 8},
		},
	}

	got := h.templateValuesForAnalysis(fmt.Sprint(tpl.ID), owner.ID, pdf)
	want := map[string]string{"consumption_total": "27", "bill_number": "1079599"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for field, value := range want {
		if got[field] != value {
			t.Errorf("%s = %q, want %q", field, got[field], value)
		}
	}

	if got := h.templateValuesForAnalysis(fmt.Sprint(tpl.ID), owner.ID+1, pdf); got != nil {
		t.Errorf("another household's template leaked values: %v", got)
	}
	if got := h.templateValuesForAnalysis("", owner.ID, pdf); got != nil {
		t.Errorf("no template requested, got %v", got)
	}
}
