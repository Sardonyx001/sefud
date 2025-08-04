// Package handlers contains HTTP request handlers for file operations.
package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/Sardonyx001/sefud/models"
)

// DownloadFile handles file download requests with high-performance streaming
// @Summary Download a file
// @Description Download a file by its ID with streaming support and range requests
// @Tags files
// @Produce application/octet-stream
// @Param id path string true "File ID (UUID)"
// @Param Range header string false "Range header for partial content requests"
// @Success 200 {file} binary "File content"
// @Success 206 {file} binary "Partial file content"
// @Failure 400 {object} ErrorResponse "Invalid file ID"
// @Failure 404 {object} ErrorResponse "File not found"
// @Failure 410 {object} ErrorResponse "File expired"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /{id} [get]
func (h *FileHandler) DownloadFile(c echo.Context) error {
	fileID := c.Param("id")
	if fileID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "File ID is required",
			Code:  "MISSING_FILE_ID",
		})
	}

	// Retrieve file metadata from database
	var fileRecord models.File
	err := h.DB.Where("id = ?", fileID).First(&fileRecord).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "File not found",
				Code:  "FILE_NOT_FOUND",
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Database error",
			Code:  "DATABASE_ERROR",
			Details: err.Error(),
		})
	}

	// Check if file has expired
	if fileRecord.IsExpired() {
		return c.JSON(http.StatusGone, ErrorResponse{
			Error: "File has expired",
			Code:  "FILE_EXPIRED",
		})
	}

	// Get file from R2 with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Check if client supports range requests
	rangeHeader := c.Request().Header.Get("Range")
	
	var reader io.ReadCloser
	var contentLength int64 = fileRecord.Size
	var statusCode int = http.StatusOK
	
	if rangeHeader != "" {
		// Handle range requests for partial content
		reader, contentLength, statusCode, err = h.handleRangeRequest(ctx, &fileRecord, rangeHeader)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to process range request",
				Code:  "RANGE_ERROR",
				Details: err.Error(),
			})
		}
	} else {
		// Full file download
		reader, err = h.R2Client.Download(ctx, fileRecord.R2Key)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "Failed to retrieve file from storage",
				Code:  "STORAGE_ERROR",
				Details: err.Error(),
			})
		}
	}
	defer reader.Close()

	// Set response headers for optimal download performance
	h.setDownloadHeaders(c, &fileRecord, contentLength, statusCode, rangeHeader)

	// Stream the file content directly to the client
	// This is more memory-efficient than loading the entire file
	_, err = io.Copy(c.Response().Writer, reader)
	if err != nil {
		// Log error but don't return JSON since we've already started streaming
		c.Logger().Errorf("Error streaming file %s: %v", fileID, err)
		return err
	}

	return nil
}

// handleRangeRequest processes HTTP range requests for partial content
func (h *FileHandler) handleRangeRequest(ctx context.Context, fileRecord *models.File, rangeHeader string) (io.ReadCloser, int64, int, error) {
	// Parse range header (simplified - only handles single range)
	// Format: "bytes=start-end" or "bytes=start-" or "bytes=-suffix"
	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	rangeParts := strings.Split(rangeSpec, "-")
	
	if len(rangeParts) != 2 {
		return nil, 0, http.StatusBadRequest, fmt.Errorf("invalid range format")
	}

	var start, end int64
	var err error
	
	if rangeParts[0] == "" {
		// Suffix range: bytes=-500 (last 500 bytes)
		if rangeParts[1] == "" {
			return nil, 0, http.StatusBadRequest, fmt.Errorf("invalid range format")
		}
		suffix, err := strconv.ParseInt(rangeParts[1], 10, 64)
		if err != nil {
			return nil, 0, http.StatusBadRequest, err
		}
		start = fileRecord.Size - suffix
		if start < 0 {
			start = 0
		}
		end = fileRecord.Size - 1
	} else {
		// Start range: bytes=0-499 or bytes=500-
		start, err = strconv.ParseInt(rangeParts[0], 10, 64)
		if err != nil {
			return nil, 0, http.StatusBadRequest, err
		}
		
		if rangeParts[1] == "" {
			// Open-ended range: bytes=500-
			end = fileRecord.Size - 1
		} else {
			end, err = strconv.ParseInt(rangeParts[1], 10, 64)
			if err != nil {
				return nil, 0, http.StatusBadRequest, err
			}
		}
	}

	// Validate range
	if start < 0 || end >= fileRecord.Size || start > end {
		return nil, 0, http.StatusRequestedRangeNotSatisfiable, fmt.Errorf("invalid range")
	}

	// For now, we'll get the full file and return a section reader
	// In a production system, you might want to use S3's range request capabilities
	fullReader, err := h.R2Client.Download(ctx, fileRecord.R2Key)
	if err != nil {
		return nil, 0, http.StatusInternalServerError, err
	}

	// Skip to the start position
	if start > 0 {
		_, err = io.CopyN(io.Discard, fullReader, start)
		if err != nil {
			fullReader.Close()
			return nil, 0, http.StatusInternalServerError, err
		}
	}

	// Create a limited reader for the range
	contentLength := end - start + 1
	limitedReader := io.LimitReader(fullReader, contentLength)
	
	// Wrap in a ReadCloser
	rangeReader := &rangeReadCloser{
		Reader: limitedReader,
		closer: fullReader,
	}

	return rangeReader, contentLength, http.StatusPartialContent, nil
}

// rangeReadCloser wraps a limited reader with a closer
type rangeReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r *rangeReadCloser) Close() error {
	return r.closer.Close()
}

// setDownloadHeaders sets appropriate headers for file downloads
func (h *FileHandler) setDownloadHeaders(c echo.Context, fileRecord *models.File, contentLength int64, statusCode int, rangeHeader string) {
	response := c.Response()
	
	// Set content headers
	response.Header().Set("Content-Type", fileRecord.ContentType)
	response.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	
	// Set filename for download
	escapedName := strings.ReplaceAll(fileRecord.OriginalName, `"`, `\"`)
	response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, escapedName))
	
	// Set caching headers
	response.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
	response.Header().Set("ETag", fmt.Sprintf(`"%s"`, fileRecord.MD5Hash))
	
	// Set range headers for partial content
	if statusCode == http.StatusPartialContent {
		rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
		response.Header().Set("Content-Range", fmt.Sprintf("bytes %s/%d", rangeSpec, fileRecord.Size))
		response.Header().Set("Accept-Ranges", "bytes")
	} else {
		response.Header().Set("Accept-Ranges", "bytes")
	}
	
	// Performance headers
	response.Header().Set("X-Content-Type-Options", "nosniff")
	
	// Set status code
	response.WriteHeader(statusCode)
}
