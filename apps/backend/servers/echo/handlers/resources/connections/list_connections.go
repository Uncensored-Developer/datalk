package connections

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/labstack/echo/v4"
)

func (h *Handler) ListConnections(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}

	connections, err := h.connections.ListConnections(c.Request().Context(), connectionsapi.ListConnectionsParams{
		UserID:  user.ID,
		IsAdmin: user.IsAdmin(),
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, toConnectionResponses(connections))
}

func toConnectionResponses(connections []*connectiontypes.Connection) []connectionResponse {
	out := make([]connectionResponse, 0, len(connections))
	for _, connection := range connections {
		if connection == nil {
			continue
		}
		out = append(out, toConnectionResponse(connection))
	}

	return out
}
