package models

import (
	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service chatapi.Client
}

func New(service chatapi.Client) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(group *echo.Group) {
	group.GET("/models", h.ListModels)
}
