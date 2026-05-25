package schemas

import (
	"log/slog"

	schemasapi "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/api"
	"github.com/labstack/echo/v4"
)

const connectionIDParam = "connection_id"

type Handler struct {
	schemas *schemasapi.Api
	logger  *slog.Logger
}

func New(schemas *schemasapi.Api, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		schemas: schemas,
		logger:  logger.With("resource", "schemas"),
	}
}

func (h *Handler) Register(group *echo.Group) {
	group.POST("/connections/:"+connectionIDParam+"/schema-snapshot/refresh", h.RefreshSchemaSnapshot)
}
