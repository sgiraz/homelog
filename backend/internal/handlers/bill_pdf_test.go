package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

type billPDFFixture struct {
	db       *gorm.DB
	router   *gin.Engine
	bill     models.Bill
	token    string
	outsider string
	uploads  string
}

func setupBillPDFFixture(t *testing.T) *billPDFFixture {
	t.Helper()
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	dataDir := t.TempDir()
	t.Setenv("DB_PATH", filepath.Join(dataDir, "homelog.db"))
	uploads := filepath.Join(dataDir, "uploads")
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		t.Fatal(err)
	}
	db := testutil.NewDB(t)

	mkUser := func(email string) models.User {
		u := models.User{Email: email, PasswordHash: "x", Name: email, Role: "admin", IsActive: true}
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		return u
	}
	owner, other := mkUser("owner@example.com"), mkUser("other@example.com")
	prop := models.Property{UserID: owner.ID, Name: "Casa", IsCurrent: true}
	if err := db.Create(&prop).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.HouseholdMember{PropertyID: prop.ID, UserID: &owner.ID, Name: "Owner", Role: "admin"}).Error; err != nil {
		t.Fatal(err)
	}
	util := models.Utility{UserID: owner.ID, PropertyID: prop.ID, Type: "water", Provider: "Acme"}
	if err := db.Create(&util).Error; err != nil {
		t.Fatal(err)
	}
	bill := models.Bill{UtilityID: util.ID, BillNumber: "B1"}
	if err := db.Create(&bill).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(middleware.AuthRequired())
	pdf := NewPDFHandler(db)
	g.GET("/utilities/:id/bills/:billId/pdf", pdf.ServeBillPDF)
	g.POST("/utilities/:id/bills/:billId/pdf", pdf.AttachBillPDF)
	g.DELETE("/utilities/:id/bills/:billId/pdf", pdf.DeleteBillPDF)
	g.DELETE("/utilities/:id/bills/:billId", NewUtilityHandler(db).DeleteBill)
	g.POST("/utilities/:id/bills", NewUtilityHandler(db).AddBill)

	return &billPDFFixture{
		db: db, router: r, bill: bill, uploads: uploads,
		token: testutil.SignToken(t, &owner), outsider: testutil.SignToken(t, &other),
	}
}

func (f *billPDFFixture) path() string { return "/utilities/1/bills/" + itoa(f.bill.ID) + "/pdf" }

func (f *billPDFFixture) upload(t *testing.T, token, filename string) *httptest.ResponseRecorder {
	t.Helper()
	return f.uploadBytes(t, token, filename, []byte("%PDF-1.4 test"))
}

