package db

import (
	"context"
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/info"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

type Storage struct {
	*common.Storage
}

func NewStorage(conn *sql.DB) *Storage {
	return &Storage{
		Storage: common.NewStorage("chat", conn),
	}
}

func (s *Storage) InsertConversation(ctx context.Context, conversation *chattype.Conversation) error {
	dbConversation, err := models.ChatConversations.Insert(conversationToDB(conversation)).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	inserted, err := conversationFromDB(dbConversation)
	if err != nil {
		return xerrors.Newf("failed to map db conversation: %w", err)
	}

	*conversation = *inserted
	return nil
}

func (s *Storage) GetConversation(ctx context.Context, id int64) (*chattype.Conversation, error) {
	dbConversation, err := models.FindChatConversation(ctx, s.Executor(ctx), id)
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch conversation: %w", err)
	}
	if dbConversation == nil {
		return nil, nil
	}

	conversation, err := conversationFromDB(dbConversation)
	if err != nil {
		return nil, xerrors.Newf("failed to map db conversation: %w", err)
	}

	return conversation, nil
}

func (s *Storage) ListConversations(ctx context.Context, filter chatstorage.ConversationsFilter) ([]*chattype.Conversation, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(filter.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatConversations.ID.In(filter.ID...))
	}
	if len(filter.UserID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatConversations.UserID.In(filter.UserID...))
	}
	if len(filter.ConnectionID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatConversations.ConnectionID.In(filter.ConnectionID...))
	}

	queryMods = append(queryMods, common.PaginationToBobMods(filter.Pagination)...)

	orderingMods, err := common.OrderingToBobMods(filter.Ordering, listConversationsOrderingExpr)
	if err != nil {
		return nil, xerrors.Newf("invalid conversations filter: %w", err)
	}
	queryMods = append(queryMods, orderingMods...)

	dbConversations, err := models.ChatConversations.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch conversations: %w", err)
	}

	return conversationsFromDB(dbConversations)
}

func (s *Storage) InsertMessage(ctx context.Context, message *chattype.Message) error {
	dbMessage, err := models.ChatMessages.Insert(messageToDB(message)).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	inserted, err := messageFromDB(dbMessage)
	if err != nil {
		return xerrors.Newf("failed to map db message: %w", err)
	}

	*message = *inserted
	return nil
}

func (s *Storage) GetMessage(ctx context.Context, id int64) (*chattype.Message, error) {
	dbMessage, err := models.FindChatMessage(ctx, s.Executor(ctx), id)
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch message: %w", err)
	}
	if dbMessage == nil {
		return nil, nil
	}

	message, err := messageFromDB(dbMessage)
	if err != nil {
		return nil, xerrors.Newf("failed to map db message: %w", err)
	}

	return message, nil
}

func (s *Storage) ListMessages(ctx context.Context, filter chatstorage.MessagesFilter) ([]*chattype.Message, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(filter.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatMessages.ID.In(filter.ID...))
	}
	if len(filter.ConversationID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatMessages.ConversationID.In(filter.ConversationID...))
	}
	if len(filter.Role) > 0 {
		roles := make([]string, 0, len(filter.Role))
		for _, role := range filter.Role {
			roles = append(roles, string(role))
		}
		queryMods = append(queryMods, models.SelectWhere.ChatMessages.Role.In(roles...))
	}
	if len(filter.Status) > 0 {
		statuses := make([]string, 0, len(filter.Status))
		for _, status := range filter.Status {
			statuses = append(statuses, string(status))
		}
		queryMods = append(queryMods, models.SelectWhere.ChatMessages.Status.In(statuses...))
	}

	queryMods = append(queryMods, common.PaginationToBobMods(filter.Pagination)...)

	orderingMods, err := common.OrderingToBobMods(filter.Ordering, listMessagesOrderingExpr)
	if err != nil {
		return nil, xerrors.Newf("invalid messages filter: %w", err)
	}
	queryMods = append(queryMods, orderingMods...)

	dbMessages, err := models.ChatMessages.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch messages: %w", err)
	}

	return messagesFromDB(dbMessages)
}

func (s *Storage) InsertExecution(ctx context.Context, execution *chattype.MessageExecution) error {
	executionSetter, err := executionToDB(execution)
	if err != nil {
		return err
	}

	dbExecution, err := models.ChatMessageExecutions.Insert(executionSetter).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	inserted, err := executionFromDB(dbExecution)
	if err != nil {
		return xerrors.Newf("failed to map db execution: %w", err)
	}

	*execution = *inserted
	return nil
}

