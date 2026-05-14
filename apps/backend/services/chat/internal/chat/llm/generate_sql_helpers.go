package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
)

type generateSQLPayload struct {
	SQL         string   `json:"sql"`
	Explanation string   `json:"explanation"`
	Assumptions []string `json:"assumptions,omitempty"`
	Confidence  *float32 `json:"confidence,omitempty"`
}

type PromptMessage struct {
	Role    string
	Content string
}

func GenerateSQLSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sql": map[string]any{
				"type":        "string",
				"description": "A single read-only SQL statement that answers the user's request.",
			},
			"explanation": map[string]any{
				"type":        "string",
				"description": "A brief explanation of what the SQL is doing.",
			},
			"assumptions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Any assumptions made because the schema context was incomplete or ambiguous.",
			},
			"confidence": map[string]any{
				"type":        "number",
				"description": "A confidence score from 0 to 1.",
			},
		},
		"required":             []string{"sql", "explanation"},
		"additionalProperties": false,
	}
}

func GenerateSQLSystemPrompt(req llmtypes.GenerateSQLRequest) string {
	var builder strings.Builder

	builder.WriteString("You generate a single read-only SQL query from the provided database context.\n")
	builder.WriteString("Return only structured data that matches the requested schema.\n")
	builder.WriteString("Do not invent tables or columns.\n\n")
	builder.WriteString("SQL safety rules:\n")
	builder.WriteString("- Generate exactly one SQL statement.\n")
	builder.WriteString("- Generate only SELECT queries or WITH queries that end in SELECT.\n")
	builder.WriteString("- Do not generate INSERT, UPDATE, DELETE, MERGE, DDL, transaction commands, temp objects, COPY, or procedure calls.\n")
	builder.WriteString("- If the schema context is insufficient, explain the missing context instead of guessing.\n\n")

	builder.WriteString(fmt.Sprintf("Database dialect: %s\n", req.Conversation.DatabaseKind))
	builder.WriteString("Retrieved schema context:\n")
	builder.WriteString(formatSchemaChunks(req.Schema.Chunks, req.Options.MaxSchemaChunks))
	builder.WriteString("\n")

	return builder.String()
}

func GenerateSQLMessages(req llmtypes.GenerateSQLRequest) []PromptMessage {
	history := trimHistory(req.Conversation.History, req.Options.MaxHistoryMessages)
	messages := make([]PromptMessage, 0, len(history)+1)

	for _, message := range history {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		content := strings.TrimSpace(message.Content)
		if role == "" || content == "" {
			continue
		}
		messages = append(messages, PromptMessage{
			Role:    role,
			Content: content,
		})
	}

	if userPrompt := strings.TrimSpace(req.UserPrompt); userPrompt != "" {
		messages = append(messages, PromptMessage{
			Role:    "user",
			Content: userPrompt,
		})
	}

	return messages
}

func ParseGenerateSQLResponse(rawRequest, rawResponse []byte, payloadText string, usage *llmtypes.Usage, finishReason *string) (*llmtypes.GenerateSQLResponse, error) {
	var payload generateSQLPayload
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		return nil, xerrors.Newf("failed to decode structured SQL payload: %w", err)
	}

	payload.SQL = strings.TrimSpace(payload.SQL)
	payload.Explanation = strings.TrimSpace(payload.Explanation)
	if payload.SQL == "" {
		return nil, xerrors.New("structured SQL payload did not include sql")
	}

	return &llmtypes.GenerateSQLResponse{
		SQL:          payload.SQL,
		Explanation:  payload.Explanation,
		Assumptions:  payload.Assumptions,
		Confidence:   payload.Confidence,
		FinishReason: finishReason,
		Usage:        usage,
		RawRequest:   append([]byte(nil), rawRequest...),
		RawResponse:  append([]byte(nil), rawResponse...),
	}, nil
}

func trimHistory(history []llmtypes.ConversationMessage, maxHistoryMessages int) []llmtypes.ConversationMessage {
	start := 0
	if maxHistoryMessages > 0 && len(history) > maxHistoryMessages {
		start = len(history) - maxHistoryMessages
	}

	return history[start:]
}

func formatSchemaChunks(chunks []schematypes.RetrievedChunk, maxSchemaChunks int) string {
	if len(chunks) == 0 {
		return "- No schema context was retrieved."
	}

	limit := len(chunks)
	if maxSchemaChunks > 0 && maxSchemaChunks < limit {
		limit = maxSchemaChunks
	}

	lines := make([]string, 0, limit*3)
	for _, chunk := range chunks[:limit] {
		lines = append(lines, fmt.Sprintf("- %s %s", chunk.ObjectType, chunk.ObjectName))
		if chunk.Content != "" {
			lines = append(lines, fmt.Sprintf("  %s", strings.TrimSpace(chunk.Content)))
		}
	}

	return strings.Join(lines, "\n")
}
