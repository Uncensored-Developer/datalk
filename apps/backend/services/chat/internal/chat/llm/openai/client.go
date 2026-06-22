package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	llmcore "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
)

const defaultBaseURL = "https://api.openai.com"

var (
	ErrInvalidBaseURL = xerrors.New("openai base url cannot be empty")
	ErrMissingAPIKey  = xerrors.New("openai api key cannot be empty")
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type model struct {
	ID string `json:"id"`
}

type modelsResponse struct {
	Data []model `json:"data"`
}

type errorDetail struct {
	Message string `json:"message"`
}

type responseError struct {
	Error errorDetail `json:"error"`
}

type generateRequest struct {
	Model string         `json:"model"`
	Input []inputMessage `json:"input"`
	Text  textConfig     `json:"text"`
}

type inputMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type textConfig struct {
	Format map[string]any `json:"format"`
}

type outputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type outputItem struct {
	Type    string          `json:"type"`
	Content []outputContent `json:"content"`
}

type usage struct {
	InputTokens  *int `json:"input_tokens"`
	OutputTokens *int `json:"output_tokens"`
	TotalTokens  *int `json:"total_tokens"`
}

type incompleteDetails struct {
	Reason string `json:"reason"`
}

type generateResponse struct {
	Output            []outputItem       `json:"output"`
	OutputText        string             `json:"output_text"`
	Status            string             `json:"status"`
	Usage             usage              `json:"usage"`
	IncompleteDetails *incompleteDetails `json:"incomplete_details"`
}

func NewClient(config *llmtypes.ProviderConfig, httpClient *http.Client) (*Client, error) {
	if config == nil {
		return nil, xerrors.New("provider config cannot be nil")
	}
	if strings.TrimSpace(config.APIKeyEnc) == "" {
		return nil, ErrMissingAPIKey
	}

	baseURL := defaultBaseURL
	if config.BaseURL != nil {
		baseURL = strings.TrimSpace(*config.BaseURL)
	}
	if baseURL == "" {
		return nil, ErrInvalidBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     config.APIKeyEnc,
		httpClient: httpClient,
	}, nil
}

func (c *Client) ListModels(ctx context.Context) ([]llmtypes.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, xerrors.Newf("failed to create openai models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, xerrors.Newf("failed to call openai models api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read openai models response: %w", err)
	}
	if err := decodeHTTPError("openai models api", resp.StatusCode, body); err != nil {
		return nil, err
	}

	var payload modelsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode openai models response: %w", err)
	}

	models := make([]llmtypes.Model, 0, len(payload.Data))
	for _, item := range payload.Data {
		modelID := strings.TrimSpace(item.ID)
		if modelID == "" {
			continue
		}
		models = append(models, llmtypes.Model{
			ID:           modelID,
			DisplayName:  modelID,
			IsEnabled:    true,
			Capabilities: modelCapabilities(),
		})
	}

	return models, nil
}

func (c *Client) GenerateSQL(ctx context.Context, req llmtypes.GenerateSQLRequest) (*llmtypes.GenerateSQLResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, xerrors.New("model is required")
	}

	requestBody := generateRequest{
		Model: req.Model,
		Input: openAIInputMessages(req),
		Text: textConfig{
			Format: map[string]any{
				"type":   "json_schema",
				"name":   "sql_generation",
				"strict": true,
				"schema": llmcore.GenerateSQLSchema(),
			},
		},
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal openai generate request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/responses", bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create openai generate request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call openai responses api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read openai generate response: %w", err)
	}
	if err := decodeHTTPError("openai responses api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload generateResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode openai generate response: %w", err)
	}

	responseText := strings.TrimSpace(extractOutputText(payload.Output, payload.OutputText))
	if responseText == "" {
		return nil, xerrors.New("openai generate response did not include structured output text")
	}

	var finishReason *string
	if payload.IncompleteDetails != nil && strings.TrimSpace(payload.IncompleteDetails.Reason) != "" {
		finishReason = &payload.IncompleteDetails.Reason
	} else if strings.TrimSpace(payload.Status) != "" && payload.Status != "completed" {
		finishReason = &payload.Status
	}

	return llmcore.ParseGenerateSQLResponse(rawRequest, rawResponse, responseText, &llmtypes.Usage{
		InputTokens:  payload.Usage.InputTokens,
		OutputTokens: payload.Usage.OutputTokens,
		TotalTokens:  payload.Usage.TotalTokens,
	}, finishReason)
}

func (c *Client) GenerateAnswer(ctx context.Context, req llmtypes.GenerateAnswerRequest) (*llmtypes.GenerateAnswerResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, xerrors.New("model is required")
	}

	requestBody := generateRequest{
		Model: req.Model,
		Input: openAIAnswerInputMessages(req),
		Text: textConfig{
			Format: map[string]any{
				"type":   "json_schema",
				"name":   "answer_generation",
				"strict": true,
				"schema": llmcore.GenerateAnswerSchema(),
			},
		},
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal openai answer request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/responses", bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create openai answer request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call openai responses api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read openai answer response: %w", err)
	}
	if err := decodeHTTPError("openai responses api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload generateResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode openai answer response: %w", err)
	}

	responseText := strings.TrimSpace(extractOutputText(payload.Output, payload.OutputText))
	if responseText == "" {
		return nil, xerrors.New("openai answer response did not include structured output text")
	}

	var finishReason *string
	if payload.IncompleteDetails != nil && strings.TrimSpace(payload.IncompleteDetails.Reason) != "" {
		finishReason = &payload.IncompleteDetails.Reason
	} else if strings.TrimSpace(payload.Status) != "" && payload.Status != "completed" {
		finishReason = &payload.Status
	}

	return llmcore.ParseGenerateAnswerResponse(rawRequest, rawResponse, responseText, &llmtypes.Usage{
		InputTokens:  payload.Usage.InputTokens,
		OutputTokens: payload.Usage.OutputTokens,
		TotalTokens:  payload.Usage.TotalTokens,
	}, finishReason)
}

