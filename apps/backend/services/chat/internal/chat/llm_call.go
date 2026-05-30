package chat

import (
	"time"

	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
)

func buildLLMCall(
	messageID int64,
	resolved *chatllm.ResolvedClient,
	req llmtypes.GenerateSQLRequest,
	resp *llmtypes.GenerateSQLResponse,
	latencyMS int32,
) *chattype.MessageLLMCall {
	inputTokens, outputTokens := usageTokenPointers(resp.Usage)

	return &chattype.MessageLLMCall{
		MessageID:        messageID,
		ProviderConfigID: resolved.ProviderConfig.ID,
		Provider:         resolved.ProviderConfig.Provider,
		Model:            req.Model,
		RequestJSON:      redactSensitiveJSON(resp.RawRequest),
		ResponseJSON:     redactSensitiveJSON(resp.RawResponse),
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		LatencyMS:        latencyMS,
		CreatedAt:        time.Now().UTC(),
	}
}

func buildAnswerLLMCall(
	messageID int64,
	resolved *chatllm.ResolvedClient,
	req llmtypes.GenerateAnswerRequest,
	resp *llmtypes.GenerateAnswerResponse,
	latencyMS int32,
) *chattype.MessageLLMCall {
	inputTokens, outputTokens := usageTokenPointers(resp.Usage)

	return &chattype.MessageLLMCall{
		MessageID:        messageID,
		ProviderConfigID: resolved.ProviderConfig.ID,
		Provider:         resolved.ProviderConfig.Provider,
		Model:            req.Model,
		RequestJSON:      redactSensitiveJSON(resp.RawRequest),
		ResponseJSON:     redactSensitiveJSON(resp.RawResponse),
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		LatencyMS:        latencyMS,
		CreatedAt:        time.Now().UTC(),
	}
}

func usageTokenPointers(usage *llmtypes.Usage) (*int32, *int32) {
	if usage == nil {
		return nil, nil
	}

	return intPtrToInt32Ptr(usage.InputTokens), intPtrToInt32Ptr(usage.OutputTokens)
}

func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func elapsedMilliseconds(startedAt time.Time) int32 {
	return int32(time.Since(startedAt).Milliseconds())
}
