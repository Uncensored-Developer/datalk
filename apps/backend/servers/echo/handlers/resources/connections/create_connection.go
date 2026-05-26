package connections

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/labstack/echo/v4"
)

type createConnectionRequest struct {
	Name     string                   `json:"name"`
	Database string                   `json:"database"`
	DSN      string                   `json:"dsn"`
	Metadata connectiontypes.Metadata `json:"metadata"`
}

type connectionResponse struct {
	ID        int32                    `json:"id"`
	Name      string                   `json:"name"`
	Database  string                   `json:"database"`
	UserID    int32                    `json:"user_id"`
	IsEnabled bool                     `json:"is_enabled"`
	Metadata  connectiontypes.Metadata `json:"metadata"`
}

func (h *Handler) CreateConnection(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	var req createConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	connection, err := h.connections.CreateConnection(c.Request().Context(), connectionsapi.NewConnectionParams{
		Name:     req.Name,
		Database: connectiontypes.Database(req.Database),
		DSN:      req.DSN,
		UserID:   user.ID,
		Metadata: req.Metadata,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusCreated, toConnectionResponse(connection))
}

func toConnectionResponse(connection *connectiontypes.Connection) connectionResponse {
	return connectionResponse{
		ID:        connection.ID,
		Name:      connection.Name,
		Database:  string(connection.Database),
		UserID:    connection.UserID,
		IsEnabled: connection.IsEnabled,
		Metadata:  connection.Metadata,
	}
}
