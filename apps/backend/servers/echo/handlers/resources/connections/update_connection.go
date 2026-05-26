package connections

import (
	"math"
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/labstack/echo/v4"
)

type updateConnectionRequest struct {
	Name      *string                   `json:"name"`
	Database  *string                   `json:"database"`
	DSN       *string                   `json:"dsn"`
	IsEnabled *bool                     `json:"is_enabled"`
	Metadata  *connectiontypes.Metadata `json:"metadata"`
}

func (h *Handler) UpdateConnection(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	connectionID, err := connectionID(c)
	if err != nil {
		return err
	}

	var req updateConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	var database *connectiontypes.Database
	if req.Database != nil {
		value := connectiontypes.Database(*req.Database)
		database = &value
	}

	connection, err := h.connections.UpdateConnection(c.Request().Context(), connectionsapi.UpdateConnectionParams{
		ID:        connectionID,
		Name:      req.Name,
		Database:  database,
		DSN:       req.DSN,
		IsEnabled: req.IsEnabled,
		Metadata:  req.Metadata,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, toConnectionResponse(connection))
}

func (h *Handler) DeleteConnection(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	connectionID, err := connectionID(c)
	if err != nil {
		return err
	}

	if err := h.connections.DeleteConnection(c.Request().Context(), connectionID); err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func connectionID(c echo.Context) (int32, error) {
	rawID, err := echohandlers.Int64Param(c, connectionIDParam)
	if err != nil {
		return 0, err
	}
	if rawID > math.MaxInt32 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid " + connectionIDParam})
	}

	return int32(rawID), nil
}
