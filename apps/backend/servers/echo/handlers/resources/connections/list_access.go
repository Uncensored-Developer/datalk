package connections

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/labstack/echo/v4"
)

func (h *Handler) ListAccess(c echo.Context) error {
	user, err := echohandlers.User(c)
	if err != nil {
		return err
	}
	if !user.IsAdmin() {
		return echohandlers.Error(c, h.logger, usererrors.ErrForbidden)
	}

	connectionID, err := echohandlers.Int64Param(c, connectionIDParam)
	if err != nil {
		return err
	}

	access, err := h.connections.ListAccess(c.Request().Context(), connectionsapi.ListAccessParams{
		ConnectionID: []int32{int32(connectionID)},
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, toAccessResponses(access))
}

func toAccessResponses(access []*connectiontypes.Access) []accessResponse {
	out := make([]accessResponse, 0, len(access))
	for _, item := range access {
		if item == nil {
			continue
		}
		out = append(out, toAccessResponse(item))
	}

	return out
}