func (s *Storage) GetExecution(ctx context.Context, messageID int64) (*chattype.MessageExecution, error) {
	dbExecution, err := models.FindChatMessageExecution(ctx, s.Executor(ctx), messageID)
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch execution: %w", err)
	}
	if dbExecution == nil {
		return nil, nil
	}

	execution, err := executionFromDB(dbExecution)
	if err != nil {
		return nil, xerrors.Newf("failed to map db execution: %w", err)
	}

	return execution, nil
}

func (s *Storage) InsertRetrieval(ctx context.Context, retrieval *chattype.MessageRetrieval) error {
	retrievalSetter, err := retrievalToDB(retrieval)
	if err != nil {
		return err
	}

	dbRetrieval, err := models.ChatMessageRetrievals.Insert(retrievalSetter).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	inserted, err := retrievalFromDB(dbRetrieval)
	if err != nil {
		return xerrors.Newf("failed to map db retrieval: %w", err)
	}

	*retrieval = *inserted
	return nil
}

func (s *Storage) GetRetrieval(ctx context.Context, messageID int64) (*chattype.MessageRetrieval, error) {
	dbRetrieval, err := models.FindChatMessageRetrieval(ctx, s.Executor(ctx), messageID)
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch retrieval: %w", err)
	}
	if dbRetrieval == nil {
		return nil, nil
	}

	retrieval, err := retrievalFromDB(dbRetrieval)
	if err != nil {
		return nil, xerrors.Newf("failed to map db retrieval: %w", err)
	}

	return retrieval, nil
}

func (s *Storage) InsertLLMCall(ctx context.Context, call *chattype.MessageLLMCall) error {
	dbCall, err := models.ChatMessageLLMCalls.Insert(llmCallToDB(call)).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	inserted, err := llmCallFromDB(dbCall)
	if err != nil {
		return xerrors.Newf("failed to map db llm call: %w", err)
	}

	*call = *inserted
	return nil
}

func (s *Storage) ListLLMCalls(ctx context.Context, filter chatstorage.LLMCallsFilter) ([]*chattype.MessageLLMCall, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(filter.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatMessageLLMCalls.ID.In(filter.ID...))
	}
	if len(filter.MessageID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatMessageLLMCalls.MessageID.In(filter.MessageID...))
	}
	if len(filter.ProviderConfigID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ChatMessageLLMCalls.ProviderConfigID.In(filter.ProviderConfigID...))
	}

	queryMods = append(queryMods, common.PaginationToBobMods(filter.Pagination)...)

	orderingMods, err := common.OrderingToBobMods(filter.Ordering, listLLMCallsOrderingExpr)
	if err != nil {
		return nil, xerrors.Newf("invalid llm calls filter: %w", err)
	}
	queryMods = append(queryMods, orderingMods...)

	dbCalls, err := models.ChatMessageLLMCalls.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch llm calls: %w", err)
	}

	return llmCallsFromDB(dbCalls)
}

func (s *Storage) UpsertProviderConfig(ctx context.Context, config *llm.ProviderConfig) error {
	dbConfig, err := models.LLMProviderConfigs.Insert(
		providerConfigToDB(config),
		im.OnConflict(info.LLMProviderConfigs.Columns.Provider.Name).DoUpdate(
			im.SetExcluded(
				info.LLMProviderConfigs.Columns.DisplayName.Name,
				info.LLMProviderConfigs.Columns.APIKeyEnc.Name,
				info.LLMProviderConfigs.Columns.BaseURL.Name,
				info.LLMProviderConfigs.Columns.IsEnabled.Name,
				info.LLMProviderConfigs.Columns.Metadata.Name,
			),
			im.SetCol(info.LLMProviderConfigs.Columns.UpdatedAt.Name).To(psql.Raw("CURRENT_TIMESTAMP")),
		),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	inserted, err := providerConfigFromDB(dbConfig)
	if err != nil {
		return xerrors.Newf("failed to map db provider config: %w", err)
	}

	*config = *inserted
	return nil
}

func (s *Storage) GetProviderConfig(ctx context.Context, id int64) (*llm.ProviderConfig, error) {
	dbConfig, err := models.FindLLMProviderConfig(ctx, s.Executor(ctx), id)
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch provider config: %w", err)
	}
	if dbConfig == nil {
		return nil, nil
	}

	config, err := providerConfigFromDB(dbConfig)
	if err != nil {
		return nil, xerrors.Newf("failed to map db provider config: %w", err)
	}

	return config, nil
}

