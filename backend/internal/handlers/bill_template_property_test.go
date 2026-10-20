package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

// templateFixture: a household of two (owner + partner) and a neighbour with
// a home of their own.
type templateFixture struct {
	db                         *gorm.DB
	router                     *gin.Engine
	h                          *PDFHandler
	home, away                 models.Property
	owner, partner, neighbour  models.User
	ownerTok, partnerTok, nTok string
}

func setupTemplateFixture(t *testing.T) *templateFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	t.Setenv("DB_PATH", t.TempDir()+"/homelog.db")
	db := testutil.NewDB(t)
	f := &templateFixture{db: db, h: NewPDFHandler(db)}

	mkUser := func(email string) models.User {
		u := models.User{Email: email, PasswordHash: "x", Name: email, Role: "user", IsActive: true}
		mustCreate(t, db, &u)
		return u
	}
	f.owner, f.partner, f.neighbour = mkUser("owner@example.com"), mkUser("partner@example.com"), mkUser("neighbour@example.com")
	f.home = models.Property{UserID: f.owner.ID, Name: "Home", IsCurrent: true}
	f.away = models.Property{UserID: f.neighbour.ID, Name: "Away", IsCurrent: true}
	mustCreate(t, db, &f.home, &f.away)
	mustCreate(t, db,
		&models.HouseholdMember{PropertyID: f.home.ID, UserID: &f.owner.ID, Name: "Owner", Role: "admin"},
		&models.HouseholdMember{PropertyID: f.home.ID, UserID: &f.partner.ID, Name: "Partner", Role: "member"},
		&models.HouseholdMember{PropertyID: f.away.ID, UserID: &f.neighbour.ID, Name: "Neighbour", Role: "admin"},
	)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(middleware.AuthRequired())
	g.GET("/templates/bills", f.h.ListBillTemplates)
	g.POST("/templates/bills", f.h.CreateBillTemplate)
	g.POST("/templates/bills/import", f.h.ImportBillTemplate)
	g.PUT("/templates/bills/:id", f.h.UpdateBillTemplate)
	g.DELETE("/templates/bills/:id", f.h.DeleteBillTemplate)
	g.PUT("/utilities/:id", NewUtilityHandler(db).Update)
	f.router = r

	f.ownerTok = testutil.SignToken(t, &f.owner)
	f.partnerTok = testutil.SignToken(t, &f.partner)
	f.nTok = testutil.SignToken(t, &f.neighbour)
	return f
}

