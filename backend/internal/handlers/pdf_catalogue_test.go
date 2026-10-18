package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// A field the wizard lets the user map but the extractor ignores is a silent
// gap: the template saves, the bill upload leaves the field empty.
func TestEveryCatalogueFieldIsExtracted(t *testing.T) {
	h := &PDFHandler{}
	for _, set := range models.BillFieldCatalogue {
		for _, field := range set.Fields {
			text := "Valore: 123,45"
			if strings.HasSuffix(field.Key, "_date") {
				text = "Valore: 12/03/2026"
			}
			rules, _ := json.Marshal([]BillTemplateRule{{Field: field.Key, Pattern: `Valore:\s*(\S+)`}})
			template := models.BillTemplate{UtilityType: set.Type, ExtractionRules: string(rules)}

			got := h.extractBillDataWithTemplateAndWords(text, template, "", nil)
			if reflect.DeepEqual(got, ExtractedBillData{RawText: text}) {
				t.Errorf("%s/%s: no extraction case fills it", set.Type, field.Key)
			}
		}
	}
}

func TestBillTemplateRulesMustBeInTheCatalogue(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	user := models.User{Email: "owner@example.com", PasswordHash: "x", Name: "Owner", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	home := models.Property{UserID: user.ID, Name: "Home", IsCurrent: true}
	if err := db.Create(&home).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.HouseholdMember{PropertyID: home.ID, UserID: &user.ID, Name: "Owner", Role: "admin"}).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(middleware.AuthRequired())
	h := NewPDFHandler(db)
	g.POST("/templates/bills", h.CreateBillTemplate)
	g.PUT("/templates/bills/:id", h.UpdateBillTemplate)
	token := testutil.SignToken(t, &user)

	create := func(utilityType, field string) *httptest.ResponseRecorder {
		return doJSON(t, r, http.MethodPost, "/templates/bills", token, map[string]any{
			"property_id": home.ID, "name": "T", "provider": "P", "utility_type": utilityType,
			"extraction_rules": []map[string]any{{"field": field, "pattern": `(\d+)`}},
		})
	}

	if rec := create("electricity", "provider_reading_f1"); rec.Code != http.StatusCreated {
		t.Fatalf("catalogue field rejected: %d %s", rec.Code, rec.Body.String())
	}
	for _, tc := range []struct{ utilityType, field, code string }{
		{"internet", "provider_reading", "template_field_not_supported"},
		{"water", "provider_reading_f1", "template_field_not_supported"},
		{"affitto", "amount_total", "invalid_request"},
	} {
		rec := create(tc.utilityType, tc.field)
		var body struct {
			Code string `json:"error_code"`
		}
		json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != http.StatusBadRequest || body.Code != tc.code {
			t.Errorf("%s/%s: got %d %q, want 400 %q", tc.utilityType, tc.field, rec.Code, body.Code, tc.code)
		}
	}

	var tpl models.BillTemplate
	db.First(&tpl)
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/templates/bills/%d", tpl.ID), token, map[string]any{"utility_type": "gas"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("switching to gas kept an electricity-only rule: %d %s", rec.Code, rec.Body.String())
	}
}
