// Package handlers contains HTTP request handlers for file operations.
package handlers

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Sardonyx001/sefud/config"
	"github.com/Sardonyx001/sefud/models"
	"github.com/Sardonyx001/sefud/storage"
)

// UploadResponse represents the response structure for successful uploads
type UploadResponse struct {
	ID          string `json:"id"`
	OriginalName string `json:"original_name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"`
	DeleteToken string `json:"delete_token"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// FileHandler contains dependencies for file operations
type FileHandler struct {
	DB       *gorm.DB
	R2Client *storage.R2Client
	Config   *config.Config
}

// NewFileHandler creates a new file handler with dependencies
func NewFileHandler(db *gorm.DB, r2Client *storage.R2Client, cfg *config.Config) *FileHandler {
	return &FileHandler{
		DB:       db,
		R2Client: r2Client,
		Config:   cfg,
	}
}

// UploadFile handles file upload requests with high-performance processing
// @Summary Upload a file
// @Description Upload a file to R2 storage with high-performance chunked upload
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param expires formData string false "Expiration duration (e.g., '24h', '7d')"
// @Success 200 {object} UploadResponse "File uploaded successfully"
// @Failure 400 {object} ErrorResponse "Bad request - invalid file or parameters"
// @Failure 413 {object} ErrorResponse "File too large"
// @Failure 415 {object} ErrorResponse "Unsupported media type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /up [post]
func (h *FileHandler) UploadFile(c echo.Context) error {
	startTime := time.Now()
	
	// Parse multipart form with size limit
	err := c.Request().ParseMultipartForm(h.Config.App.MaxUploadSize)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Failed to parse multipart form",
			Code:  "INVALID_MULTIPART",
			Details: err.Error(),
		})
	}

	// Get the file from form
	file, fileHeader, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No file provided",
			Code:  "MISSING_FILE",
		})
	}
	defer file.Close()

	// Validate file size
	if fileHeader.Size > h.Config.App.MaxUploadSize {
		return c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
			Error: fmt.Sprintf("File too large. Maximum size: %d bytes", h.Config.App.MaxUploadSize),
			Code:  "FILE_TOO_LARGE",
		})
	}

	// Detect content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(fileHeader.Filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// Check MIME type blacklist
	if h.isBlacklistedMimeType(contentType) {
		return c.JSON(http.StatusUnsupportedMediaType, ErrorResponse{
			Error: "File type not allowed",
			Code:  "FORBIDDEN_MIME_TYPE",
			Details: contentType,
		})
	}

	// Generate file ID and keys
	fileID := uuid.New().String()
	deleteToken := h.generateDeleteToken()
	r2Key := h.generateR2Key(fileID, fileHeader.Filename)

	// Parse expiration if provided
	var expiresAt *time.Time
	if expiresStr := c.FormValue("expires"); expiresStr != "" {
		if duration, err := time.ParseDuration(expiresStr); err == nil {
			expires := time.Now().Add(duration)
			expiresAt = &expires
		}
	}

	// Create tee readers for concurrent hashing and upload
	md5Hash := md5.New()
	sha256Hash := sha256.New()
	
	// Create a multi-writer that writes to both hash functions
	hashWriter := io.MultiWriter(md5Hash, sha256Hash)
	
	// Use TeeReader to hash while reading for upload
	teeReader := io.TeeReader(file, hashWriter)

	// Setup upload options for performance
	uploadOpts := storage.DefaultUploadOptions()
	uploadOpts.ContentType = contentType
	uploadOpts.Metadata = map[string]string{
		"original-name": fileHeader.Filename,
		"file-id":       fileID,
		"upload-time":   startTime.Format(time.RFC3339),
	}

	// Determine if we should use multipart upload
	uploadOpts.EnableMultipart = fileHeader.Size > 10*1024*1024 // 10MB threshold
	if uploadOpts.EnableMultipart {
		uploadOpts.MaxConcurrency = 6 // Higher concurrency for large files
		uploadOpts.ChunkSize = 8 * 1024 * 1024 // 8MB chunks for optimal performance
	}

	// Upload to R2
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = h.R2Client.Upload(ctx, r2Key, teeReader, uploadOpts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to upload file to storage",
			Code:  "STORAGE_ERROR",
			Details: err.Error(),
		})
	}

	// Calculate upload duration
	uploadDuration := time.Since(startTime)

	// Create file record
	fileRecord := &models.File{
		ID:             fileID,
		OriginalName:   fileHeader.Filename,
		ContentType:    contentType,
		Size:           fileHeader.Size,
		R2Key:          r2Key,
		R2Bucket:       h.Config.R2.BucketName,
		DeleteToken:    deleteToken,
		ExpiresAt:      expiresAt,
		UploadIP:       c.RealIP(),
		UserAgent:     c.Request().UserAgent(),
		MD5Hash:        hex.EncodeToString(md5Hash.Sum(nil)),
		SHA256Hash:     hex.EncodeToString(sha256Hash.Sum(nil)),
		UploadDuration: uploadDuration,
	}

	// Save to database
	if err := h.DB.Create(fileRecord).Error; err != nil {
		// If DB save fails, try to cleanup R2 upload
		go func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			h.R2Client.Delete(cleanupCtx, r2Key)
		}()

		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to save file metadata",
			Code:  "DATABASE_ERROR",
			Details: err.Error(),
		})
	}

	// Build response
	baseURL := h.getBaseURL(c)
	response := UploadResponse{
		ID:           fileID,
		OriginalName: fileHeader.Filename,
		Size:         fileHeader.Size,
		ContentType:  contentType,
		URL:          baseURL + "/" + fileID,
		DeleteToken:  deleteToken,
		ExpiresAt:    expiresAt,
	}

	return c.JSON(http.StatusOK, response)
}

// isBlacklistedMimeType checks if the MIME type is in the blacklist
func (h *FileHandler) isBlacklistedMimeType(contentType string) bool {
	// Extract base MIME type (remove parameters like charset)
	baseMimeType := strings.Split(contentType, ";")[0]
	baseMimeType = strings.TrimSpace(strings.ToLower(baseMimeType))
	
	return slices.Contains(h.Config.App.MimeBlacklist, baseMimeType)
}

// generateDeleteToken creates a secure random token for file deletion
func (h *FileHandler) generateDeleteToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateR2Key creates a unique key for R2 storage
func (h *FileHandler) generateR2Key(fileID, originalName string) string {
	ext := filepath.Ext(originalName)
	timestamp := time.Now().Format("2006/01/02")
	return fmt.Sprintf("%s/%s%s", timestamp, fileID, ext)
}

// getBaseURL extracts the base URL from the request
func (h *FileHandler) getBaseURL(c echo.Context) string {
	scheme := "http"
	if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request().Host)
}
