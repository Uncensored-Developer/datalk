package conversations

import (
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
)

func toConversationResponses(conversations []*chattype.Conversation) []conversationResponse {
	out := make([]conversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation == nil {
			continue
		}
		out = append(out, toConversationResponse(conversation))
	}

	return out
}

func toConversationResponse(conversation *chattype.Conversation) conversationResponse {
	if conversation == nil {
		return conversationResponse{}
	}

	return conversationResponse{
		ID:           conversation.ID,
		UserID:       conversation.UserID,
		ConnectionID: conversation.ConnectionID,
		Title:        conversation.Title,
		CreatedAt:    conversation.CreatedAt,
		UpdatedAt:    conversation.UpdatedAt,
	}
}

func toMessageDetailsResponses(details []*chattype.MessageDetails) []messageDetailsResponse {
	out := make([]messageDetailsResponse, 0, len(details))
	for _, detail := range details {
		if detail == nil || detail.Message == nil {
			continue
		}
		out = append(out, toMessageDetailsResponse(detail))
	}

	return out
}

func toMessageDetailsResponse(detail *chattype.MessageDetails) messageDetailsResponse {
	return messageDetailsResponse{
		Message:   toMessageResponse(detail.Message),
		Execution: toExecutionResponse(detail.Execution),
		Retrieval: toRetrievalResponse(detail.Retrieval),
	}
}

func toAssistantTurnResponse(turn *chattype.AssistantTurn) assistantTurnResponse {
	if turn == nil {
		return assistantTurnResponse{}
	}

	return assistantTurnResponse{
		Conversation:     toConversationResponse(turn.Conversation),
		UserMessage:      toMessageResponse(turn.UserMessage),
		AssistantMessage: toMessageResponse(turn.AssistantMessage),
		Execution:        toExecutionResponse(turn.Execution),
		Retrieval:        toRetrievalResponse(turn.Retrieval),
	}
}

func toMessageResponse(message *chattype.Message) messageResponse {
	if message == nil {
		return messageResponse{}
	}

	return messageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		Role:           message.Role,
		Content:        message.Content,
		Provider:       message.Provider,
		Model:          message.Model,
		Status:         message.Status,
		ErrorMessage:   message.ErrorMessage,
		CreatedAt:      message.CreatedAt,
	}
}

func toExecutionResponse(execution *chattype.MessageExecution) *executionResponse {
	if execution == nil {
		return nil
	}

	return &executionResponse{
		MessageID:          execution.MessageID,
		ConnectionID:       execution.ConnectionID,
		DatabaseKind:       string(execution.DatabaseKind),
		GeneratedSQL:       execution.GeneratedSQL,
		NormalizedSQL:      execution.NormalizedSQL,
		Result:             execution.Result,
		ExecutionLatencyMS: execution.ExecutionLatencyMS,
		ExecutedAt:         execution.ExecutedAt,
	}
}

func toRetrievalResponse(retrieval *chattype.MessageRetrieval) *retrievalResponse {
	if retrieval == nil {
		return nil
	}

	return &retrievalResponse{
		MessageID:   retrieval.MessageID,
		SnapshotID:  retrieval.SnapshotID,
		QueryText:   retrieval.QueryText,
		Chunks:      toRetrievedChunkResponses(retrieval.Chunks),
		RetrievedAt: retrieval.RetrievedAt,
	}
}

func toRetrievedChunkResponses(chunks []schematypes.RetrievedChunk) []retrievedChunkResponse {
	out := make([]retrievedChunkResponse, 0, len(chunks))
	for _, chunk := range chunks {
		out = append(out, retrievedChunkResponse{
			ChunkID:    chunk.ChunkID,
			ObjectType: chunk.ObjectType,
			ObjectName: chunk.ObjectName,
			Content:    chunk.Content,
			SchemaJSON: chunk.SchemaJSON,
			Similarity: chunk.Similarity,
		})
	}

	return out
}
