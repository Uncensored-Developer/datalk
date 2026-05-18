package models

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	"github.com/labstack/echo/v4"
)

func (h *Handler) ListModels(c echo.Context) error {
	if _, err := echohandlers.UserID(c); err != nil {
		return err
	}

	models, err := h.service.ListAvailableModels(c.Request().Context())
	if err != nil {
		return echohandlers.Error(err)
	}

	return c.JSON(http.StatusOK, toModelResponses(models))
}
