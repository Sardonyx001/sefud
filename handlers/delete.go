package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func DeleteFile(c echo.Context) error {
	return c.String(http.StatusOK, "DeleteFile")
}
