package api

import (
	"github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat"
	chattype "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/chat"
)

type (
	CreateConversationParams = chattype.CreateConversationParams
	ListConversationsFilter  = chattype.ListConversationsFilter
	ListMessagesFilter       = chattype.ListMessagesFilter
	SendMessageParams        = chattype.SendMessageParams
	SaveProviderConfigParams = chat.SaveProviderConfigParams
)