func (s *Storage) ListProviderConfigs(ctx context.Context, filter chatstorage.ProviderConfigsFilter) ([]*llm.ProviderConfig, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(filter.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.LLMProviderConfigs.ID.In(filter.ID...))
	}
	if len(filter.Provider) > 0 {
		providers := make([]string, 0, len(filter.Provider))
		for _, provider := range filter.Provider {
			providers = append(providers, string(provider))
		}
		queryMods = append(queryMods, models.SelectWhere.LLMProviderConfigs.Provider.In(providers...))
	}
	if filter.IsEnabled != nil {
		queryMods = append(queryMods, models.SelectWhere.LLMProviderConfigs.IsEnabled.EQ(*filter.IsEnabled))
	}

	dbConfigs, err := models.LLMProviderConfigs.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch provider configs: %w", err)
	}

	return providerConfigsFromDB(dbConfigs)
}

func (s *Storage) UpsertProviderModel(ctx context.Context, model *llm.ProviderModel) error {
	dbModel, err := models.LLMProviderModels.Insert(
		providerModelToDB(model),
		im.OnConflict(
			info.LLMProviderModels.Columns.ProviderConfigID.Name,
			info.LLMProviderModels.Columns.Model.Name,
		).DoUpdate(
			im.SetExcluded(
				info.LLMProviderModels.Columns.DisplayName.Name,
				info.LLMProviderModels.Columns.ContextWindow.Name,
				info.LLMProviderModels.Columns.SupportsSQL.Name,
				info.LLMProviderModels.Columns.IsEnabled.Name,
				info.LLMProviderModels.Columns.DiscoveredAt.Name,
			),
		),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	inserted, err := providerModelFromDB(dbModel)
	if err != nil {
		return xerrors.Newf("failed to map db provider model: %w", err)
	}

	*model = *inserted
	return nil
}

func (s *Storage) ListProviderModels(ctx context.Context, filter chatstorage.ProviderModelsFilter) ([]*llm.ProviderModel, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(filter.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.LLMProviderModels.ID.In(filter.ID...))
	}
	if len(filter.ProviderConfigID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.LLMProviderModels.ProviderConfigID.In(filter.ProviderConfigID...))
	}
	if len(filter.Model) > 0 {
		queryMods = append(queryMods, models.SelectWhere.LLMProviderModels.Model.In(filter.Model...))
	}
	if filter.IsEnabled != nil {
		queryMods = append(queryMods, models.SelectWhere.LLMProviderModels.IsEnabled.EQ(*filter.IsEnabled))
	}
	if filter.SupportsSQL != nil {
		queryMods = append(queryMods, models.SelectWhere.LLMProviderModels.SupportsSQL.EQ(*filter.SupportsSQL))
	}

	dbModels, err := models.LLMProviderModels.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch provider models: %w", err)
	}

	return providerModelsFromDB(dbModels)
}

func listConversationsOrderingExpr(field chatstorage.ConversationOrdering) (bob.Expression, error) {
	switch field {
	case chatstorage.ConversationOrderingCreatedAt:
		return models.ChatConversations.Columns.CreatedAt, nil
	case chatstorage.ConversationOrderingUpdatedAt:
		return models.ChatConversations.Columns.UpdatedAt, nil
	case chatstorage.ConversationOrderingID:
		return models.ChatConversations.Columns.ID, nil
	default:
		return nil, xerrors.Newf("unsupported conversations ordering: %v", field)
	}
}

func listMessagesOrderingExpr(field chatstorage.MessageOrdering) (bob.Expression, error) {
	switch field {
	case chatstorage.MessageOrderingCreatedAt:
		return models.ChatMessages.Columns.CreatedAt, nil
	case chatstorage.MessageOrderingID:
		return models.ChatMessages.Columns.ID, nil
	default:
		return nil, xerrors.Newf("unsupported messages ordering: %v", field)
	}
}

func listLLMCallsOrderingExpr(field chatstorage.LLMCallOrdering) (bob.Expression, error) {
	switch field {
	case chatstorage.LLMCallOrderingCreatedAt:
		return models.ChatMessageLLMCalls.Columns.CreatedAt, nil
	case chatstorage.LLMCallOrderingID:
		return models.ChatMessageLLMCalls.Columns.ID, nil
	default:
		return nil, xerrors.Newf("unsupported llm calls ordering: %v", field)
	}
}
