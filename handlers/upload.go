// Package handlers contains HTTP request handlers for file operations.
package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand/v2"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sqids/sqids-go"
	"gorm.io/gorm"

	"github.com/Sardonyx001/sefud/config"
	"github.com/Sardonyx001/sefud/models"
	"github.com/Sardonyx001/sefud/storage"
)

// UploadResponse represents the response structure for successful uploads
type UploadResponse struct {
	ID           string     `json:"id"`
	OriginalName string     `json:"original_name"`
	Size         int64      `json:"size"`
	URL          string     `json:"url"`
	DeleteToken  string     `json:"delete_token"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// FileHandler contains dependencies for file operations
type FileHandler struct {
	DB            *gorm.DB
	StorageClient *storage.StorageClient
	Config        *config.Config
	sqids         *sqids.Sqids
}

// NewFileHandler creates a new file handler with dependencies
func NewFileHandler(db *gorm.DB, storageClient *storage.StorageClient, cfg *config.Config) *FileHandler {
	// Create sqids instance with custom alphabet for shorter, URL-safe IDs
	s, _ := sqids.New(sqids.Options{
		MinLength: 6, // Exactly 6 characters
		Alphabet:  "FxnXM1kBN6cuhsAvjW3Co7l2RePyY8DwaU04Tzt9fHQrqSVKdpimLGIJOgb5ZE",
	})

	return &FileHandler{
		DB:            db,
		StorageClient: storageClient,
		Config:        cfg,
		sqids:         s,
	}
}

// UploadFile handles file upload requests with high-performance processing
// @Summary Upload a file
// @Description Upload a file to storage with high-performance chunked upload
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
			Error:   "Failed to parse multipart form",
			Code:    "INVALID_MULTIPART",
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
			Error:   "File type not allowed",
			Code:    "FORBIDDEN_MIME_TYPE",
			Details: contentType,
		})
	}

	// Generate UUID for database, short ID for public
	fileUUID := uuid.New().String()
	publicID := h.generateShortID()

	// Ensure short ID is unique (retry if collision)
	for {
		var existingFile models.File
		err := h.DB.Where("short_id = ?", publicID).First(&existingFile).Error
		if err == gorm.ErrRecordNotFound {
			break // ID is unique
		}
		// Generate new ID if collision
		publicID = h.generateShortID()
	}
	deleteToken := h.generateDeleteToken()
	storageKey := h.generateStorageKey(fileUUID, fileHeader.Filename)

	// Parse expiration if provided
	var expiresAt *time.Time
	if expiresStr := c.FormValue("expires"); expiresStr != "" {
		if duration, err := time.ParseDuration(expiresStr); err == nil {
			expires := time.Now().Add(duration)
			expiresAt = &expires
		}
	}

	// Read file data for hashing and seekable upload
	fileData, err := io.ReadAll(file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to read file data",
			Code:    "READ_ERROR",
			Details: err.Error(),
		})
	}

	// Calculate hashes
	md5Hash := md5.Sum(fileData)
	sha256Hash := sha256.Sum256(fileData)

	// Setup upload options for performance
	uploadOpts := storage.DefaultUploadOptions()
	uploadOpts.Metadata = map[string]string{
		"original-name": fileHeader.Filename,
		"file-id":       fileUUID,
		"upload-time":   startTime.Format(time.RFC3339),
	}

	// Use multipart upload for all files > 5MB for better performance through parallelization
	uploadOpts.EnableMultipart = fileHeader.Size > 5*1024*1024 // 5MB threshold

	log.Info("Upload configuration", "file_size", fileHeader.Size, "multipart", uploadOpts.EnableMultipart,
		"chunk_size", uploadOpts.ChunkSize, "concurrency", uploadOpts.MaxConcurrency)

	// Upload to storage
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create seekable reader for upload
	fileReader := bytes.NewReader(fileData)

	uploadStart := time.Now()
	log.Info("Starting upload", "file_id", publicID, "size", fileHeader.Size)
	err = h.StorageClient.Upload(ctx, storageKey, fileReader, uploadOpts)
	uploadDuration := time.Since(uploadStart)
	log.Info("Upload completed", "file_id", publicID, "duration", uploadDuration)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to upload file to storage",
			Code:    "STORAGE_ERROR",
			Details: err.Error(),
		})
	}

	// Calculate total duration
	totalDuration := time.Since(startTime)

	// Create file record (store UUID and short ID in database)
	fileRecord := &models.File{
		ID:             fileUUID,
		ShortID:        publicID,
		OriginalName:   fileHeader.Filename,
		ContentType:    contentType,
		Size:           fileHeader.Size,
		StorageKey:     storageKey,
		StorageBucket:  h.Config.Storage.BucketName,
		DeleteToken:    deleteToken,
		ExpiresAt:      expiresAt,
		UploadIP:       c.RealIP(),
		UserAgent:      c.Request().UserAgent(),
		MD5Hash:        hex.EncodeToString(md5Hash[:]),
		SHA256Hash:     hex.EncodeToString(sha256Hash[:]),
		UploadDuration: totalDuration,
	}

	// Save to database
	if err := h.DB.Create(fileRecord).Error; err != nil {
		// If DB save fails, try to cleanup Storage upload
		go func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			h.StorageClient.Delete(cleanupCtx, storageKey)
		}()

		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save file metadata",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
	}

	// Build response (return sqids hash as public ID)
	baseURL := h.getBaseURL(c)
	response := UploadResponse{
		ID:           publicID,
		OriginalName: fileHeader.Filename,
		Size:         fileHeader.Size,
		URL:          baseURL + "/" + publicID,
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

// generateShortID creates a short 6-character ID using sqids
func (h *FileHandler) generateShortID() string {
	// Generate a smaller random number that will encode to exactly 6 characters
	// With 62-char alphabet, 6 chars can represent up to 62^6 = ~56 billion combinations
	randomNum := uint64(rand.Uint64() % 56_800_000_000) // Stay well under the limit

	id, _ := h.sqids.Encode([]uint64{randomNum})
	// Ensure it's exactly 6 characters by padding if needed
	for len(id) < 6 {
		id = "F" + id // Pad with first character of alphabet
	}
	return id[:6] // Truncate to exactly 6 characters
}

// generateDeleteToken creates a secure random token for file deletion
func (h *FileHandler) generateDeleteToken() string {
	bytes := make([]byte, 32)
	cryptoRand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateStorageKey creates a unique key for storage
func (h *FileHandler) generateStorageKey(fileID, originalName string) string {
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
