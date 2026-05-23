package chat

import (
	"strings"
	"testing"

	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnforceGenerateSQLRequestLimits_TrimsSchemaContext(t *testing.T) {
	t.Parallel()

	req := llmtypes.GenerateSQLRequest{
		Model:      "gpt-5.2",
		UserPrompt: "how many users subscribed in May 2026?",
		Schema: schematypes.RetrievedSchemaContext{
			Chunks: []schematypes.RetrievedChunk{
				{Content: strings.Repeat("a", defaultMaxSchemaChunkContentBytes+100)},
				{Content: "second"},
				{Content: "third"},
			},
		},
		Options: llmtypes.GenerateSQLOptions{
			MaxSchemaChunks: 2,
			MaxPromptBytes:  defaultMaxPromptBytes,
		},
	}

	require.NoError(t, enforceGenerateSQLRequestLimits(&req))
	require.Len(t, req.Schema.Chunks, 2)
	assert.Len(t, req.Schema.Chunks[0].Content, defaultMaxSchemaChunkContentBytes)
	assert.Equal(t, "second", req.Schema.Chunks[1].Content)
}

func TestEnforceGenerateSQLRequestLimits_RejectsOversizedPrompt(t *testing.T) {
	t.Parallel()

	req := llmtypes.GenerateSQLRequest{
		Model:      "gpt-5.2",
		UserPrompt: strings.Repeat("x", 256),
		Options: llmtypes.GenerateSQLOptions{
			MaxSchemaChunks: 2,
			MaxPromptBytes:  32,
		},
	}

	err := enforceGenerateSQLRequestLimits(&req)

	require.ErrorIs(t, err, chaterrors.ErrPromptTooLarge)
}