func (f *billPDFFixture) uploadBytes(t *testing.T, token, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("pdf_file", filename)
	part.Write(content)
	w.Close()
	req := httptest.NewRequest(http.MethodPost, f.path(), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func (f *billPDFFixture) storedURL(t *testing.T) string {
	t.Helper()
	var b models.Bill
	if err := f.db.First(&b, f.bill.ID).Error; err != nil {
		t.Fatal(err)
	}
	return b.PDFURL
}

func (f *billPDFFixture) fileExists(url string) bool {
	_, err := os.Stat(filepath.Join(f.uploads, strings.TrimPrefix(url, "/uploads/")))
	return err == nil
}

func TestAttachBillPDF_StoresFileAndURL(t *testing.T) {
	f := setupBillPDFFixture(t)
	rec := f.upload(t, f.token, "bolletta.pdf")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
	var got struct {
		PDFURL string `json:"pdf_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.PDFURL, "/uploads/") || !f.fileExists(got.PDFURL) {
		t.Fatalf("file not saved for %q", got.PDFURL)
	}
	if f.storedURL(t) != got.PDFURL {
		t.Errorf("bill.pdf_url = %q, want %q", f.storedURL(t), got.PDFURL)
	}
}

func TestAttachBillPDF_ReplaceDeletesPreviousFile(t *testing.T) {
	f := setupBillPDFFixture(t)
	f.upload(t, f.token, "a.pdf")
	first := f.storedURL(t)
	if rec := f.upload(t, f.token, "b.pdf"); rec.Code != http.StatusOK {
		t.Fatalf("replace: %d %s", rec.Code, rec.Body.String())
	}
	second := f.storedURL(t)
	if first == second {
		t.Fatal("URL did not change on replace")
	}
	if f.fileExists(first) {
		t.Error("old file left on disk after replace")
	}
	if !f.fileExists(second) {
		t.Error("new file missing")
	}
}

func TestAttachBillPDF_RejectsNonPDFAndOutsiders(t *testing.T) {
	f := setupBillPDFFixture(t)
	if rec := f.upload(t, f.token, "evil.exe"); rec.Code != http.StatusBadRequest {
		t.Errorf("non-pdf: status %d, want 400", rec.Code)
	}
	if rec := f.upload(t, f.outsider, "a.pdf"); rec.Code != http.StatusForbidden {
		t.Errorf("outsider: status %d, want 403", rec.Code)
	}
	if got := f.storedURL(t); got != "" {
		t.Errorf("rejected uploads still set pdf_url = %q", got)
	}
	if entries, _ := os.ReadDir(f.uploads); len(entries) != 0 {
		t.Errorf("rejected uploads left %d file(s) on disk", len(entries))
	}
}

func TestDeleteBillPDF_RemovesFileAndURL(t *testing.T) {
	f := setupBillPDFFixture(t)
	f.upload(t, f.token, "a.pdf")
	url := f.storedURL(t)

	if rec := doJSON(t, f.router, http.MethodDelete, f.path(), f.outsider, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("outsider delete: %d", rec.Code)
	}
	if rec := doJSON(t, f.router, http.MethodDelete, f.path(), f.token, nil); rec.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	if f.storedURL(t) != "" || f.fileExists(url) {
		t.Error("PDF not fully removed")
	}
}

func TestDeleteBill_RemovesAttachedPDF(t *testing.T) {
	f := setupBillPDFFixture(t)
	f.upload(t, f.token, "a.pdf")
	url := f.storedURL(t)

	rec := doJSON(t, f.router, http.MethodDelete, "/utilities/1/bills/"+itoa(f.bill.ID), f.token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete bill: %d %s", rec.Code, rec.Body.String())
	}
	if f.fileExists(url) {
		t.Error("PDF file survived bill deletion")
	}
}

func TestRemoveUploadedFile_IgnoresPathsOutsideUploads(t *testing.T) {
	f := setupBillPDFFixture(t)
	outside := filepath.Join(filepath.Dir(f.uploads), "keep.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/uploads/../keep.txt", "/uploads/sub/../../keep.txt", "../keep.txt", "keep.txt", "/uploads/", ""} {
		removeUploadedFile(url)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("file outside uploads was deleted: %v", err)
	}
}

func TestAttachBillPDF_RejectsRenamedNonPDF(t *testing.T) {
	f := setupBillPDFFixture(t)
	rec := f.uploadBytes(t, f.token, "bolletta.pdf", []byte("<html><script>alert(1)</script></html>"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
	if f.storedURL(t) != "" {
		t.Error("renamed file was attached")
	}
	if entries, _ := os.ReadDir(f.uploads); len(entries) != 0 {
		t.Errorf("renamed file left on disk (%d entries)", len(entries))
	}
}

func TestAttachBillPDF_RejectsOversizedBody(t *testing.T) {
	f := setupBillPDFFixture(t)
	big := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("a"), pdfUploadBodyLimit+1024)...)
	rec := f.uploadBytes(t, f.token, "big.pdf", big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 413, body %s", rec.Code, rec.Body.String())
	}
	if entries, _ := os.ReadDir(f.uploads); len(entries) != 0 {
		t.Errorf("oversized upload left %d file(s) on disk", len(entries))
	}
}

func (f *billPDFFixture) addBill(t *testing.T, number, pdfURL string) *httptest.ResponseRecorder {
	t.Helper()
	return doJSON(t, f.router, http.MethodPost, "/utilities/1/bills", f.token, map[string]any{
		"bill_number": number, "amount_total": 10, "pdf_url": pdfURL,
		"period_start": "2026-01-01T00:00:00Z", "period_end": "2026-02-01T00:00:00Z",
		"due_date": "2026-03-01T00:00:00Z", "issue_date": "2026-02-05T00:00:00Z",
	})
}

func TestAddBill_PDFURLMustBeAnUnusedUploadOfThatUtility(t *testing.T) {
	f := setupBillPDFFixture(t)
	f.upload(t, f.token, "a.pdf")
	own := f.storedURL(t) // already attached to the fixture bill

	for name, url := range map[string]string{
		"already attached to another bill": own,
		"other utility":                    "/uploads/bill_99_1791246675_baf0ae955a9e196f451c370ff831a283.pdf",
		"traversal":                        "/uploads/../homelog.db",
		"external":                         "https://evil.example/x.pdf",
		"wrong shape":                      "/uploads/avatar_1.png",
	} {
		if rec := f.addBill(t, "N-"+name, url); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, rec.Code)
		}
	}

	fresh := "/uploads/bill_1_1791246675_baf0ae955a9e196f451c370ff831a283.pdf"
	if rec := f.addBill(t, "N-ok", fresh); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Errorf("valid url: status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec := f.addBill(t, "N-empty", ""); rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Errorf("empty url: status %d, body %s", rec.Code, rec.Body.String())
	}
}
