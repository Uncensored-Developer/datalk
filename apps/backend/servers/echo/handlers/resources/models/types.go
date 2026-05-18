package models

import llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"

type modelResponse struct {
	ID           string                    `json:"id"`
	Provider     llmtypes.Provider         `json:"provider"`
	DisplayName  string                    `json:"display_name"`
	Description  *string                   `json:"description,omitempty"`
	IsEnabled    bool                      `json:"is_enabled"`
	Capabilities modelCapabilitiesResponse `json:"capabilities"`
}

type modelCapabilitiesResponse struct {
	SupportsToolCalling        bool `json:"supports_tool_calling"`
	SupportsStructuredOutput   bool `json:"supports_structured_output"`
	SupportsStreaming          bool `json:"supports_streaming"`
	SupportsSystemInstructions bool `json:"supports_system_instructions"`
	SupportsVision             bool `json:"supports_vision"`
	MaxContextTokens           *int `json:"max_context_tokens,omitempty"`
	MaxOutputTokens            *int `json:"max_output_tokens,omitempty"`
}
