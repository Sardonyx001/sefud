// Package handlers contains HTTP request handlers for file operations.
package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// DeleteFile handles file deletion requests
// @Summary Delete a file
// @Description Delete a file by its ID
// @Tags files
// @Produce json
// @Param id path string true "File ID"
// @Success 200 {object} map[string]interface{} "File deleted successfully"
// @Failure 404 {object} map[string]interface{} "File not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /{id} [delete]
func DeleteFile(c echo.Context) error {
	return c.String(http.StatusOK, "DeleteFile")
}
