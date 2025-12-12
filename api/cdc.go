package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"rag4financenew/config"
	"rag4financenew/handle"
)

type StartCDCReq struct {
	Configs []config.Cdc `json:"configs"`
}

func StartCDC(c echo.Context) error {
	var req StartCDCReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	configs := req.Configs
	if len(configs) == 0 {
		configs = handle.GetDefaultCDC()
	}
	if len(configs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "cdc configs is empty")
	}

	started, err := handle.StartCDC(configs)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"started": started,
	})
}

func StopCDC(c echo.Context) error {
	stopped := handle.StopAllCDC()
	return c.JSON(http.StatusOK, map[string]any{
		"stopped": stopped,
	})
}
