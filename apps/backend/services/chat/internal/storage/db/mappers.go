package db

import (
	"encoding/json"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/gotidy/ptr"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob/types"
)

func conversationToDB(conversation *chattype.Conversation) *models.ChatConversationSetter {
	var createdAt omit.Val[time.Time]
	var updatedAt omit.Val[time.Time]

	if !conversation.CreatedAt.IsZero() {
		createdAt = omit.From(conversation.CreatedAt)
	}
	if !conversation.UpdatedAt.IsZero() {
		updatedAt = omit.From(conversation.UpdatedAt)
	}

	return &models.ChatConversationSetter{
		UserID:       omit.From(conversation.UserID),
		ConnectionID: omit.From(conversation.ConnectionID),
		Title:        omitnull.FromPtr(conversation.Title),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func conversationFromDB(dbConversation *models.ChatConversation) (*chattype.Conversation, error) {
	var title *string
	if val, ok := dbConversation.Title.Get(); ok {
		title = &val
	}

	return &chattype.Conversation{
		ID:           dbConversation.ID,
		UserID:       dbConversation.UserID,
		ConnectionID: dbConversation.ConnectionID,
		Title:        title,
		CreatedAt:    dbConversation.CreatedAt,
		UpdatedAt:    dbConversation.UpdatedAt,
	}, nil
}

func conversationsFromDB(dbConversations []*models.ChatConversation) ([]*chattype.Conversation, error) {
	return slices.Map(dbConversations, conversationFromDB)
}

func messageToDB(message *chattype.Message) *models.ChatMessageSetter {
	var provider omitnull.Val[string]
	var createdAt omit.Val[time.Time]

	if message.Provider != nil {
		value := string(*message.Provider)
		provider = omitnull.From(value)
	}
	if !message.CreatedAt.IsZero() {
		createdAt = omit.From(message.CreatedAt)
	}

	return &models.ChatMessageSetter{
		ConversationID: omit.From(message.ConversationID),
		Role:           omit.From(string(message.Role)),
		Content:        omit.From(message.Content),
		Provider:       provider,
		Model:          omitnull.FromPtr(message.Model),
		Status:         omit.From(string(message.Status)),
		ErrorMessage:   omitnull.FromPtr(message.ErrorMessage),
		CreatedAt:      createdAt,
	}
}

func messageFromDB(dbMessage *models.ChatMessage) (*chattype.Message, error) {
	var provider *llm.Provider
	if val, ok := dbMessage.Provider.Get(); ok {
		p := llm.Provider(val)
		provider = &p
	}

	var model *string
	if val, ok := dbMessage.Model.Get(); ok {
		model = &val
	}

	var errorMessage *string
	if val, ok := dbMessage.ErrorMessage.Get(); ok {
		errorMessage = &val
	}

	return &chattype.Message{
		ID:             dbMessage.ID,
		ConversationID: dbMessage.ConversationID,
		Role:           chattype.MessageRole(dbMessage.Role),
		Content:        dbMessage.Content,
		Provider:       provider,
		Model:          model,
		Status:         chattype.MessageStatus(dbMessage.Status),
		ErrorMessage:   errorMessage,
		CreatedAt:      dbMessage.CreatedAt,
	}, nil
}

func messagesFromDB(dbMessages []*models.ChatMessage) ([]*chattype.Message, error) {
	return slices.Map(dbMessages, messageFromDB)
}

func executionToDB(execution *chattype.MessageExecution) (*models.ChatMessageExecutionSetter, error) {
	var executedAt omit.Val[time.Time]
	if !execution.ExecutedAt.IsZero() {
		executedAt = omit.From(execution.ExecutedAt)
	}

	resultJSON, err := marshalExecutionResult(execution.Result)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal execution result: %w", err)
	}

	return &models.ChatMessageExecutionSetter{
		MessageID:          omit.From(execution.MessageID),
		ConnectionID:       omit.From(execution.ConnectionID),
		DatabaseKind:       omit.From(string(execution.DatabaseKind)),
		GeneratedSQL:       omit.From(execution.GeneratedSQL),
		NormalizedSQL:      omit.From(execution.NormalizedSQL),
		ResultJSON:         omit.From(types.NewJSON(resultJSON)),
		ExecutionLatencyMS: omit.From(execution.ExecutionLatencyMS),
		ExecutedAt:         executedAt,
	}, nil
}

func executionFromDB(dbExecution *models.ChatMessageExecution) (*chattype.MessageExecution, error) {
	result, err := unmarshalExecutionResult(dbExecution.ResultJSON.Val)
	if err != nil {
		return nil, xerrors.Newf("failed to unmarshal execution result: %w", err)
	}

	return &chattype.MessageExecution{
		MessageID:          dbExecution.MessageID,
		ConnectionID:       dbExecution.ConnectionID,
		DatabaseKind:       connectiontypes.Database(dbExecution.DatabaseKind),
		GeneratedSQL:       dbExecution.GeneratedSQL,
		NormalizedSQL:      dbExecution.NormalizedSQL,
		Result:             result,
		ExecutionLatencyMS: dbExecution.ExecutionLatencyMS,
		ExecutedAt:         dbExecution.ExecutedAt,
	}, nil
}

func marshalExecutionResult(result chattype.QueryResult) (json.RawMessage, error) {
	return json.Marshal(result)
}

func unmarshalExecutionResult(raw json.RawMessage) (chattype.QueryResult, error) {
	if len(raw) == 0 {
		return chattype.QueryResult{}, nil
	}

	var result chattype.QueryResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return chattype.QueryResult{}, err
	}

	return result, nil
}

