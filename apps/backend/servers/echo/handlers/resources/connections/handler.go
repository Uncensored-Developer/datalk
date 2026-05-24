package connections

import (
	"log/slog"

	connectionsapi "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	"github.com/labstack/echo/v4"
)

const connectionIDParam = "connection_id"

type Handler struct {
	connections *connectionsapi.Api
	logger      *slog.Logger
}

func New(connections *connectionsapi.Api, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		connections: connections,
		logger:      logger.With("resource", "connections"),
	}
}

func (h *Handler) Register(group *echo.Group) {
	group.POST("/connections", h.CreateConnection)
	group.POST("/connections/:"+connectionIDParam+"/access", h.CreateAccess)
}
