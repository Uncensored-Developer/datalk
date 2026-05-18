package conversations

import (
	chatapi "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/api"
	"github.com/labstack/echo/v4"
)

const conversationIDParam = "conversation_id"

type Handler struct {
	service chatapi.Client
}

func New(service chatapi.Client) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(group *echo.Group) {
	group.POST("/conversations", h.CreateConversation)
	group.GET("/conversations", h.ListConversations)
	group.GET("/conversations/:"+conversationIDParam, h.GetConversation)
	group.GET("/conversations/:"+conversationIDParam+"/messages", h.ListMessages)
	group.POST("/conversations/:"+conversationIDParam+"/messages", h.SendMessage)
}
