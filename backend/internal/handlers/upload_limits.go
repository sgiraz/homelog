package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// defaultUploadQuotaMB is the per-household stored-PDF budget when
// UPLOAD_QUOTA_MB is unset: hundreds of bills, but bounded for the Pi's SD card.
const defaultUploadQuotaMB = 500

// uploadQuotaBytes returns the per-household cap on attached bill PDFs, read
// from UPLOAD_QUOTA_MB. 0 disables the cap; an unparsable or negative value
// falls back to the default rather than silently removing the protection.
func uploadQuotaBytes() int64 {
	mb := int64(defaultUploadQuotaMB)
	if v := os.Getenv("UPLOAD_QUOTA_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			mb = n
		} else {
			log.Printf("Warning: ignoring invalid UPLOAD_QUOTA_MB=%q, using %d", v, defaultUploadQuotaMB)
		}
	}
	return mb << 20
}

// propertyUploadBytes sums the size on disk of the PDFs attached to the bills
// of a household, ignoring excludeBillID (the bill whose PDF is being replaced).
func (h *PDFHandler) propertyUploadBytes(propertyID, excludeBillID uint) int64 {
	var urls []string
	h.db.Unscoped().Model(&models.Bill{}).
		Joins("JOIN utilities ON utilities.id = bills.utility_id").
		Where("utilities.property_id = ? AND bills.id <> ? AND bills.pdf_url <> ''", propertyID, excludeBillID).
		Pluck("bills.pdf_url", &urls)

	var total int64
	for _, u := range urls {
		name, ok := uploadFileName(u)
		if !ok {
			continue
		}
		if info, err := os.Stat(filepath.Join(h.uploadsDir, name)); err == nil {
			total += info.Size()
		}
	}
	return total
}

// withinUploadQuota answers 413 `upload_quota_exceeded` and returns false when
// adding a file of `size` bytes would push the household over its quota.
func (h *PDFHandler) withinUploadQuota(c *gin.Context, propertyID, excludeBillID uint, size int64) bool {
	quota := uploadQuotaBytes()
	if quota == 0 {
		return true
	}
	if h.propertyUploadBytes(propertyID, excludeBillID)+size > quota {
		apierr.FailWith(c, http.StatusRequestEntityTooLarge, "upload_quota_exceeded",
			"The household's PDF storage quota is full",
			map[string]string{"quota_mb": strconv.FormatInt(quota>>20, 10)})
		return false
	}
	return true
}

type uploadWindow struct {
	count int
	start time.Time
}

// UploadRateLimiter limits uploads per user per window (fixed window; per IP
// when unauthenticated). Uploads write to disk and shell out to poppler, so they
// get a tighter bound than the global rate limit.
func UploadRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	seen := make(map[string]*uploadWindow)

	return func(c *gin.Context) {
		key := c.ClientIP()
		if id, ok := middleware.GetUserID(c); ok {
			key = "u" + strconv.FormatUint(uint64(id), 10)
		}
		now := time.Now()

		mu.Lock()
		w, ok := seen[key]
		if !ok || now.Sub(w.start) > window {
			w = &uploadWindow{start: now}
			seen[key] = w
		}
		w.count++
		exceeded := w.count > limit
		if len(seen) > 10000 {
			for k, v := range seen {
				if now.Sub(v.start) > window {
					delete(seen, k)
				}
			}
		}
		mu.Unlock()

		if exceeded {
			apierr.Fail(c, http.StatusTooManyRequests, "too_many_uploads", "Too many uploads, slow down")
			c.Abort()
			return
		}
		c.Next()
	}
}

// sweepableUpload matches the files this server creates in uploads/ and may
// reclaim: bill/contract PDFs and the wizard's temporary renders. Anything else
// in the directory is left alone.
var sweepableUpload = regexp.MustCompile(`^(bill_|contract_|analyze_|temp_|page_|template_page_)`)

// SweepUploads deletes files in dir older than maxAge that no bill references
// (abandoned uploads, contract PDFs, leftover wizard renders). It returns how
// many were removed.
func SweepUploads(db *gorm.DB, dir string, maxAge time.Duration) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !sweepableUpload.MatchString(name) {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		var attached int64
		db.Unscoped().Model(&models.Bill{}).Where("pdf_url = ?", "/uploads/"+name).Count(&attached)
		if attached > 0 {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err == nil {
			removed++
		}
	}
	if removed > 0 {
		log.Printf("🧹 Removed %d unreferenced upload(s)", removed)
	}
	return removed
}
