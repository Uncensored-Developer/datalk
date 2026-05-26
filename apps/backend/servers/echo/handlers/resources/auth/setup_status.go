package auth

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	"github.com/labstack/echo/v4"
)

type setupStatusResponse struct {
	SetupRequired bool `json:"setup_required"`
}

func (h *Handler) SetupStatus(c echo.Context) error {
	status, err := h.users.SetupStatus(c.Request().Context())
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, setupStatusResponse{SetupRequired: status.SetupRequired})
}
