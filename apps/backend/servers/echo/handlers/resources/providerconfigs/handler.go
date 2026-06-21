package providerconfigs

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
		logger:  logger.With("resource", "provider-configs"),
	}
}

func (h *Handler) Register(group *echo.Group) {
	group.GET("/provider-configs", h.ListProviderConfigs)
	group.POST("/provider-configs/:provider/test", h.TestProviderConfig)
	group.PUT("/provider-configs/:provider", h.SaveProviderConfig)
}
