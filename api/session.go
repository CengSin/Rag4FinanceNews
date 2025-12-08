package api

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"rag4financenew/dao"
)

func NewSession(c echo.Context) error {
	return c.String(http.StatusOK, uuid.New().String())
}

func GetSessionHistory(c echo.Context) error {
	sessionId := c.QueryParam("session_id")
	if sessionId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "session_id is required")
	}

	messages, err := dao.QueryMessagesBySessionId(c.Request().Context(), sessionId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"session_id": sessionId,
		"messages":   messages,
	})
}

func ListSessions(c echo.Context) error {
	sessions, err := dao.ListSessions(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": sessions,
	})
}

func DeleteSession(c echo.Context) error {
	sessionId := c.Param("session_id")
	if sessionId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "session_id is required")
	}

	if err := dao.DeleteSession(c.Request().Context(), sessionId); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
