package chat

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/testing"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_ListMessages_ReturnsJoinedDetailsInCreationOrder(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	userMessage := &chattype.Message{ID: 100, ConversationID: conversation.ID, Role: chattype.MessageRoleUser, Content: "how many users?", Status: chattype.MessageStatusCompleted}
	assistantMessage := &chattype.Message{ID: 101, ConversationID: conversation.ID, Role: chattype.MessageRoleAssistant, Content: "3 users", Status: chattype.MessageStatusCompleted}
	retrieval := &chattype.MessageRetrieval{MessageID: userMessage.ID, SnapshotID: 55, QueryText: "how many users?"}
	execution := &chattype.MessageExecution{
		MessageID:    assistantMessage.ID,
		ConnectionID: conversation.ConnectionID,
		DatabaseKind: connectiontypes.DatabasePostgres,
		GeneratedSQL: "select count(*) from users",
		Result:       chattype.QueryResult{RowCount: 1, Kind: chattype.QueryResultKindScalar},
	}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()
	mockStorage.
		On("ListMessages", ctx, mock.MatchedBy(func(filter chatstorage.MessagesFilter) bool {
			require.Equal(t, []int64{conversation.ID}, filter.ConversationID)
			require.Equal(t, 20, filter.Pagination.Limit)
			require.Equal(t, 40, filter.Pagination.Offset)
			require.Len(t, filter.Ordering, 2)
			assert.Equal(t, chatstorage.MessageOrderingCreatedAt, filter.Ordering[0].Field)
			assert.Equal(t, ordering.DirectionAsc, filter.Ordering[0].Direction)
			assert.Equal(t, chatstorage.MessageOrderingID, filter.Ordering[1].Field)
			assert.Equal(t, ordering.DirectionAsc, filter.Ordering[1].Direction)
			return true
		})).
		Return([]*chattype.Message{userMessage, assistantMessage}, nil).
		Once()
	mockStorage.On("GetRetrieval", ctx, userMessage.ID).Return(retrieval, nil).Once()
	mockStorage.On("GetExecution", ctx, assistantMessage.ID).Return(execution, nil).Once()

	service := newTestService(mockStorage, nil, nil, nil, nil)

	got, err := service.ListMessages(ctx, userID, chattype.ListMessagesFilter{
		ConversationID: conversation.ID,
		Limit:          20,
		Offset:         40,
	})

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, userMessage, got[0].Message)
	assert.Equal(t, retrieval, got[0].Retrieval)
	assert.Nil(t, got[0].Execution)
	assert.Equal(t, assistantMessage, got[1].Message)
	assert.Equal(t, execution, got[1].Execution)
	assert.Nil(t, got[1].Retrieval)
}

func TestService_ListMessages_RejectsOtherUserConversation(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.On("GetConversation", ctx, int64(10)).Return(&chattype.Conversation{ID: 10, UserID: 99, ConnectionID: 42}, nil).Once()

	service := newTestService(mockStorage, nil, nil, nil, nil)

	_, err := service.ListMessages(ctx, 7, chattype.ListMessagesFilter{ConversationID: 10})

	require.ErrorIs(t, err, chaterrors.ErrConversationNotFound)
	mockStorage.AssertNotCalled(t, "ListMessages", mock.Anything, mock.Anything)
}
