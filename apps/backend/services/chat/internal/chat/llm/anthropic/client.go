package anthropic

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

const (
	defaultBaseURL   = "https://api.anthropic.com"
	anthropicVersion = "2023-06-01"
	toolName         = "propose_sql_query"
)

var (
	ErrInvalidBaseURL = xerrors.New("anthropic base url cannot be empty")
	ErrMissingAPIKey  = xerrors.New("anthropic api key cannot be empty")
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type model struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	CreatedAt   *string `json:"created_at"`
}

type modelsResponse struct {
	Data []model `json:"data"`
}

type errorDetail struct {
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type generateRequest struct {
	Model      string     `json:"model"`
	MaxTokens  int        `json:"max_tokens"`
	System     string     `json:"system"`
	Messages   []message  `json:"messages"`
	Tools      []tool     `json:"tools"`
	ToolChoice toolChoice `json:"tool_choice"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type toolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type usage struct {
	InputTokens  *int `json:"input_tokens"`
	OutputTokens *int `json:"output_tokens"`
}

type generateResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      usage          `json:"usage"`
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
		return nil, xerrors.Newf("failed to create anthropic models request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, xerrors.Newf("failed to call anthropic models api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read anthropic models response: %w", err)
	}
	if err := decodeHTTPError("anthropic models api", resp.StatusCode, body); err != nil {
		return nil, err
	}

	var payload modelsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode anthropic models response: %w", err)
	}

	models := make([]llmtypes.Model, 0, len(payload.Data))
	for _, item := range payload.Data {
		modelID := strings.TrimSpace(item.ID)
		if modelID == "" {
			continue
		}
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		models = append(models, llmtypes.Model{
			ID:           modelID,
			DisplayName:  displayName,
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
		Model:     req.Model,
		MaxTokens: 1500,
		System:    llmcore.GenerateSQLSystemPrompt(req),
		Messages:  anthropicMessages(req),
		Tools: []tool{
			{
				Name:        toolName,
				Description: "Return the generated SQL query and supporting metadata.",
				InputSchema: llmcore.GenerateSQLSchema(),
			},
		},
		ToolChoice: toolChoice{
			Type: "tool",
			Name: toolName,
		},
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal anthropic generate request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create anthropic generate request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call anthropic messages api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read anthropic generate response: %w", err)
	}
	if err := decodeHTTPError("anthropic messages api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload generateResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode anthropic generate response: %w", err)
	}

	var toolInput json.RawMessage
	for _, block := range payload.Content {
		if block.Type == "tool_use" && block.Name == toolName {
			toolInput = block.Input
			break
		}
	}
	if len(toolInput) == 0 {
		return nil, xerrors.New("anthropic generate response did not include tool output")
	}

	var finishReason *string
	if strings.TrimSpace(payload.StopReason) != "" {
		finishReason = &payload.StopReason
	}

	usage := &llmtypes.Usage{
		InputTokens:  payload.Usage.InputTokens,
		OutputTokens: payload.Usage.OutputTokens,
	}
	if payload.Usage.InputTokens != nil && payload.Usage.OutputTokens != nil {
		total := *payload.Usage.InputTokens + *payload.Usage.OutputTokens
		usage.TotalTokens = &total
	}

	return llmcore.ParseGenerateSQLResponse(rawRequest, rawResponse, string(toolInput), usage, finishReason)
}

func decodeHTTPError(apiName string, statusCode int, body []byte) error {
	if statusCode < http.StatusBadRequest {
		return nil
	}

	var errPayload errorResponse
	if json.Unmarshal(body, &errPayload) == nil && strings.TrimSpace(errPayload.Error.Message) != "" {
		return xerrors.Newf("%s failed with status %d: %s", apiName, statusCode, errPayload.Error.Message)
	}

	return xerrors.Newf("%s failed with status %d: %s", apiName, statusCode, strings.TrimSpace(string(body)))
}

func anthropicMessages(req llmtypes.GenerateSQLRequest) []message {
	promptMessages := llmcore.GenerateSQLMessages(req)
	messages := make([]message, 0, len(promptMessages))
	for _, promptMessage := range promptMessages {
		messages = append(messages, message{
			Role:    normalizeAnthropicRole(promptMessage.Role),
			Content: promptMessage.Content,
		})
	}
	return messages
}

func normalizeAnthropicRole(role string) string {
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
		SupportsVision:             true,
	}
}
