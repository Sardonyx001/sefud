// Package handlers contains HTTP request handlers for file operations.
package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// UploadFile handles file upload requests
// @Summary Upload a file
// @Description Upload a file to the server
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 200 {object} map[string]interface{} "File uploaded successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /up [post]
func UploadFile(c echo.Context) error {
	return c.String(http.StatusOK, "UploadFile")
}
