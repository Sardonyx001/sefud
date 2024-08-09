package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func DownloadFile(c echo.Context) error {
	return c.String(http.StatusOK, "DownloadFile")
}
