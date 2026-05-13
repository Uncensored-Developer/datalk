package ollama

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

const defaultBaseURL = "http://127.0.0.1:11434"

var ErrInvalidBaseURL = xerrors.New("ollama base url cannot be empty")

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type modelDetails struct {
	ParameterSize string `json:"parameter_size"`
}

type model struct {
	Name    string       `json:"name"`
	Model   string       `json:"model"`
	Details modelDetails `json:"details"`
}

type tagsResponse struct {
	Models []model `json:"models"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []message      `json:"messages"`
	Format   map[string]any `json:"format"`
	Stream   bool           `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message         message `json:"message"`
	DoneReason      string  `json:"done_reason"`
	PromptEvalCount *int    `json:"prompt_eval_count"`
	EvalCount       *int    `json:"eval_count"`
}

func NewClient(config *llmtypes.ProviderConfig, httpClient *http.Client) (*Client, error) {
	if config == nil {
		return nil, xerrors.New("provider config cannot be nil")
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
		httpClient: httpClient,
	}, nil
}

func (c *Client) ListModels(ctx context.Context) ([]llmtypes.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, xerrors.Newf("failed to create ollama tags request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, xerrors.Newf("failed to call ollama tags api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read ollama tags response: %w", err)
	}
	if err := decodeHTTPError("ollama tags api", resp.StatusCode, body); err != nil {
		return nil, err
	}

	var payload tagsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode ollama tags response: %w", err)
	}

	models := make([]llmtypes.Model, 0, len(payload.Models))
	for _, item := range payload.Models {
		modelID := strings.TrimSpace(item.Model)
		if modelID == "" {
			modelID = strings.TrimSpace(item.Name)
		}
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

	requestBody := chatRequest{
		Model:    req.Model,
		Messages: ollamaMessages(req),
		Format:   llmcore.GenerateSQLSchema(),
		Stream:   false,
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal ollama generate request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create ollama chat request: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call ollama chat api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read ollama chat response: %w", err)
	}
	if err := decodeHTTPError("ollama chat api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload chatResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode ollama chat response: %w", err)
	}
	if strings.TrimSpace(payload.Message.Content) == "" {
		return nil, xerrors.New("ollama chat response did not include structured output")
	}

	var finishReason *string
	if strings.TrimSpace(payload.DoneReason) != "" {
		finishReason = &payload.DoneReason
	}

	usage := &llmtypes.Usage{
		InputTokens:  payload.PromptEvalCount,
		OutputTokens: payload.EvalCount,
	}
	if payload.PromptEvalCount != nil && payload.EvalCount != nil {
		total := *payload.PromptEvalCount + *payload.EvalCount
		usage.TotalTokens = &total
	}

	return llmcore.ParseGenerateSQLResponse(rawRequest, rawResponse, payload.Message.Content, usage, finishReason)
}

func decodeHTTPError(apiName string, statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var errPayload errorResponse
	if json.Unmarshal(body, &errPayload) == nil && strings.TrimSpace(errPayload.Error) != "" {
		return xerrors.Newf("%s failed with status %d: %s", apiName, statusCode, errPayload.Error)
	}

	return xerrors.Newf("%s failed with status %d: %s", apiName, statusCode, strings.TrimSpace(string(body)))
}

func ollamaMessages(req llmtypes.GenerateSQLRequest) []message {
	promptMessages := llmcore.GenerateSQLMessages(req)
	messages := make([]message, 0, len(promptMessages)+1)
	messages = append(messages, message{
		Role:    "system",
		Content: llmcore.GenerateSQLSystemPrompt(req),
	})
	for _, promptMessage := range promptMessages {
		messages = append(messages, message{
			Role:    normalizeOllamaRole(promptMessage.Role),
			Content: promptMessage.Content,
		})
	}
	return messages
}

func normalizeOllamaRole(role string) string {
	switch role {
	case "assistant":
		return "assistant"
	default:
		return "user"
	}
}

func modelCapabilities() llmtypes.ModelCapabilities {
	return llmtypes.ModelCapabilities{
		SupportsStructuredOutput:   true,
		SupportsStreaming:          true,
		SupportsSystemInstructions: true,
	}
}
