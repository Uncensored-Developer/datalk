package models

import llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"

func toModelResponses(models []llmtypes.Model) []modelResponse {
	out := make([]modelResponse, 0, len(models))
	for _, model := range models {
		out = append(out, modelResponse{
			ID:          model.ID,
			Provider:    model.Provider,
			DisplayName: model.DisplayName,
			Description: model.Description,
			IsEnabled:   model.IsEnabled,
			Capabilities: modelCapabilitiesResponse{
				SupportsToolCalling:        model.Capabilities.SupportsToolCalling,
				SupportsStructuredOutput:   model.Capabilities.SupportsStructuredOutput,
				SupportsStreaming:          model.Capabilities.SupportsStreaming,
				SupportsSystemInstructions: model.Capabilities.SupportsSystemInstructions,
				SupportsVision:             model.Capabilities.SupportsVision,
				MaxContextTokens:           model.Capabilities.MaxContextTokens,
				MaxOutputTokens:            model.Capabilities.MaxOutputTokens,
			},
		})
	}

	return out
}
