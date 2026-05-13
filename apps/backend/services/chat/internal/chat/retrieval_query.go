package chat

import (
	"strings"

	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

const (
	maxPreviousQuestionLength = 240
	maxPreviousSQLLength      = 240

	previousQuestionLabel = "Previous user question: "
	previousSQLLabel      = "Previous SQL: "
	currentFollowUpLabel  = "Current follow-up question: "
)

// Single-word phrases intentionally include surrounding spaces because
// looksLikeFollowUp pads the normalized input before substring matching.
// This gives us an inexpensive word-boundary check and avoids false positives like
// matching "it" inside longer words such as "audit".
var followUpPhrases = []string{
	" that ",
	" those ",
	" them ",
	" it ",
	" same ",
	" also ",
	" instead ",
	" previous ",
	" above ",
	"what about",
	"how about",
}

func buildRetrievalQuery(currentMessage string, history []llmtypes.ConversationMessage, lastAssistantSQL *string) string {
	currentMessage = normalizeRetrievalText(currentMessage)
	if currentMessage == "" {
		return ""
	}
	if !looksLikeFollowUp(currentMessage) {
		return currentMessage
	}

	previousQuestion := lastUserQuestion(history)
	previousSQL := normalizeRetrievalText(derefStringOrEmpty(lastAssistantSQL))
	if previousQuestion == "" && previousSQL == "" {
		return currentMessage
	}

	parts := buildFollowUpContextParts(currentMessage, previousQuestion, previousSQL)
	return strings.Join(parts, "\n")
}

func buildFollowUpContextParts(currentMessage, previousQuestion, previousSQL string) []string {
	parts := make([]string, 0, 3)

	if previousQuestion != "" {
		parts = append(parts, previousQuestionLabel+truncateRetrievalContext(previousQuestion, maxPreviousQuestionLength))
	}
	if previousSQL != "" {
		parts = append(parts, previousSQLLabel+truncateRetrievalContext(previousSQL, maxPreviousSQLLength))
	}

	parts = append(parts, currentFollowUpLabel+currentMessage)
	return parts
}

func lastUserQuestion(history []llmtypes.ConversationMessage) string {
	for i := len(history) - 1; i >= 0; i-- {
		message := history[i]
		if !strings.EqualFold(message.Role, "user") {
			continue
		}
		content := normalizeRetrievalText(message.Content)
		if content != "" {
			return content
		}
	}
	return ""
}

func looksLikeFollowUp(message string) bool {
	normalized := " " + strings.ToLower(normalizeRetrievalText(message)) + " "
	for _, marker := range followUpPhrases {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// normalizeRetrievalText standardizes the input text by trimming spaces, normalizing whitespace, and collapsing it into a single space.
func normalizeRetrievalText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func truncateRetrievalContext(text string, maxLength int) string {
	if len(text) <= maxLength || maxLength <= 3 {
		return text
	}
	return strings.TrimSpace(text[:maxLength-3]) + "..."
}

func derefStringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
