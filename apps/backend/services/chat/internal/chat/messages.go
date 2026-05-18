package chat

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) ListMessages(ctx context.Context, userID int32, filter chattype.ListMessagesFilter) ([]*chattype.MessageDetails, error) {
	conversation, err := s.GetConversation(ctx, userID, filter.ConversationID)
	if err != nil {
		return nil, err
	}

	messages, err := s.storage.ListMessages(ctx, chatstorage.MessagesFilter{
		ConversationID: []int64{conversation.ID},
		Pagination: pagination.LimitOffsetPagination{
			Limit:  filter.Limit,
			Offset: filter.Offset,
		},
		Ordering: ordering.Orderings[chatstorage.MessageOrdering]{
			ordering.OrderByAsc(chatstorage.MessageOrderingCreatedAt),
			ordering.OrderByAsc(chatstorage.MessageOrderingID),
		},
	})
	if err != nil {
		return nil, xerrors.Newf("failed to list messages: %w", err)
	}

	details := make([]*chattype.MessageDetails, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}

		detail := &chattype.MessageDetails{Message: message}
		switch message.Role {
		case chattype.MessageRoleUser:
			retrieval, err := s.storage.GetRetrieval(ctx, message.ID)
			if err != nil {
				return nil, xerrors.Newf("failed to load message retrieval: %w", err)
			}
			detail.Retrieval = retrieval
		case chattype.MessageRoleAssistant:
			execution, err := s.storage.GetExecution(ctx, message.ID)
			if err != nil {
				return nil, xerrors.Newf("failed to load message execution: %w", err)
			}
			detail.Execution = execution
		}

		details = append(details, detail)
	}

	return details, nil
}
