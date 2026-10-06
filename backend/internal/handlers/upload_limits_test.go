package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
	"github.com/sgiraz/homelog/internal/testutil"
)

func pdfOfSize(n int) []byte {
	return append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("a"), n)...)
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Code string `json:"error_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return body.Code
}

func TestUploadQuota_CapsAttachedPDFsPerHousehold(t *testing.T) {
	t.Setenv("UPLOAD_QUOTA_MB", "1")
	f := setupBillPDFFixture(t)
	second := models.Bill{UtilityID: f.bill.UtilityID, BillNumber: "B2"}
	if err := f.db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	attach := func(id uint, size int) *httptest.ResponseRecorder {
		f.bill.ID = id
		return f.uploadBytes(t, f.token, "a.pdf", pdfOfSize(size))
	}

	first := f.bill.ID
	if rec := attach(first, 600<<10); rec.Code != http.StatusOK {
		t.Fatalf("first 600 KiB: %d %s", rec.Code, rec.Body.String())
	}
	rec := attach(second.ID, 600<<10)
	if rec.Code != http.StatusRequestEntityTooLarge || errorCode(t, rec) != "upload_quota_exceeded" {
		t.Fatalf("second 600 KiB: status %d, body %s", rec.Code, rec.Body.String())
	}
	// Replacing the first bill's PDF does not count the old file twice.
	if rec := attach(first, 900<<10); rec.Code != http.StatusOK {
		t.Fatalf("replace with 900 KiB: %d %s", rec.Code, rec.Body.String())
	}
	// Removing it frees the space again.
	f.bill.ID = first
	if rec := doJSON(t, f.router, http.MethodDelete, f.path(), f.token, nil); rec.Code != http.StatusOK {
		t.Fatalf("remove: %d", rec.Code)
	}
	if rec := attach(second.ID, 600<<10); rec.Code != http.StatusOK {
		t.Fatalf("after freeing space: %d %s", rec.Code, rec.Body.String())
	}
}

func TestUploadQuota_ZeroDisablesAndGarbageFallsBack(t *testing.T) {
	t.Setenv("UPLOAD_QUOTA_MB", "0")
	if got := uploadQuotaBytes(); got != 0 {
		t.Errorf("0 should disable the cap, got %d", got)
	}
	t.Setenv("UPLOAD_QUOTA_MB", "lots")
	if got := uploadQuotaBytes(); got != defaultUploadQuotaMB<<20 {
		t.Errorf("garbage should fall back to the default, got %d", got)
	}
	t.Setenv("UPLOAD_QUOTA_MB", "-5")
	if got := uploadQuotaBytes(); got != defaultUploadQuotaMB<<20 {
		t.Errorf("negative should fall back to the default, got %d", got)
	}
}

func TestUploadRateLimiter_PerUser(t *testing.T) {
	t.Setenv("JWT_SECRET", testutil.TestJWTSecret)
	db := testutil.NewDB(t)
	mk := func(email string) models.User {
		u := models.User{Email: email, PasswordHash: "x", Name: email, Role: "admin", IsActive: true}
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		return u
	}
	a, b := mk("a@example.com"), mk("b@example.com")
	ta, tb := testutil.SignToken(t, &a), testutil.SignToken(t, &b)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("")
	g.Use(middleware.AuthRequired())
	g.POST("/up", UploadRateLimiter(2, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 1; i <= 2; i++ {
		if rec := doJSON(t, r, http.MethodPost, "/up", ta, nil); rec.Code != http.StatusOK {
			t.Fatalf("request %d: %d", i, rec.Code)
		}
	}
	rec := doJSON(t, r, http.MethodPost, "/up", ta, nil)
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != "too_many_uploads" {
		t.Fatalf("3rd request: status %d, body %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, r, http.MethodPost, "/up", tb, nil); rec.Code != http.StatusOK {
		t.Errorf("another user must not be throttled: %d", rec.Code)
	}
}

func TestSweepUploads_RemovesOnlyOldUnreferencedFiles(t *testing.T) {
	f := setupBillPDFFixture(t)
	old := time.Now().Add(-48 * time.Hour)
	write := func(name string, mtime time.Time) {
		p := filepath.Join(f.uploads, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	attached := "bill_1_1791246675_baf0ae955a9e196f451c370ff831a283.pdf"
	write(attached, old)
	if err := f.db.Model(&f.bill).Update("pdf_url", "/uploads/"+attached).Error; err != nil {
		t.Fatal(err)
	}
	write("bill_1_1791246676_00000000000000000000000000000000.pdf", old) // orphan, old
	write("contract_1_1791246675_00000000000000000000000000000000.pdf", old)
	write("template_page_1791246675_ab12_1.png", old)
	write("bill_1_1791246677_11111111111111111111111111111111.pdf", time.Now()) // orphan but fresh
	write("notes.txt", old)                                                     // not ours

	if got := SweepUploads(f.db, f.uploads, 24*time.Hour); got != 3 {
		t.Errorf("removed %d files, want 3", got)
	}
	for name, want := range map[string]bool{
		attached: true,
		"bill_1_1791246676_00000000000000000000000000000000.pdf":     false,
		"contract_1_1791246675_00000000000000000000000000000000.pdf": false,
		"template_page_1791246675_ab12_1.png":                        false,
		"bill_1_1791246677_11111111111111111111111111111111.pdf":     true,
		"notes.txt": true,
	} {
		_, err := os.Stat(filepath.Join(f.uploads, name))
		if (err == nil) != want {
			t.Errorf("%s: exists=%v, want %v", name, err == nil, want)
		}
	}
}
