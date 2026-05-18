package chat

import (
	"context"
	"strings"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) loadRecentHistory(ctx context.Context, conversationID int64, currentMessageID int64) ([]*chattype.Message, error) {
	messages, err := s.storage.ListMessages(ctx, chatstorage.MessagesFilter{
		ConversationID: []int64{conversationID},
		Pagination: pagination.LimitOffsetPagination{
			Limit: defaultHistoryLimit + 1,
		},
		// Fetch newest first so the DB limit selects the recent window, then
		// reverse the small bounded result below to send history oldest-to-newest.
		Ordering: ordering.Orderings[chatstorage.MessageOrdering]{
			ordering.OrderByDesc(chatstorage.MessageOrderingCreatedAt),
			ordering.OrderByDesc(chatstorage.MessageOrderingID),
		},
	})
	if err != nil {
		return nil, xerrors.Newf("failed to load message history: %w", err)
	}

	history := make([]*chattype.Message, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message == nil || message.ID == currentMessageID {
			continue
		}
		history = append(history, message)
	}

	return history, nil
}

func (s *Service) latestAssistantSQL(ctx context.Context, history []*chattype.Message) (*string, error) {
	for i := len(history) - 1; i >= 0; i-- {
		message := history[i]
		if message.Role != chattype.MessageRoleAssistant || message.Status != chattype.MessageStatusCompleted {
			continue
		}

		execution, err := s.storage.GetExecution(ctx, message.ID)
		if err != nil {
			return nil, xerrors.Newf("failed to load previous execution: %w", err)
		}
		if execution != nil && strings.TrimSpace(execution.GeneratedSQL) != "" {
			return &execution.GeneratedSQL, nil
		}
	}

	return nil, nil
}

func toConversationMessages(messages []*chattype.Message) []llmtypes.ConversationMessage {
	out := make([]llmtypes.ConversationMessage, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		out = append(out, llmtypes.ConversationMessage{
			Role:    string(message.Role),
			Content: message.Content,
		})
	}

	return out
}
