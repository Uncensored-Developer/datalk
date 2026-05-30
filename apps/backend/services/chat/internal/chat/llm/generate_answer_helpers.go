package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
)

type generateAnswerPayload struct {
	Answer      string   `json:"answer"`
	Limitations []string `json:"limitations,omitempty"`
}

func GenerateAnswerSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"answer": map[string]any{
				"type":        "string",
				"description": "A concise natural language answer based only on the executed SQL result.",
			},
			"limitations": map[string]any{
				"type": []string{"array", "null"},
				"items": map[string]any{
					"type": "string",
				},
				"description": "Important caveats when the result is empty, truncated, ambiguous, or insufficient.",
			},
		},
		"required":             []string{"answer", "limitations"},
		"additionalProperties": false,
	}
}

func GenerateAnswerSystemPrompt(req llmtypes.GenerateAnswerRequest) string {
	var builder strings.Builder

	builder.WriteString("You write concise natural language answers from executed SQL results.\n")
	builder.WriteString("Return only structured data that matches the requested schema.\n")
	builder.WriteString("Use only the provided SQL result. Do not invent facts, rows, columns, units, or causes.\n")
	builder.WriteString("If the result is empty, truncated, or insufficient, state only what can be concluded and include limitations.\n")
	builder.WriteString("Do not include a markdown table; the table is already available to the user.\n\n")
	builder.WriteString(fmt.Sprintf("Database dialect: %s\n", req.DatabaseKind))
	if sql := strings.TrimSpace(req.GeneratedSQL); sql != "" {
		builder.WriteString("Executed SQL:\n")
		builder.WriteString(sql)
		builder.WriteString("\n")
	}
	builder.WriteString("SQL result preview:\n")
	builder.WriteString(formatQueryResultPreview(req.Result))

	return builder.String()
}

func GenerateAnswerMessages(req llmtypes.GenerateAnswerRequest) []PromptMessage {
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

func ParseGenerateAnswerResponse(rawRequest, rawResponse []byte, payloadText string, usage *llmtypes.Usage, finishReason *string) (*llmtypes.GenerateAnswerResponse, error) {
	var payload generateAnswerPayload
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		return nil, xerrors.Newf("failed to decode structured answer payload: %w", err)
	}

	payload.Answer = strings.TrimSpace(payload.Answer)
	if payload.Answer == "" {
		return nil, xerrors.New("structured answer payload did not include answer")
	}

	return &llmtypes.GenerateAnswerResponse{
		Answer:       payload.Answer,
		Limitations:  payload.Limitations,
		FinishReason: finishReason,
		Usage:        usage,
		RawRequest:   append([]byte(nil), rawRequest...),
		RawResponse:  append([]byte(nil), rawResponse...),
	}, nil
}

func BuildQueryResultPreview(result chattype.QueryResult, maxRows int, maxBytes int) llmtypes.QueryResultPreview {
	columns := make([]llmtypes.QueryResultColumn, 0, len(result.Columns))
	for _, column := range result.Columns {
		columns = append(columns, llmtypes.QueryResultColumn{
			Name:     column.Name,
			DataType: column.DataType,
		})
	}

	rowLimit := len(result.Rows)
	if maxRows > 0 && maxRows < rowLimit {
		rowLimit = maxRows
	}

	rows := make([]map[string]any, 0, rowLimit)
	truncated := result.Truncated || rowLimit < len(result.Rows)
	for _, row := range result.Rows[:rowLimit] {
		nextRows := append(rows, cloneResultRow(row))
		if maxBytes > 0 && exceedsJSONBytes(nextRows, maxBytes) {
			truncated = true
			break
		}
		rows = nextRows
	}

	return llmtypes.QueryResultPreview{
		Columns:   columns,
		Rows:      rows,
		RowCount:  result.RowCount,
		Truncated: truncated,
		Kind:      string(result.Kind),
	}
}

func formatQueryResultPreview(preview llmtypes.QueryResultPreview) string {
	payload, err := json.MarshalIndent(preview, "", "  ")
	if err != nil {
		return fmt.Sprintf("columns=%d rows=%d row_count=%d truncated=%t kind=%s", len(preview.Columns), len(preview.Rows), preview.RowCount, preview.Truncated, preview.Kind)
	}

	return string(payload)
}

func cloneResultRow(row map[string]any) map[string]any {
	if row == nil {
		return nil
	}

	clone := make(map[string]any, len(row))
	for key, value := range row {
		clone[key] = value
	}

	return clone
}

func exceedsJSONBytes(rows []map[string]any, maxBytes int) bool {
	payload, err := json.Marshal(rows)
	if err != nil {
		return true
	}

	return len(payload) > maxBytes
}
