package conversations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	echohandlers "github.com/Uncensored-Developer/datalk/apps/backend/servers/echo/handlers"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/labstack/echo/v4"
)

func (h *Handler) SendMessageStream(c echo.Context) error {
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

	res := c.Response()
	streamStarted := false

	writeEvent := func(event string, payload any) error {
		if !streamStarted {
			res.Header().Set(echo.HeaderContentType, "text/event-stream")
			res.Header().Set(echo.HeaderCacheControl, "no-cache")
			res.Header().Set("Connection", "keep-alive")
			res.Header().Set("X-Accel-Buffering", "no")
			res.WriteHeader(http.StatusOK)
			streamStarted = true
		}
		if err := writeSSEEvent(res, event, payload); err != nil {
			return err
		}
		res.Flush()
		return nil
	}

	turn, err := h.service.SendMessageWithProgress(
		c.Request().Context(),
		sendMessageParams(userID, conversationID, req),
		func(progress chattype.SendMessageProgress) error {
			return writeEvent("progress", progress)
		},
	)
	if err != nil {
		if !streamStarted {
			return echohandlers.Error(c, h.logger, err)
		}
		return writeEvent("error", streamErrorResponse{
			Error:  echohandlers.MessageForError(err),
			Status: echohandlers.StatusForError(err),
		})
	}

	return writeEvent("final", toAssistantTurnResponse(turn))
}

func sendMessageParams(userID int32, conversationID int64, req sendMessageRequest) chattype.SendMessageParams {
	return chattype.SendMessageParams{
		UserID:                 userID,
		ConversationID:         conversationID,
		Content:                req.Content,
		Provider:               req.Provider,
		Model:                  req.Model,
		RequireNaturalResponse: req.RequireNaturalResponse,
	}
}

func writeSSEEvent(res *echo.Response, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(res, "event: %s\ndata: %s\n\n", event, data)
	return err
}
