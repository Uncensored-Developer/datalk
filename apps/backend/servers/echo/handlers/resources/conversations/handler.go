package conversations

import (
	"log/slog"

	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	"github.com/labstack/echo/v4"
)

const conversationIDParam = "conversation_id"

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
		logger:  logger.With("resource", "conversations"),
	}
}

func (h *Handler) Register(group *echo.Group) {
	group.POST("/conversations", h.CreateConversation)
	group.GET("/conversations", h.ListConversations)
	group.GET("/conversations/:"+conversationIDParam, h.GetConversation)
	group.DELETE("/conversations/:"+conversationIDParam, h.DeleteConversation)
	group.GET("/conversations/:"+conversationIDParam+"/messages", h.ListMessages)
	group.POST("/conversations/:"+conversationIDParam+"/messages", h.SendMessage)
	group.POST("/conversations/:"+conversationIDParam+"/messages/stream", h.SendMessageStream)
}
