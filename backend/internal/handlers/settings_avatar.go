package handlers

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/sgiraz/homelog/internal/apierr"
	"github.com/sgiraz/homelog/internal/middleware"
	"github.com/sgiraz/homelog/internal/models"
)

// UploadAvatar - POST /api/v1/settings/avatar
func (h *SettingsHandler) UploadAvatar(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "no_file", "No file was uploaded")
		return
	}
	defer file.Close()

	// Validate file size (max 5MB)
	if header.Size > 5*1024*1024 {
		apierr.Fail(c, http.StatusBadRequest, "file_too_large", "The file is too large (max 5 MB)")
		return
	}

	// Validate content type by reading first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "file_unreadable", "The file could not be read")
		return
	}
	contentType := http.DetectContentType(buf[:n])
	allowed := map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}
	if !allowed[contentType] {
		apierr.Fail(c, http.StatusBadRequest, "image_type_invalid", "The file must be an image (JPEG, PNG or WebP)")
		return
	}

	// Reset reader to beginning
	if seeker, ok := file.(io.ReadSeeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// Decode image
	src, _, err := image.Decode(file)
	if err != nil {
		apierr.Fail(c, http.StatusBadRequest, "image_unreadable", "The image could not be read")
		return
	}

	// Center-crop to square and resize to 256x256
	bounds := src.Bounds()
	imgW, imgH := bounds.Dx(), bounds.Dy()
	cropSize := imgW
	if imgH < imgW {
		cropSize = imgH
	}
	offsetX := (imgW - cropSize) / 2
	offsetY := (imgH - cropSize) / 2

	// Create cropped sub-image
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	var cropped image.Image
	if si, ok := src.(subImager); ok {
		cropped = si.SubImage(image.Rect(
			bounds.Min.X+offsetX, bounds.Min.Y+offsetY,
			bounds.Min.X+offsetX+cropSize, bounds.Min.Y+offsetY+cropSize,
		))
	} else {
		cropped = src
	}

	// Resize to 256x256
	dst := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)

	// Ensure avatars directory exists (derive from DB_PATH for consistent paths across dev/prod)
	avatarDir := filepath.Join(dataDir(), "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to create avatar directory")
		return
	}

	// Generate filename. A cryptographic suffix keeps /avatars URLs from being
	// enumerated by guessing (userID, timestamp).
	filename := fmt.Sprintf("%d_%d_%s.jpg", userID, time.Now().UnixNano(), randomSuffix(8))
	avatarPath := filepath.Join(avatarDir, filename)

	outFile, err := os.Create(avatarPath)
	if err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to save avatar")
		return
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, dst, &jpeg.Options{Quality: 85}); err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to encode avatar")
		return
	}

	// Delete old avatar if exists
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "user_not_found", "User not found")
		return
	}
	if user.AvatarPath != "" {
		oldPath := filepath.Join(dataDir(), user.AvatarPath)
		os.Remove(oldPath)
	}

	// Update user in DB
	relativePath := "avatars/" + filename
	if err := h.db.Model(&user).Update("avatar_path", relativePath).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update user")
		return
	}
	user.AvatarPath = relativePath

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// DeleteAvatar - DELETE /api/v1/settings/avatar
func (h *SettingsHandler) DeleteAvatar(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierr.Fail(c, http.StatusUnauthorized, "not_authenticated", "You are not signed in")
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "user_not_found", "User not found")
		return
	}

	// Delete file from disk
	if user.AvatarPath != "" {
		oldPath := filepath.Join(dataDir(), user.AvatarPath)
		os.Remove(oldPath)
	}

	// Clear in DB
	if err := h.db.Model(&user).Update("avatar_path", "").Error; err != nil {
		apierr.Fail(c, http.StatusInternalServerError, "server_error", "Failed to update user")
		return
	}
	user.AvatarPath = ""

	c.JSON(http.StatusOK, gin.H{"user": user})
}
