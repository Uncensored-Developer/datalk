package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGenerateConversationTitleResponse(t *testing.T) {
	t.Parallel()

	resp, err := ParseGenerateConversationTitleResponse(
		[]byte(`{"request":true}`),
		[]byte(`{"response":true}`),
		`{"title":"  \"Revenue Growth?\"  "}`,
		nil,
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Revenue Growth", resp.Title)
	assert.JSONEq(t, `{"request":true}`, string(resp.RawRequest))
	assert.JSONEq(t, `{"response":true}`, string(resp.RawResponse))
}

func TestCleanConversationTitleTruncates(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "A Very Long", CleanConversationTitle("A Very Long Revenue Growth Question", 11))
}
