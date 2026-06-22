package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
)

type generateConversationTitlePayload struct {
	Title string `json:"title"`
}

func GenerateConversationTitleSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "A short title for the conversation.",
			},
		},
		"required":             []string{"title"},
		"additionalProperties": false,
	}
}

func GenerateConversationTitleSystemPrompt(req llmtypes.GenerateConversationTitleRequest) string {
	maxWords := req.MaxWords
	if maxWords <= 0 {
		maxWords = 6
	}
	maxChars := req.MaxChars
	if maxChars <= 0 {
		maxChars = 80
	}

	return fmt.Sprintf(
		"Generate a short conversation title from the user's first message.\n"+
			"Return only structured data that matches the requested schema.\n"+
			"Use %d words or fewer and %d characters or fewer.\n"+
			"Do not wrap the title in quotes.\n"+
			"Do not end with punctuation unless it is part of a proper noun.\n"+
			"Do not include labels such as Title:.",
		maxWords,
		maxChars,
	)
}

func GenerateConversationTitleMessages(req llmtypes.GenerateConversationTitleRequest) []PromptMessage {
	messages := []PromptMessage{{
		Role:    "user",
		Content: strings.TrimSpace(req.UserPrompt),
	}}
	if assistant := strings.TrimSpace(req.Assistant); assistant != "" {
		messages = append(messages, PromptMessage{
			Role:    "assistant",
			Content: assistant,
		})
	}

	return messages
}

func ParseGenerateConversationTitleResponse(rawRequest, rawResponse []byte, payloadText string, usage *llmtypes.Usage, finishReason *string) (*llmtypes.GenerateConversationTitleResponse, error) {
	var payload generateConversationTitlePayload
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		return nil, xerrors.Newf("failed to decode structured conversation title payload: %w", err)
	}

	title := CleanConversationTitle(payload.Title, 80)
	if title == "" {
		return nil, xerrors.New("structured conversation title payload did not include title")
	}

	return &llmtypes.GenerateConversationTitleResponse{
		Title:        title,
		FinishReason: finishReason,
		Usage:        usage,
		RawRequest:   append([]byte(nil), rawRequest...),
		RawResponse:  append([]byte(nil), rawResponse...),
	}, nil
}

func CleanConversationTitle(title string, maxChars int) string {
	title = strings.TrimSpace(title)
	title = strings.Trim(title, "\"'`“”‘’")
	title = strings.Join(strings.Fields(title), " ")
	title = strings.TrimSpace(title)
	title = strings.TrimRight(title, ".!?;:")
	if maxChars > 0 && len(title) > maxChars {
		title = strings.TrimSpace(title[:maxChars])
		title = strings.TrimRight(title, " ,.-_")
	}

	return title
}
