// Package handlers contains HTTP request handlers for file operations.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Sardonyx001/sefud/models"
)

// DeleteResponse represents the response structure for successful deletions
type DeleteResponse struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	DeletedAt time.Time `json:"deleted_at"`
}

// DeleteFile handles file deletion requests with proper cleanup
// @Summary Delete a file
// @Description Delete a file by its ID using delete token for authorization
// @Tags files
// @Accept json
// @Produce json
// @Param id path string true "File ID (UUID)"
// @Param token query string true "Delete token for authorization"
// @Success 200 {object} DeleteResponse "File deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid request parameters"
// @Failure 401 {object} ErrorResponse "Invalid delete token"
// @Failure 404 {object} ErrorResponse "File not found"
// @Failure 410 {object} ErrorResponse "File already deleted or expired"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /{id} [delete]
func (h *FileHandler) DeleteFile(c echo.Context) error {
	publicID := c.Param("id")
	if publicID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "File ID is required",
			Code:  "MISSING_FILE_ID",
		})
	}

	// Get delete token from query parameter or header
	deleteToken := c.QueryParam("token")
	if deleteToken == "" {
		deleteToken = c.Request().Header.Get("X-Delete-Token")
	}

	if deleteToken == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Delete token is required",
			Code:  "MISSING_DELETE_TOKEN",
		})
	}

	// Begin database transaction for atomic operations
	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Retrieve and lock the file record using short_id
	var fileRecord models.File
	err := tx.Where("short_id = ? AND delete_token = ?", publicID, deleteToken).First(&fileRecord).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "File not found or invalid delete token",
				Code:  "FILE_NOT_FOUND_OR_INVALID_TOKEN",
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Database error",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
	}

	// Check if file is already deleted
	if fileRecord.DeletedAt.Valid {
		tx.Rollback()
		return c.JSON(http.StatusGone, ErrorResponse{
			Error: "File has already been deleted",
			Code:  "FILE_ALREADY_DELETED",
		})
	}

	// Mark file as deleted in database (soft delete)
	now := time.Now()
	err = tx.Model(&fileRecord).Update("deleted_at", now).Error
	if err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to mark file as deleted",
			Code:    "DATABASE_UPDATE_ERROR",
			Details: err.Error(),
		})
	}

	// Commit the database transaction first
	if err := tx.Commit().Error; err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to commit deletion",
			Code:    "DATABASE_COMMIT_ERROR",
			Details: err.Error(),
		})
	}

	// Delete from storage asynchronously to avoid blocking the response
	// If Storage deletion fails, we can handle cleanup later via a background job
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := h.StorageClient.Delete(ctx, fileRecord.StorageKey); err != nil {
			// Log the error for monitoring/cleanup processes
			// In production, you might want to queue this for retry
			// or store failed deletions for manual cleanup
			c.Logger().Errorf("Failed to delete file %s from storage: %v", publicID, err)

			// Optionally, you could store failed deletions in a cleanup queue
			// h.enqueueCleanupTask(fileRecord.StorageKey, fileRecord.ID)
		} else {
			c.Logger().Infof("Successfully deleted file %s from storage", publicID)
		}
	}()

	// Return success response immediately
	response := DeleteResponse{
		ID:        publicID,
		Message:   "File deleted successfully",
		DeletedAt: now,
	}

	return c.JSON(http.StatusOK, response)
}

// CleanupExpiredFiles removes expired files from both database and storage
// This should be called periodically by a background job
func (h *FileHandler) CleanupExpiredFiles(ctx context.Context) error {
	// Find expired files that haven't been deleted yet
	var expiredFiles []models.File
	err := h.DB.Where("expires_at < ? AND deleted_at IS NULL", time.Now()).Find(&expiredFiles).Error
	if err != nil {
		return err
	}

	for _, file := range expiredFiles {
		// Mark as deleted in database
		if err := h.DB.Model(&file).Update("deleted_at", time.Now()).Error; err != nil {
			continue // Log error and continue with next file
		}

		// Delete from storage
		if err := h.StorageClient.Delete(ctx, file.StorageKey); err != nil {
			// Log error but don't fail the entire cleanup process
			continue
		}
	}

	return nil
}

// GetFileInfo returns file metadata without downloading the file
// This is useful for checking file existence and properties
func (h *FileHandler) GetFileInfo(c echo.Context) error {
	publicID := c.Param("id")
	if publicID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "File ID is required",
			Code:  "MISSING_FILE_ID",
		})
	}

	// Retrieve file metadata from database using short_id
	var fileRecord models.File
	err := h.DB.Where("short_id = ?", publicID).First(&fileRecord).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "File not found",
				Code:  "FILE_NOT_FOUND",
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Database error",
			Code:    "DATABASE_ERROR",
			Details: err.Error(),
		})
	}

	// Check if file has been deleted
	if fileRecord.DeletedAt.Valid {
		return c.JSON(http.StatusGone, ErrorResponse{
			Error: "File has been deleted",
			Code:  "FILE_DELETED",
		})
	}

	// Check if file has expired
	if fileRecord.IsExpired() {
		return c.JSON(http.StatusGone, ErrorResponse{
			Error: "File has expired",
			Code:  "FILE_EXPIRED",
		})
	}

	// Return file metadata (excluding sensitive fields) with public ID
	response := map[string]any{
		"id":            publicID,
		"original_name": fileRecord.OriginalName,
		"content_type":  fileRecord.ContentType,
		"size":          fileRecord.Size,
		"created_at":    fileRecord.CreatedAt,
		"expires_at":    fileRecord.ExpiresAt,
		"md5_hash":      fileRecord.MD5Hash,
		"sha256_hash":   fileRecord.SHA256Hash,
	}

	return c.JSON(http.StatusOK, response)
}
