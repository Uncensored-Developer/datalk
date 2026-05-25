package schemas

import (
	"math"
	"net/http"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	"github.com/labstack/echo/v4"
)

type refreshSchemaSnapshotResponse struct {
	ConnectionID int32  `json:"connection_id"`
	Status       string `json:"status"`
}

func (h *Handler) RefreshSchemaSnapshot(c echo.Context) error {
	if _, err := echohandlers.User(c); err != nil {
		return err
	}

	connectionID, err := echohandlers.Int64Param(c, connectionIDParam)
	if err != nil {
		return err
	}
	if connectionID > math.MaxInt32 {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid " + connectionIDParam})
	}

	if err := h.schemas.SnapshotDatabase(c.Request().Context(), int32(connectionID)); err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusAccepted, refreshSchemaSnapshotResponse{
		ConnectionID: int32(connectionID),
		Status:       "accepted",
	})
}
