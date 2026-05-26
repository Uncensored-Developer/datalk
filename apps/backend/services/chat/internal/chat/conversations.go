package chat

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) CreateConversation(ctx context.Context, userID int32, params chattype.CreateConversationParams) (*chattype.Conversation, error) {
	if _, err := s.getQueryableConnection(ctx, userID, params.ConnectionID); err != nil {
		return nil, err
	}

	conversation := &chattype.Conversation{
		UserID:       userID,
		ConnectionID: params.ConnectionID,
		Title:        params.Title,
	}
	if err := s.storage.InsertConversation(ctx, conversation); err != nil {
		return nil, xerrors.Newf("failed to create conversation: %w", err)
	}

	return conversation, nil
}

func (s *Service) GetConversation(ctx context.Context, userID int32, conversationID int64) (*chattype.Conversation, error) {
	conversation, err := s.storage.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, xerrors.Newf("failed to fetch conversation: %w", err)
	}
	if conversation == nil || conversation.UserID != userID {
		return nil, chaterrors.ErrConversationNotFound
	}

	return conversation, nil
}

func (s *Service) ListConversations(ctx context.Context, userID int32, filter chattype.ListConversationsFilter) ([]*chattype.Conversation, error) {
	conversations, err := s.storage.ListConversations(ctx, chatstorage.ConversationsFilter{
		UserID:       []int32{userID},
		ConnectionID: filter.ConnectionID,
		Pagination: pagination.LimitOffsetPagination{
			Limit:  filter.Limit,
			Offset: filter.Offset,
		},
		Ordering: ordering.Orderings[chatstorage.ConversationOrdering]{
			ordering.OrderByDesc(chatstorage.ConversationOrderingUpdatedAt),
			ordering.OrderByDesc(chatstorage.ConversationOrderingID),
		},
	})
	if err != nil {
		return nil, xerrors.Newf("failed to list conversations: %w", err)
	}

	return conversations, nil
}

func (s *Service) DeleteConversation(ctx context.Context, userID int32, conversationID int64) error {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return err
	}

	if err := s.storage.DeleteConversation(ctx, conversationID); err != nil {
		return xerrors.Newf("failed to delete conversation: %w", err)
	}

	return nil
}
