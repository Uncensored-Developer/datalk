package chat

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	chattesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/testing"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage/testing"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateConversation(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	connectionID := int32(42)
	title := "Revenue questions"
	mockStorage := storagetesting.NewStorage(t)
	mockConnections := chattesting.NewConnectionService(t)
	mockConnections.On("GetConnection", ctx, connectionID).Return(&connectiontypes.Connection{ID: connectionID, Database: connectiontypes.DatabasePostgres}, nil).Once()
	mockConnections.On("GetAccess", ctx, userID, connectionID).Return(&connectiontypes.Access{CanQuery: true}, nil).Once()
	mockStorage.
		On("UpsertConversation", ctx, mock.MatchedBy(func(conversation *chattype.Conversation) bool {
			return conversation.UserID == userID &&
				conversation.ConnectionID == connectionID &&
				conversation.Title != nil &&
				*conversation.Title == title
		})).
		Run(func(args mock.Arguments) {
			args.Get(1).(*chattype.Conversation).ID = 10
		}).
		Return(nil).
		Once()

	service := newTestService(mockStorage, mockConnections, nil, nil, nil)

	conversation, err := service.CreateConversation(ctx, userID, chattype.CreateConversationParams{
		ConnectionID: connectionID,
		Title:        ptr.Of(title),
	})

	require.NoError(t, err)
	require.NotNil(t, conversation)
	assert.Equal(t, int64(10), conversation.ID)
	assert.Equal(t, userID, conversation.UserID)
	assert.Equal(t, connectionID, conversation.ConnectionID)
}

func TestService_GetConversation(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	conversation := &chattype.Conversation{ID: 10, UserID: userID, ConnectionID: 42}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.On("GetConversation", ctx, conversation.ID).Return(conversation, nil).Once()

	service := newTestService(mockStorage, nil, nil, nil, nil)

	got, err := service.GetConversation(ctx, userID, conversation.ID)

	require.NoError(t, err)
	assert.Equal(t, conversation, got)
}

func TestService_GetConversation_RejectsOtherUser(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.On("GetConversation", ctx, int64(10)).Return(&chattype.Conversation{ID: 10, UserID: 99, ConnectionID: 42}, nil).Once()

	service := newTestService(mockStorage, nil, nil, nil, nil)

	_, err := service.GetConversation(ctx, 7, 10)

	require.ErrorIs(t, err, chaterrors.ErrConversationNotFound)
}

func TestService_ListConversations(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	userID := int32(7)
	connectionID := int32(42)
	conversations := []*chattype.Conversation{
		{ID: 12, UserID: userID, ConnectionID: connectionID},
		{ID: 10, UserID: userID, ConnectionID: connectionID},
	}
	mockStorage := storagetesting.NewStorage(t)
	mockStorage.
		On("ListConversations", ctx, mock.MatchedBy(func(filter chatstorage.ConversationsFilter) bool {
			require.Equal(t, []int32{userID}, filter.UserID)
			require.Equal(t, []int32{connectionID}, filter.ConnectionID)
			require.Equal(t, 25, filter.Pagination.Limit)
			require.Equal(t, 50, filter.Pagination.Offset)
			require.Len(t, filter.Ordering, 2)
			assert.Equal(t, chatstorage.ConversationOrderingUpdatedAt, filter.Ordering[0].Field)
			assert.Equal(t, ordering.DirectionDesc, filter.Ordering[0].Direction)
			assert.Equal(t, chatstorage.ConversationOrderingID, filter.Ordering[1].Field)
			assert.Equal(t, ordering.DirectionDesc, filter.Ordering[1].Direction)
			return true
		})).
		Return(conversations, nil).
		Once()

	service := newTestService(mockStorage, nil, nil, nil, nil)

	got, err := service.ListConversations(ctx, userID, chattype.ListConversationsFilter{
		ConnectionID: []int32{connectionID},
		Limit:        25,
		Offset:       50,
	})

	require.NoError(t, err)
	assert.Equal(t, conversations, got)
}
