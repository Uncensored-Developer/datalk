package chat

import (
	"context"
	"testing"

	chatllm "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/testing"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_InferConversationTitle(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	provider := llmtypes.ProviderOpenAI
	model := "openai:gpt-5.2"
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockStorage.
		On("ListMessages", ctx, mock.MatchedBy(func(filter chatstorage.MessagesFilter) bool {
			assert.Equal(t, []int64{conversation.ID}, filter.ConversationID)
			assert.Equal(t, []chattype.MessageStatus{chattype.MessageStatusCompleted}, filter.Status)
			return true
		})).
		Return([]*chattype.Message{
			{
				ID:             100,
				ConversationID: conversation.ID,
				Role:           chattype.MessageRoleUser,
				Content:        "Which customers grew revenue the most last quarter?",
				Status:         chattype.MessageStatusCompleted,
			},
			{
				ID:              101,
				ConversationID:  conversation.ID,
				Role:            chattype.MessageRoleAssistant,
				Content:         "Revenue growth by customer.",
				Provider:        &provider,
				Model:           &model,
				Status:          chattype.MessageStatusCompleted,
				NaturalResponse: ptr.Of("Acme had the largest revenue growth."),
			},
		}, nil).
		Once()
	mockStorage.On("GetConversation", ctx, conversation.ID).Return(&chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}, nil).Once()
	mockStorage.
		On("UpsertConversation", ctx, mock.MatchedBy(func(updated *chattype.Conversation) bool {
			require.NotNil(t, updated.Title)
			assert.Equal(t, "Revenue Growth", *updated.Title)
			return updated.ID == conversation.ID
		})).
		Return(nil).
		Once()

	service := newTestService(mockStorage, nil, nil, titleResolver{title: "Revenue Growth"}, nil)

	got, err := service.InferConversationTitle(ctx, userID, conversation.ID)

	require.NoError(t, err)
	require.NotNil(t, got.Title)
	assert.Equal(t, "Revenue Growth", *got.Title)
}

func TestService_InferConversationTitle_DoesNotOverwriteExistingTitle(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	title := "Existing"
	conversation := &chattype.Conversation{ID: 10, UserID: 7, ConnectionID: 42, Title: &title}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()

	service := newTestService(mockStorage, nil, nil, titleResolver{title: "New Title"}, nil)

	got, err := service.InferConversationTitle(ctx, 7, conversation.ID)

	require.NoError(t, err)
	assert.Equal(t, conversation, got)
	mockStorage.AssertNotCalled(t, "UpsertConversation", mock.Anything, mock.Anything)
}

func TestService_InferConversationTitle_NotReady(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	conversation := &chattype.Conversation{ID: 10, UserID: 7, ConnectionID: 42}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockStorage.On("ListMessages", ctx, mock.Anything).Return([]*chattype.Message{
		{
			ID:             100,
			ConversationID: conversation.ID,
			Role:           chattype.MessageRoleUser,
			Content:        "hello",
			Status:         chattype.MessageStatusCompleted,
		},
	}, nil).Once()

	service := newTestService(mockStorage, nil, nil, titleResolver{title: "Hello"}, nil)

	_, err := service.InferConversationTitle(ctx, 7, conversation.ID)

	require.ErrorIs(t, err, chaterrors.ErrConversationTitleNotReady)
}

type titleResolver struct {
	title string
}

func (r titleResolver) ResolveClient(context.Context, llmtypes.Provider, string) (*chatllm.ResolvedClient, error) {
	return &chatllm.ResolvedClient{
		ResolvedModel: &chatllm.ResolvedModel{
			ProviderModelID: "gpt-5.2",
		},
		Client: titleClient{title: r.title},
	}, nil
}

type titleClient struct {
	title string
}

func (c titleClient) ListModels(context.Context) ([]llmtypes.Model, error) {
	return nil, nil
}

func (c titleClient) GenerateSQL(context.Context, llmtypes.GenerateSQLRequest) (*llmtypes.GenerateSQLResponse, error) {
	return nil, nil
}

func (c titleClient) GenerateAnswer(context.Context, llmtypes.GenerateAnswerRequest) (*llmtypes.GenerateAnswerResponse, error) {
	return nil, nil
}

func (c titleClient) GenerateConversationTitle(context.Context, llmtypes.GenerateConversationTitleRequest) (*llmtypes.GenerateConversationTitleResponse, error) {
	return &llmtypes.GenerateConversationTitleResponse{Title: c.title}, nil
}
