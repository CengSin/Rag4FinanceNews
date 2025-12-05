package api

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
)

func NewSession(c echo.Context) error {
	return c.String(http.StatusOK, uuid.New().String())
}
