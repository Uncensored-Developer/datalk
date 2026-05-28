package conversations

import (
	"net/http"
	"strings"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/labstack/echo/v4"
)

func (h *Handler) SendMessage(c echo.Context) error {
	userID, err := echohandlers.UserID(c)
	if err != nil {
		return err
	}

	conversationID, err := echohandlers.Int64Param(c, conversationIDParam)
	if err != nil {
		return err
	}

	var req sendMessageRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "invalid request body"})
	}
	if strings.TrimSpace(req.Content) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, echohandlers.ErrorResponse{Error: "message content is required"})
	}

	turn, err := h.service.SendMessage(c.Request().Context(), chattype.SendMessageParams{
		UserID:                 userID,
		ConversationID:         conversationID,
		Content:                req.Content,
		Provider:               req.Provider,
		Model:                  req.Model,
		RequireNaturalResponse: req.RequireNaturalResponse,
	})
	if err != nil {
		return echohandlers.Error(c, h.logger, err)
	}

	return c.JSON(http.StatusOK, toAssistantTurnResponse(turn))
}
