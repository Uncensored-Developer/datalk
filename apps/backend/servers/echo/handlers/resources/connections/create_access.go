package connections

import (
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	usererrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/labstack/echo/v4"
)

type createAccessRequest struct {
	UserID      int32 `json:"user_id"`
	CanQuery    bool  `json:"can_query"`
	AllowWrites bool  `json:"allow_writes"`
	CanManage   bool  `json:"can_manage"`
}

type accessResponse struct {
	UserID       int32 `json:"user_id"`
	ConnectionID int32 `json:"connection_id"`
	CanQuery     bool  `json:"can_query"`
	AllowWrites  bool  `json:"allow_writes"`
	CanManage    bool  `json:"can_manage"`
}

func (h *Handler) CreateAccess(c echo.Context) error {
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

	var req createAccessRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request"})
	}

	access, err := h.connections.CreateAccess(c.Request().Context(), connectionsapi.NewAccessParams{
		UserID:       req.UserID,
		ConnectionID: int32(connectionID),
		CanQuery:     req.CanQuery,
		AllowWrites:  req.AllowWrites,
		CanManage:    req.CanManage,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}
	return c.JSON(http.StatusCreated, toAccessResponse(access))
}

func toAccessResponse(access *connectiontypes.Access) accessResponse {
	return accessResponse{
		UserID:       access.UserID,
		ConnectionID: access.ConnectionID,
		CanQuery:     access.CanQuery,
		AllowWrites:  access.AllowWrites,
		CanManage:    access.CanManage,
	}
}