func retrievalToDB(retrieval *chattype.MessageRetrieval) (*models.ChatMessageRetrievalSetter, error) {
	var retrievedAt omit.Val[time.Time]
	if !retrieval.RetrievedAt.IsZero() {
		retrievedAt = omit.From(retrieval.RetrievedAt)
	}

	chunksJSON, err := json.Marshal(retrieval.Chunks)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal retrieval chunks: %w", err)
	}

	return &models.ChatMessageRetrievalSetter{
		MessageID:   omit.From(retrieval.MessageID),
		SnapshotID:  omit.From(retrieval.SnapshotID),
		QueryText:   omit.From(retrieval.QueryText),
		Chunks:      omit.From(types.NewJSON[json.RawMessage](chunksJSON)),
		RetrievedAt: retrievedAt,
	}, nil
}

func retrievalFromDB(dbRetrieval *models.ChatMessageRetrieval) (*chattype.MessageRetrieval, error) {
	var chunks []schematypes.RetrievedChunk
	if len(dbRetrieval.Chunks.Val) > 0 {
		if err := json.Unmarshal(dbRetrieval.Chunks.Val, &chunks); err != nil {
			return nil, xerrors.Newf("failed to unmarshal retrieval chunks: %w", err)
		}
	}

	return &chattype.MessageRetrieval{
		MessageID:   dbRetrieval.MessageID,
		SnapshotID:  dbRetrieval.SnapshotID,
		QueryText:   dbRetrieval.QueryText,
		Chunks:      chunks,
		RetrievedAt: dbRetrieval.RetrievedAt,
	}, nil
}

func llmCallToDB(call *chattype.MessageLLMCall) *models.ChatMessageLLMCallSetter {
	var createdAt omit.Val[time.Time]

	if !call.CreatedAt.IsZero() {
		createdAt = omit.From(call.CreatedAt)
	}

	requestJSON := call.RequestJSON
	if len(requestJSON) == 0 {
		requestJSON = []byte("{}")
	}
	responseJSON := call.ResponseJSON
	if len(responseJSON) == 0 {
		responseJSON = []byte("{}")
	}

	return &models.ChatMessageLLMCallSetter{
		MessageID:        omit.From(call.MessageID),
		ProviderConfigID: omit.From(call.ProviderConfigID),
		Provider:         omit.From(string(call.Provider)),
		Model:            omit.From(call.Model),
		RequestJSON:      omit.From(types.NewJSON(requestJSON)),
		ResponseJSON:     omit.From(types.NewJSON(responseJSON)),
		InputTokens:      omitnull.FromPtr(call.InputTokens),
		OutputTokens:     omitnull.FromPtr(call.OutputTokens),
		LatencyMS:        omit.From(call.LatencyMS),
		CreatedAt:        createdAt,
	}
}

