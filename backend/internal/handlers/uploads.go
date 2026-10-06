package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/sgiraz/homelog/internal/apierr"
)

// publicUploadRe is the only kind of file served without authentication: the
// short-lived page previews the template wizard shows in <img> tags (which
// cannot send an Authorization header). Everything else under uploads/ — bill
// and contract PDFs above all — is private and reachable only through an
// authenticated endpoint.
var publicUploadRe = regexp.MustCompile(`^template_page_\d+(?:_[a-f0-9]+)?_\d+\.png$`)

// PublicUploads serves GET /uploads/:name from dir, restricted to
// publicUploadRe. Anything else answers 404, indistinguishable from a missing
// file so the existence of private uploads is not disclosed.
func PublicUploads(dir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}
		if !publicUploadRe.MatchString(name) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "private, no-store")
		c.File(filepath.Join(dir, name))
	}
}

// ServeBillPDF - GET /api/v1/utilities/:id/bills/:billId/pdf
// Streams the PDF attached to a bill to a member of its household.
func (h *PDFHandler) ServeBillPDF(c *gin.Context) {
	bill, ok := h.findBillForUser(c)
	if !ok {
		return
	}
	name, ok := uploadFileName(bill.PDFURL)
	if !ok {
		apierr.Fail(c, http.StatusNotFound, "bill_pdf_not_found", "This bill has no PDF")
		return
	}
	path := filepath.Join(h.uploadsDir, name)
	if _, err := os.Stat(path); err != nil {
		// The row points at a file that is gone (restored DB, manual cleanup).
		apierr.Fail(c, http.StatusNotFound, "bill_pdf_not_found", "The PDF file is missing")
		return
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, no-store")
	c.File(path)
}
