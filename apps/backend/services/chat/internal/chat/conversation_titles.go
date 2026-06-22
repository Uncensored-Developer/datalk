package chat

import (
	"context"
	"strings"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	chatstorage "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/storage"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
	chaterrors "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/errors"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/gotidy/ptr"
	"github.com/mdobak/go-xerrors"
)

const (
	defaultConversationTitleMaxWords = 6
	defaultConversationTitleMaxChars = 80
)

func (s *Service) InferConversationTitle(ctx context.Context, userID int32, conversationID int64) (*chattype.Conversation, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if hasConversationTitle(conversation) {
		return conversation, nil
	}

	userMessage, assistantMessage, err := s.firstCompletedTurn(ctx, conversation.ID)
	if err != nil {
		return nil, err
	}
	if assistantMessage.Provider == nil || assistantMessage.Model == nil || strings.TrimSpace(*assistantMessage.Model) == "" {
		return nil, chaterrors.ErrConversationTitleNotReady
	}

	resolved, err := s.clientResolver.ResolveClient(ctx, *assistantMessage.Provider, *assistantMessage.Model)
	if err != nil {
		return nil, xerrors.Newf("failed to resolve title generation model: %w", err)
	}
	if resolved == nil || resolved.Client == nil {
		return nil, xerrors.Newf("model resolver returned incomplete client: %w", chaterrors.ErrModelNotAvailable)
	}

	resp, err := resolved.Client.GenerateConversationTitle(ctx, llmtypes.GenerateConversationTitleRequest{
		Model:      resolved.ProviderModelID,
		UserPrompt: userMessage.Content,
		Assistant:  messageNaturalResponseOrContent(assistantMessage),
		MaxWords:   defaultConversationTitleMaxWords,
		MaxChars:   defaultConversationTitleMaxChars,
	})
	if err != nil {
		return nil, xerrors.Newf("failed to generate conversation title: %w", err)
	}
	if resp == nil || strings.TrimSpace(resp.Title) == "" {
		return nil, xerrors.New("provider returned empty conversation title")
	}

	latest, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if hasConversationTitle(latest) {
		return latest, nil
	}

	latest.Title = ptr.Of(resp.Title)
	if err := s.storage.UpsertConversation(ctx, latest); err != nil {
		return nil, xerrors.Newf("failed to update conversation title: %w", err)
	}

	return latest, nil
}

func (s *Service) firstCompletedTurn(ctx context.Context, conversationID int64) (*chattype.Message, *chattype.Message, error) {
	messages, err := s.storage.ListMessages(ctx, chatstorage.MessagesFilter{
		ConversationID: []int64{conversationID},
		Status:         []chattype.MessageStatus{chattype.MessageStatusCompleted},
		Ordering: ordering.Orderings[chatstorage.MessageOrdering]{
			ordering.OrderByAsc(chatstorage.MessageOrderingCreatedAt),
			ordering.OrderByAsc(chatstorage.MessageOrderingID),
		},
	})
	if err != nil {
		return nil, nil, xerrors.Newf("failed to list title source messages: %w", err)
	}

	var firstUser *chattype.Message
	for _, message := range messages {
		if message == nil {
			continue
		}
		if firstUser == nil {
			if message.Role == chattype.MessageRoleUser && strings.TrimSpace(message.Content) != "" {
				firstUser = message
			}
			continue
		}
		if message.Role == chattype.MessageRoleAssistant {
			return firstUser, message, nil
		}
	}

	return nil, nil, chaterrors.ErrConversationTitleNotReady
}

func hasConversationTitle(conversation *chattype.Conversation) bool {
	return conversation != nil && conversation.Title != nil && strings.TrimSpace(*conversation.Title) != ""
}

func messageNaturalResponseOrContent(message *chattype.Message) string {
	if message == nil {
		return ""
	}
	if message.NaturalResponse != nil && strings.TrimSpace(*message.NaturalResponse) != "" {
		return *message.NaturalResponse
	}
	return message.Content
}