func (f *templateFixture) list(t *testing.T, token, query string) []models.BillTemplate {
	t.Helper()
	rec := doGET(t, f.router, "/templates/bills"+query, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var out []models.BillTemplate
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *templateFixture) addTemplate(t *testing.T, propertyID uint, provider, utilityType string, isDefault bool) models.BillTemplate {
	t.Helper()
	tpl := models.BillTemplate{PropertyID: propertyID, Name: provider, Provider: provider, UtilityType: utilityType, IsDefault: isDefault, ExtractionRules: "[]"}
	mustCreate(t, f.db, &tpl)
	return tpl
}

func TestBillTemplatesBelongToTheProperty(t *testing.T) {
	f := setupTemplateFixture(t)

	rec := doJSON(t, f.router, http.MethodPost, "/templates/bills", f.ownerTok, map[string]any{
		"property_id": f.home.ID, "name": "Acme 2026", "provider": "Acme", "utility_type": "water",
		"extraction_rules": []map[string]any{{"field": "bill_number", "pattern": `n\. (\d+)`}},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created models.BillTemplate
	json.Unmarshal(rec.Body.Bytes(), &created)
	path := fmt.Sprintf("/templates/bills/%d", created.ID)

	// The partner shares the household: sees, edits and uses the template.
	if got := f.list(t, f.partnerTok, fmt.Sprintf("?property_id=%d", f.home.ID)); len(got) != 1 || got[0].ID != created.ID {
		t.Fatalf("partner list = %+v", got)
	}
	if rec := doJSON(t, f.router, http.MethodPut, path, f.partnerTok, map[string]any{"name": "Acme (edited)"}); rec.Code != http.StatusOK {
		t.Fatalf("partner update: %d %s", rec.Code, rec.Body)
	}
	pdf := &ExtractedTextWithPositions{RawText: "Bolletta n. 42"}
	if got := f.h.templateValuesForAnalysis(fmt.Sprint(created.ID), f.partner.ID, pdf); got["bill_number"] != "42" {
		t.Errorf("partner analysis = %v", got)
	}

	// The neighbour reaches nothing of it.
	if got := f.list(t, f.nTok, ""); len(got) != 0 {
		t.Errorf("neighbour list = %+v", got)
	}
	if got := f.list(t, f.nTok, fmt.Sprintf("?property_id=%d", f.home.ID)); len(got) != 0 {
		t.Errorf("neighbour list of home = %+v", got)
	}
	if rec := doJSON(t, f.router, http.MethodPut, path, f.nTok, map[string]any{"name": "mine"}); rec.Code != http.StatusNotFound {
		t.Errorf("neighbour update: %d", rec.Code)
	}
	if rec := doJSON(t, f.router, http.MethodDelete, path, f.nTok, nil); rec.Code != http.StatusNotFound {
		t.Errorf("neighbour delete: %d", rec.Code)
	}
	if got := f.h.templateValuesForAnalysis(fmt.Sprint(created.ID), f.neighbour.ID, pdf); got != nil {
		t.Errorf("neighbour analysis leaked %v", got)
	}
	if rec := doJSON(t, f.router, http.MethodPost, "/templates/bills", f.nTok, map[string]any{
		"property_id": f.home.ID, "name": "x", "provider": "x", "utility_type": "gas",
	}); rec.Code != http.StatusForbidden {
		t.Errorf("neighbour create in home: %d", rec.Code)
	}
	homeUtility := models.Utility{UserID: f.owner.ID, PropertyID: f.home.ID, Type: "water", Provider: "Acme"}
	awayUtility := models.Utility{UserID: f.neighbour.ID, PropertyID: f.away.ID, Type: "water", Provider: "Acme"}
	mustCreate(t, f.db, &homeUtility, &awayUtility)
	if got := f.list(t, f.nTok, fmt.Sprintf("?utility_id=%d", homeUtility.ID)); len(got) != 0 {
		t.Errorf("neighbour list by home utility = %+v", got)
	}
	if got := f.list(t, f.partnerTok, fmt.Sprintf("?utility_id=%d", homeUtility.ID)); len(got) != 1 {
		t.Errorf("partner list by utility = %+v", got)
	}

	if rec := doJSON(t, f.router, http.MethodDelete, path, f.partnerTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("partner delete: %d %s", rec.Code, rec.Body)
	}
}

func TestBillTemplateDefaultIsUniquePerProperty(t *testing.T) {
	f := setupTemplateFixture(t)
	first := f.addTemplate(t, f.home.ID, "Acme", "water", true)
	elsewhere := f.addTemplate(t, f.away.ID, "Acme", "water", true)

	rec := doJSON(t, f.router, http.MethodPost, "/templates/bills", f.partnerTok, map[string]any{
		"property_id": f.home.ID, "name": "new", "provider": " acme", "utility_type": "water", "is_default": true,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	f.db.First(&first, first.ID)
	f.db.First(&elsewhere, elsewhere.ID)
	if first.IsDefault {
		t.Error("the previous default of the property is still default")
	}
	if !elsewhere.IsDefault {
		t.Error("another property's default was cleared")
	}
}

func TestImportBillTemplate(t *testing.T) {
	f := setupTemplateFixture(t)
	file := map[string]any{
		"format": billTemplateFileFormat, "name": "Acme", "provider": "Acme", "utility_type": "water",
		"extraction_rules": []map[string]any{{"field": "amount_total", "anchor_text": "Totale", "value_pattern": `(\d+,\d{2})`}},
	}
	withProperty := func(id uint, overrides map[string]any) map[string]any {
		body := map[string]any{"property_id": id}
		for k, v := range file {
			body[k] = v
		}
		for k, v := range overrides {
			body[k] = v
		}
		return body
	}

	rec := doJSON(t, f.router, http.MethodPost, "/templates/bills/import", f.partnerTok, withProperty(f.home.ID, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("import: %d %s", rec.Code, rec.Body)
	}
	if got := f.list(t, f.ownerTok, fmt.Sprintf("?property_id=%d", f.home.ID)); len(got) != 1 || got[0].Provider != "Acme" {
		t.Errorf("owner list after import = %+v", got)
	}

	for name, body := range map[string]map[string]any{
		"not a template file": withProperty(f.home.ID, map[string]any{"format": "something-else"}),
		"no rules":            withProperty(f.home.ID, map[string]any{"extraction_rules": []any{}}),
		"unknown type":        withProperty(f.home.ID, map[string]any{"utility_type": "phone"}),
		"rule without field":  withProperty(f.home.ID, map[string]any{"extraction_rules": []map[string]any{{"pattern": "x"}}}),
		"non-RE2 pattern":     withProperty(f.home.ID, map[string]any{"extraction_rules": []map[string]any{{"field": "due_date", "pattern": `(?<=Scadenza)\d+`}}}),
	} {
		rec := doJSON(t, f.router, http.MethodPost, "/templates/bills/import", f.partnerTok, body)
		var resp struct {
			Code string `json:"error_code"`
		}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if rec.Code != http.StatusBadRequest || resp.Code != "template_file_invalid" {
			t.Errorf("%s: %d %s", name, rec.Code, resp.Code)
		}
	}

	if rec := doJSON(t, f.router, http.MethodPost, "/templates/bills/import", f.nTok, withProperty(f.home.ID, nil)); rec.Code != http.StatusForbidden {
		t.Errorf("neighbour import into home: %d", rec.Code)
	}
}

// A bill upload uses, within the service's property: the template asked for,
// else the service's default, else one matching provider and type.
func TestResolveBillTemplateOrder(t *testing.T) {
	f := setupTemplateFixture(t)
	plain := f.addTemplate(t, f.home.ID, "Acme", "water", false)
	byDefault := f.addTemplate(t, f.home.ID, "  ACME ", "water", true)
	foreign := f.addTemplate(t, f.away.ID, "Acme", "water", true)
	f.addTemplate(t, f.home.ID, "Acme", "gas", true)

	utility := models.Utility{PropertyID: f.home.ID, Type: "water", Provider: "Acme"}
	resolve := func(requested string) uint {
		t.Helper()
		tpl, ok := f.h.resolveBillTemplate(utility, requested)
		if !ok {
			return 0
		}
		return tpl.ID
	}

	if got := resolve(fmt.Sprint(plain.ID)); got != plain.ID {
		t.Errorf("requested template: got %d, want %d", got, plain.ID)
	}
	if got := resolve(""); got != byDefault.ID {
		t.Errorf("provider match: got %d, want the default %d", got, byDefault.ID)
	}
	if got := resolve(fmt.Sprint(foreign.ID)); got != byDefault.ID {
		t.Errorf("another property's template must be ignored: got %d", got)
	}

	utility = models.Utility{PropertyID: f.home.ID, Type: "water", Provider: "Volta"}
	if got := resolve(""); got != 0 {
		t.Errorf("no provider match: got %d", got)
	}
}
