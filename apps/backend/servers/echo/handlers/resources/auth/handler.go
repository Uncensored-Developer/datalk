package auth

import (
	"log/slog"

	usersapi "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	users  usersapi.Client
	logger *slog.Logger
}

func New(users usersapi.Client, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		users:  users,
		logger: logger.With("resource", "auth"),
	}
}

func (h *Handler) RegisterPublic(group *echo.Group) {
	group.GET("/auth/setup/status", h.SetupStatus)
	group.POST("/auth/setup", h.Setup)
	group.POST("/auth/login", h.Login)
	group.POST("/auth/refresh", h.Refresh)
}

func (h *Handler) RegisterProtected(group *echo.Group) {
	group.POST("/auth/logout", h.Logout)
}