func (c *Client) GenerateConversationTitle(ctx context.Context, req llmtypes.GenerateConversationTitleRequest) (*llmtypes.GenerateConversationTitleResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, xerrors.New("model is required")
	}

	requestBody := generateRequest{
		Model: req.Model,
		Input: openAIConversationTitleInputMessages(req),
		Text: textConfig{
			Format: map[string]any{
				"type":   "json_schema",
				"name":   "conversation_title",
				"strict": true,
				"schema": llmcore.GenerateConversationTitleSchema(),
			},
		},
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal openai title request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/responses", bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create openai title request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call openai responses api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read openai title response: %w", err)
	}
	if err := decodeHTTPError("openai responses api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload generateResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode openai title response: %w", err)
	}

	responseText := strings.TrimSpace(extractOutputText(payload.Output, payload.OutputText))
	if responseText == "" {
		return nil, xerrors.New("openai title response did not include structured output text")
	}

	var finishReason *string
	if payload.IncompleteDetails != nil && strings.TrimSpace(payload.IncompleteDetails.Reason) != "" {
		finishReason = &payload.IncompleteDetails.Reason
	} else if strings.TrimSpace(payload.Status) != "" && payload.Status != "completed" {
		finishReason = &payload.Status
	}

	return llmcore.ParseGenerateConversationTitleResponse(rawRequest, rawResponse, responseText, &llmtypes.Usage{
		InputTokens:  payload.Usage.InputTokens,
		OutputTokens: payload.Usage.OutputTokens,
		TotalTokens:  payload.Usage.TotalTokens,
	}, finishReason)
}

func decodeHTTPError(apiName string, statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var errPayload responseError
	if json.Unmarshal(body, &errPayload) == nil && strings.TrimSpace(errPayload.Error.Message) != "" {
		return xerrors.Newf("%s failed with status %d: %s", apiName, statusCode, errPayload.Error.Message)
	}

	return xerrors.Newf("%s failed with status %d: %s", apiName, statusCode, strings.TrimSpace(string(body)))
}

func extractOutputText(output []outputItem, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}

	for _, item := range output {
		for _, content := range item.Content {
			if strings.TrimSpace(content.Text) != "" {
				return content.Text
			}
		}
	}

	return ""
}

func openAIInputMessages(req llmtypes.GenerateSQLRequest) []inputMessage {
	promptMessages := llmcore.GenerateSQLMessages(req)
	messages := make([]inputMessage, 0, len(promptMessages)+1)
	messages = append(messages, inputMessage{
		Role: "developer",
		Content: []contentPart{
			{
				Type: "input_text",
				Text: llmcore.GenerateSQLSystemPrompt(req),
			},
		},
	})

	for _, message := range promptMessages {
		messages = append(messages, inputMessage{
			Role: normalizeOpenAIRole(message.Role),
			Content: []contentPart{
				{
					Type: "input_text",
					Text: message.Content,
				},
			},
		})
	}

	return messages
}

func openAIAnswerInputMessages(req llmtypes.GenerateAnswerRequest) []inputMessage {
	promptMessages := llmcore.GenerateAnswerMessages(req)
	messages := make([]inputMessage, 0, len(promptMessages)+1)
	messages = append(messages, inputMessage{
		Role: "developer",
		Content: []contentPart{
			{
				Type: "input_text",
				Text: llmcore.GenerateAnswerSystemPrompt(req),
			},
		},
	})

	for _, message := range promptMessages {
		messages = append(messages, inputMessage{
			Role: normalizeOpenAIRole(message.Role),
			Content: []contentPart{
				{
					Type: "input_text",
					Text: message.Content,
				},
			},
		})
	}

	return messages
}

func openAIConversationTitleInputMessages(req llmtypes.GenerateConversationTitleRequest) []inputMessage {
	promptMessages := llmcore.GenerateConversationTitleMessages(req)
	messages := make([]inputMessage, 0, len(promptMessages)+1)
	messages = append(messages, inputMessage{
		Role: "developer",
		Content: []contentPart{
			{
				Type: "input_text",
				Text: llmcore.GenerateConversationTitleSystemPrompt(req),
			},
		},
	})

	for _, message := range promptMessages {
		messages = append(messages, inputMessage{
			Role: normalizeOpenAIRole(message.Role),
			Content: []contentPart{
				{
					Type: "input_text",
					Text: message.Content,
				},
			},
		})
	}

	return messages
}

func normalizeOpenAIRole(role string) string {
	switch role {
	case "assistant":
		return "assistant"
	default:
		return "user"
	}
}

func modelCapabilities() llmtypes.ModelCapabilities {
	return llmtypes.ModelCapabilities{
		SupportsToolCalling:        true,
		SupportsStructuredOutput:   true,
		SupportsStreaming:          true,
		SupportsSystemInstructions: true,
	}
}
