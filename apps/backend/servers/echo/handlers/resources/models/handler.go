package models

import (
	"log/slog"

	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service chatapi.Client
	logger  *slog.Logger
}

func New(service chatapi.Client, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		service: service,
		logger:  logger.With("resource", "models"),
	}
}

func (h *Handler) Register(group *echo.Group) {
	group.GET("/models", h.ListModels)
}
