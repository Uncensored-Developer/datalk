package connections

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/labstack/echo/v4"
)

type testConnectionRequest struct {
	Database string `json:"database"`
	DSN      string `json:"dsn"`
}

type testConnectionResponse struct {
	OK bool `json:"ok"`
}

func (h *Handler) TestConnection(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	var req testConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	if err := h.connections.TestConnection(c.Request().Context(), connectionsapi.TestConnectionParams{
		Database: connectiontypes.Database(req.Database),
		DSN:      req.DSN,
	}); err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, testConnectionResponse{OK: true})
}
