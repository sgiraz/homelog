package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestServeBillPDF_OnlyForHouseholdMembers(t *testing.T) {
	f := setupBillPDFFixture(t)

	if rec := doJSON(t, f.router, http.MethodGet, f.path(), f.token, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("no pdf yet: status %d, want 404", rec.Code)
	}

	f.upload(t, f.token, "a.pdf")

	rec := doJSON(t, f.router, http.MethodGet, f.path(), f.token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("member: status %d, body %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q", ct)
	}
	if !strings.HasPrefix(rec.Body.String(), "%PDF-") {
		t.Errorf("body is not the stored file: %q", rec.Body.String())
	}
	if rec := doJSON(t, f.router, http.MethodGet, f.path(), f.outsider, nil); rec.Code != http.StatusForbidden {
		t.Errorf("outsider: status %d, want 403", rec.Code)
	}
	if rec := doJSON(t, f.router, http.MethodGet, f.path(), "", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: status %d, want 401", rec.Code)
	}
}

func TestPublicUploads_ServesOnlyTemplatePagePreviews(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("template_page_1791246675_ab12_1.png")
	write("bill_1_1791246675_baf0ae955a9e196f451c370ff831a283.pdf")
	write("contract_1_1791246675_baf0ae955a9e196f451c370ff831a283.pdf")
	write("page_1791246675-1.png")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := PublicUploads(dir)
	r.GET("/uploads/:name", h)

	cases := map[string]int{
		"template_page_1791246675_ab12_1.png":                        http.StatusOK,
		"bill_1_1791246675_baf0ae955a9e196f451c370ff831a283.pdf":     http.StatusNotFound,
		"contract_1_1791246675_baf0ae955a9e196f451c370ff831a283.pdf": http.StatusNotFound,
		"page_1791246675-1.png":                                      http.StatusNotFound,
		"missing.png":                                                http.StatusNotFound,
		"..%2Fhomelog.db":                                            http.StatusNotFound,
	}
	for name, want := range cases {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/"+name, nil))
		if rec.Code != want {
			t.Errorf("GET /uploads/%s: status %d, want %d", name, rec.Code, want)
		}
	}
}
