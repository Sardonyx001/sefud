package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func UploadFile(c echo.Context) error {
	return c.String(http.StatusOK, "UploadFile")
}
