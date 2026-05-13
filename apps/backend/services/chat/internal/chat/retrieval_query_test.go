package chat

import (
	"strings"
	"testing"

	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/stretchr/testify/assert"
)

func TestBuildRetrievalQuery_StandalonePrompt(t *testing.T) {
	t.Parallel()

	got := buildRetrievalQuery(
		"  how many users subscribed this month  ",
		[]llmtypes.ConversationMessage{
			{Role: "user", Content: "show me active users"},
			{Role: "assistant", Content: "Here is the count"},
		},
		nil,
	)

	assert.Equal(t, "how many users subscribed this month", got)
}

func TestBuildRetrievalQuery_FollowUpPrompt(t *testing.T) {
	t.Parallel()

	lastSQL := "SELECT COUNT(*) AS subscriber_count FROM users WHERE subscribed_at >= date_trunc('month', now())"

	got := buildRetrievalQuery(
		"group that by week",
		[]llmtypes.ConversationMessage{
			{Role: "user", Content: "how many users subscribed this month"},
			{Role: "assistant", Content: "There are 42 subscribers this month."},
		},
		&lastSQL,
	)

	assert.Equal(
		t,
		"Previous user question: how many users subscribed this month\nPrevious SQL: SELECT COUNT(*) AS subscriber_count FROM users WHERE subscribed_at >= date_trunc('month', now())\nCurrent follow-up question: group that by week",
		got,
	)
}

func TestBuildRetrievalQuery_SparseHistoryFallsBackToCurrentMessage(t *testing.T) {
	t.Parallel()

	got := buildRetrievalQuery("group that by week", nil, nil)

	assert.Equal(t, "group that by week", got)
}

func TestBuildRetrievalQuery_MissingPriorSQLStillUsesPreviousQuestion(t *testing.T) {
	t.Parallel()

	got := buildRetrievalQuery(
		"what about only active ones",
		[]llmtypes.ConversationMessage{
			{Role: "assistant", Content: "I can help with that."},
			{Role: "user", Content: "how many users subscribed this month"},
		},
		nil,
	)

	assert.Equal(
		t,
		"Previous user question: how many users subscribed this month\nCurrent follow-up question: what about only active ones",
		got,
	)
}

func TestBuildRetrievalQuery_TruncatesLongPriorContext(t *testing.T) {
	t.Parallel()

	longQuestion := strings.Repeat("question ", 50)
	longSQL := strings.Repeat("select_column ", 50)

	got := buildRetrievalQuery(
		"how about that by plan",
		[]llmtypes.ConversationMessage{{Role: "user", Content: longQuestion}},
		&longSQL,
	)

	assert.Contains(t, got, "Previous user question: ")
	assert.Contains(t, got, "Previous SQL: ")
	assert.Contains(t, got, "...")
	assert.Contains(t, got, "Current follow-up question: how about that by plan")
}