func llmCallFromDB(dbCall *models.ChatMessageLLMCall) (*chattype.MessageLLMCall, error) {
	return &chattype.MessageLLMCall{
		ID:               dbCall.ID,
		MessageID:        dbCall.MessageID,
		ProviderConfigID: dbCall.ProviderConfigID,
		Provider:         llm.Provider(dbCall.Provider),
		Model:            dbCall.Model,
		RequestJSON:      dbCall.RequestJSON.Val,
		ResponseJSON:     dbCall.ResponseJSON.Val,
		InputTokens:      dbCall.InputTokens.Ptr(),
		OutputTokens:     dbCall.OutputTokens.Ptr(),
		LatencyMS:        dbCall.LatencyMS,
		CreatedAt:        dbCall.CreatedAt,
	}, nil
}

func llmCallsFromDB(dbCalls []*models.ChatMessageLLMCall) ([]*chattype.MessageLLMCall, error) {
	return slices.Map(dbCalls, llmCallFromDB)
}

func providerConfigToDB(config *llm.ProviderConfig) *models.LLMProviderConfigSetter {
	var metadata omit.Val[types.JSON[json.RawMessage]]
	var createdAt omit.Val[time.Time]
	var updatedAt omit.Val[time.Time]

	if len(config.Metadata) > 0 {
		metadata = omit.From(types.NewJSON(config.Metadata))
	}
	if !config.CreatedAt.IsZero() {
		createdAt = omit.From(config.CreatedAt)
	}
	if !config.UpdatedAt.IsZero() {
		updatedAt = omit.From(config.UpdatedAt)
	}

	return &models.LLMProviderConfigSetter{
		Provider:    omit.From(string(config.Provider)),
		DisplayName: omit.From(config.DisplayName),
		APIKeyEnc:   omit.From(config.APIKeyEnc),
		BaseURL:     omitnull.FromPtr(config.BaseURL),
		IsEnabled:   omit.From(config.IsEnabled),
		Metadata:    metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func providerConfigFromDB(dbConfig *models.LLMProviderConfig) (*llm.ProviderConfig, error) {
	return &llm.ProviderConfig{
		ID:          dbConfig.ID,
		Provider:    llm.Provider(dbConfig.Provider),
		DisplayName: dbConfig.DisplayName,
		APIKeyEnc:   dbConfig.APIKeyEnc,
		BaseURL:     dbConfig.BaseURL.Ptr(),
		IsEnabled:   dbConfig.IsEnabled,
		Metadata:    dbConfig.Metadata.Val,
		CreatedAt:   dbConfig.CreatedAt,
		UpdatedAt:   dbConfig.UpdatedAt,
	}, nil
}

func providerConfigsFromDB(dbConfigs []*models.LLMProviderConfig) ([]*llm.ProviderConfig, error) {
	return slices.Map(dbConfigs, providerConfigFromDB)
}

func providerModelToDB(model *llm.ProviderModel) *models.LLMProviderModelSetter {
	var discoveredAt omit.Val[time.Time]

	if !model.DiscoveredAt.IsZero() {
		discoveredAt = omit.From(model.DiscoveredAt)
	}

	return &models.LLMProviderModelSetter{
		ProviderConfigID: omit.From(model.ProviderConfigID),
		Model:            omit.From(model.Model),
		DisplayName:      omit.From(model.DisplayName),
		ContextWindow:    omitnull.FromPtr(model.ContextWindow),
		SupportsSQL:      omit.From(model.SupportsSQL),
		IsEnabled:        omit.From(model.IsEnabled),
		DiscoveredAt:     discoveredAt,
	}
}

func providerModelFromDB(dbModel *models.LLMProviderModel) (*llm.ProviderModel, error) {
	return &llm.ProviderModel{
		ID:               dbModel.ID,
		ProviderConfigID: dbModel.ProviderConfigID,
		Model:            dbModel.Model,
		DisplayName:      dbModel.DisplayName,
		ContextWindow:    ptr.Of(dbModel.ContextWindow.GetOrZero()),
		SupportsSQL:      dbModel.SupportsSQL,
		IsEnabled:        dbModel.IsEnabled,
		DiscoveredAt:     dbModel.DiscoveredAt,
	}, nil
}

func providerModelsFromDB(dbModels []*models.LLMProviderModel) ([]*llm.ProviderModel, error) {
	return slices.Map(dbModels, providerModelFromDB)
}
