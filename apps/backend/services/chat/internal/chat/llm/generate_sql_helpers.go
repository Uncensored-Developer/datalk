package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
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
				"type": []string{"array", "null"},
				"items": map[string]any{
					"type": "string",
				},
				"description": "Any assumptions made because the schema context was incomplete or ambiguous.",
			},
			"confidence": map[string]any{
				"type":        []string{"number", "null"},
				"description": "A confidence score from 0 to 1.",
			},
		},
		"required":             []string{"sql", "explanation", "assumptions", "confidence"},
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
	if req.Correction != nil && len(req.Correction.Attempts) > 0 {
		builder.WriteString("\nPrevious SQL attempts failed. Use the database error details to correct the SQL.\n")
		if req.Correction.AttemptNumber > 0 {
			builder.WriteString(fmt.Sprintf("Correction attempt: %d\n", req.Correction.AttemptNumber))
		}
		builder.WriteString(formatCorrectionAttempts(req.Correction.Attempts))
		builder.WriteString("\nReturn a corrected query that still follows every SQL safety rule.\n")
	}

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
		explanation := strings.TrimSpace(payload.Explanation)
		if explanation != "" {
			return nil, xerrors.Newf("%s: %w", explanation, chaterrors.ErrInvalidSQL)
		}
		return nil, xerrors.Newf("the model could not generate SQL for this question: %w", chaterrors.ErrInvalidSQL)
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

func formatCorrectionAttempts(attempts []llmtypes.SQLCorrectionAttempt) string {
	if len(attempts) == 0 {
		return "- No failed SQL attempts were provided."
	}

	lines := make([]string, 0, len(attempts)*3)
	for index, attempt := range attempts {
		sql := strings.TrimSpace(attempt.SQL)
		errorText := strings.TrimSpace(attempt.Error)
		if sql == "" && errorText == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("Failed attempt %d:", index+1))
		if sql != "" {
			lines = append(lines, fmt.Sprintf("SQL: %s", sql))
		}
		if errorText != "" {
			lines = append(lines, fmt.Sprintf("Error: %s", errorText))
		}
	}
	if len(lines) == 0 {
		return "- No failed SQL attempts were provided."
	}

	return strings.Join(lines, "\n")
}
