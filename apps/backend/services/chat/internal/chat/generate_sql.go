package chat

import (
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
)

func buildGenerateSQLRequest(
	conversation *chattype.Conversation,
	databaseKind connectiontypes.Database,
	userPrompt string,
	history []*chattype.Message,
	schemaContext *schematypes.RetrievedSchemaContext,
	modelID string,
) llmtypes.GenerateSQLRequest {
	return llmtypes.GenerateSQLRequest{
		Model: modelID,
		Conversation: llmtypes.ConversationContext{
			ConversationID: conversation.ID,
			ConnectionID:   conversation.ConnectionID,
			DatabaseKind:   databaseKind,
			History:        toConversationMessages(history),
		},
		UserPrompt: userPrompt,
		Schema:     *schemaContext,
		Options: llmtypes.GenerateSQLOptions{
			MaxHistoryMessages: defaultHistoryLimit,
			MaxSchemaChunks:    defaultSchemaChunkLimit,
			MaxPromptBytes:     defaultMaxPromptBytes,
			RequireReadOnly:    true,
			RequireSingleStmt:  true,
			AllowedDatabases:   []connectiontypes.Database{connectiontypes.DatabasePostgres, connectiontypes.DatabaseMySQL},
		},
	}
}
