package chat

import (
	"encoding/json"

	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
)

const defaultMaxSchemaChunkContentBytes = 8 * 1024

func enforceGenerateSQLRequestLimits(req *llmtypes.GenerateSQLRequest) error {
	if req == nil {
		return nil
	}

	if req.Options.MaxSchemaChunks <= 0 {
		req.Options.MaxSchemaChunks = defaultSchemaChunkLimit
	}
	if req.Options.MaxPromptBytes <= 0 {
		req.Options.MaxPromptBytes = defaultMaxPromptBytes
	}

	if len(req.Schema.Chunks) > req.Options.MaxSchemaChunks {
		req.Schema.Chunks = req.Schema.Chunks[:req.Options.MaxSchemaChunks]
	}
	for index := range req.Schema.Chunks {
		req.Schema.Chunks[index].Content = truncateStringBytes(req.Schema.Chunks[index].Content, defaultMaxSchemaChunkContentBytes)
	}

	for estimateGenerateSQLRequestBytes(*req) > req.Options.MaxPromptBytes && len(req.Schema.Chunks) > 0 {
		req.Schema.Chunks = req.Schema.Chunks[:len(req.Schema.Chunks)-1]
	}
	if estimateGenerateSQLRequestBytes(*req) > req.Options.MaxPromptBytes {
		return xerrors.Newf("sql generation prompt exceeds %d bytes: %w", req.Options.MaxPromptBytes, chaterrors.ErrPromptTooLarge)
	}

	return nil
}

func estimateGenerateSQLRequestBytes(req llmtypes.GenerateSQLRequest) int {
	payload, err := json.Marshal(req)
	if err != nil {
		return len(req.UserPrompt)
	}

	return len(payload)
}

func truncateStringBytes(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}

	return string([]byte(value)[:maxBytes])
}
