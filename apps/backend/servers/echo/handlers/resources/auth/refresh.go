package auth

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	"github.com/labstack/echo/v4"
)

func (h *Handler) Refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	session, err := h.users.Refresh(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusOK, sessionResponse(session))
}
