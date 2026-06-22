package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	llmcore "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/internal/chat/llm"
	llmtypes "github.com/Uncensored-Developer/datalk/apps/backend/services/chat/pkg/llm"
	"github.com/mdobak/go-xerrors"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com"

var (
	ErrInvalidBaseURL = xerrors.New("gemini base url cannot be empty")
	ErrMissingAPIKey  = xerrors.New("gemini api key cannot be empty")
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type model struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName"`
	InputTokenLimit            *int     `json:"inputTokenLimit"`
	OutputTokenLimit           *int     `json:"outputTokenLimit"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}

type modelsResponse struct {
	Models []model `json:"models"`
}

type errorDetail struct {
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type generateRequest struct {
	SystemInstruction content          `json:"systemInstruction"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMIMEType string         `json:"responseMimeType"`
	ResponseSchema   map[string]any `json:"responseSchema"`
}

type candidatePart struct {
	Text string `json:"text"`
}

type candidateContent struct {
	Parts []candidatePart `json:"parts"`
}

type candidate struct {
	Content      candidateContent `json:"content"`
	FinishReason string           `json:"finishReason"`
}

type usageMetadata struct {
	PromptTokenCount     *int `json:"promptTokenCount"`
	CandidatesTokenCount *int `json:"candidatesTokenCount"`
	TotalTokenCount      *int `json:"totalTokenCount"`
}

type generateResponse struct {
	Candidates    []candidate   `json:"candidates"`
	UsageMetadata usageMetadata `json:"usageMetadata"`
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1beta/models?key="+url.QueryEscape(c.apiKey), nil)
	if err != nil {
		return nil, xerrors.Newf("failed to create gemini models request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, xerrors.Newf("failed to call gemini models api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read gemini models response: %w", err)
	}
	if err := decodeHTTPError("gemini models api", resp.StatusCode, body); err != nil {
		return nil, err
	}

	var payload modelsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode gemini models response: %w", err)
	}

	models := make([]llmtypes.Model, 0, len(payload.Models))
	for _, item := range payload.Models {
		if !supportsGenerateContent(item.SupportedGenerationMethods) {
			continue
		}
		modelID := strings.TrimPrefix(strings.TrimSpace(item.Name), "models/")
		if modelID == "" {
			continue
		}
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		models = append(models, llmtypes.Model{
			ID:          modelID,
			DisplayName: displayName,
			IsEnabled:   true,
			Capabilities: llmtypes.ModelCapabilities{
				SupportsStructuredOutput:   true,
				SupportsStreaming:          true,
				SupportsSystemInstructions: true,
				SupportsVision:             true,
				MaxContextTokens:           item.InputTokenLimit,
				MaxOutputTokens:            item.OutputTokenLimit,
			},
		})
	}

	return models, nil
}

func (c *Client) GenerateSQL(ctx context.Context, req llmtypes.GenerateSQLRequest) (*llmtypes.GenerateSQLResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, xerrors.New("model is required")
	}

	requestBody := generateRequest{
		SystemInstruction: content{
			Parts: []part{{Text: llmcore.GenerateSQLSystemPrompt(req)}},
		},
		Contents: geminiContents(req),
		GenerationConfig: generationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   geminiSchema(llmcore.GenerateSQLSchema()),
		},
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal gemini generate request: %w", err)
	}

	endpoint := c.baseURL + "/v1beta/models/" + url.PathEscape(req.Model) + ":generateContent?key=" + url.QueryEscape(c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create gemini generate request: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call gemini generate api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read gemini generate response: %w", err)
	}
	if err := decodeHTTPError("gemini generate api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload generateResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode gemini generate response: %w", err)
	}
	if len(payload.Candidates) == 0 || len(payload.Candidates[0].Content.Parts) == 0 {
		return nil, xerrors.New("gemini generate response did not include candidate text")
	}

	responseText := strings.TrimSpace(payload.Candidates[0].Content.Parts[0].Text)
	if responseText == "" {
		return nil, xerrors.New("gemini generate response did not include candidate text")
	}

	var finishReason *string
	if strings.TrimSpace(payload.Candidates[0].FinishReason) != "" {
		finishReason = &payload.Candidates[0].FinishReason
	}

	return llmcore.ParseGenerateSQLResponse(rawRequest, rawResponse, responseText, &llmtypes.Usage{
		InputTokens:  payload.UsageMetadata.PromptTokenCount,
		OutputTokens: payload.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  payload.UsageMetadata.TotalTokenCount,
	}, finishReason)
}

func (c *Client) GenerateAnswer(ctx context.Context, req llmtypes.GenerateAnswerRequest) (*llmtypes.GenerateAnswerResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, xerrors.New("model is required")
	}

	requestBody := generateRequest{
		SystemInstruction: content{
			Parts: []part{{Text: llmcore.GenerateAnswerSystemPrompt(req)}},
		},
		Contents: geminiAnswerContents(req),
		GenerationConfig: generationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   geminiSchema(llmcore.GenerateAnswerSchema()),
		},
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal gemini answer request: %w", err)
	}

	endpoint := c.baseURL + "/v1beta/models/" + url.PathEscape(req.Model) + ":generateContent?key=" + url.QueryEscape(c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create gemini answer request: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call gemini generate api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read gemini answer response: %w", err)
	}
	if err := decodeHTTPError("gemini generate api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload generateResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode gemini answer response: %w", err)
	}
	if len(payload.Candidates) == 0 || len(payload.Candidates[0].Content.Parts) == 0 {
		return nil, xerrors.New("gemini answer response did not include candidate text")
	}

	responseText := strings.TrimSpace(payload.Candidates[0].Content.Parts[0].Text)
	if responseText == "" {
		return nil, xerrors.New("gemini answer response did not include candidate text")
	}

	var finishReason *string
	if strings.TrimSpace(payload.Candidates[0].FinishReason) != "" {
		finishReason = &payload.Candidates[0].FinishReason
	}

	return llmcore.ParseGenerateAnswerResponse(rawRequest, rawResponse, responseText, &llmtypes.Usage{
		InputTokens:  payload.UsageMetadata.PromptTokenCount,
		OutputTokens: payload.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  payload.UsageMetadata.TotalTokenCount,
	}, finishReason)
}

func (c *Client) GenerateConversationTitle(ctx context.Context, req llmtypes.GenerateConversationTitleRequest) (*llmtypes.GenerateConversationTitleResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, xerrors.New("model is required")
	}

	requestBody := generateRequest{
		SystemInstruction: content{
			Parts: []part{{Text: llmcore.GenerateConversationTitleSystemPrompt(req)}},
		},
		Contents: geminiConversationTitleContents(req),
		GenerationConfig: generationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   geminiSchema(llmcore.GenerateConversationTitleSchema()),
		},
	}

	rawRequest, err := json.Marshal(requestBody)
	if err != nil {
		return nil, xerrors.Newf("failed to marshal gemini title request: %w", err)
	}

	endpoint := c.baseURL + "/v1beta/models/" + url.PathEscape(req.Model) + ":generateContent?key=" + url.QueryEscape(c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawRequest))
	if err != nil {
		return nil, xerrors.Newf("failed to create gemini title request: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, xerrors.Newf("failed to call gemini generate api: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read gemini title response: %w", err)
	}
	if err := decodeHTTPError("gemini generate api", resp.StatusCode, rawResponse); err != nil {
		return nil, err
	}

	var payload generateResponse
	if err := json.Unmarshal(rawResponse, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode gemini title response: %w", err)
	}
	if len(payload.Candidates) == 0 || len(payload.Candidates[0].Content.Parts) == 0 {
		return nil, xerrors.New("gemini title response did not include candidate text")
	}

	responseText := strings.TrimSpace(payload.Candidates[0].Content.Parts[0].Text)
	if responseText == "" {
		return nil, xerrors.New("gemini title response did not include candidate text")
	}

	var finishReason *string
	if strings.TrimSpace(payload.Candidates[0].FinishReason) != "" {
		finishReason = &payload.Candidates[0].FinishReason
	}

	return llmcore.ParseGenerateConversationTitleResponse(rawRequest, rawResponse, responseText, &llmtypes.Usage{
		InputTokens:  payload.UsageMetadata.PromptTokenCount,
		OutputTokens: payload.UsageMetadata.CandidatesTokenCount,
		TotalTokens:  payload.UsageMetadata.TotalTokenCount,
	}, finishReason)
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

func supportsGenerateContent(methods []string) bool {
	for _, method := range methods {
		if method == "generateContent" {
			return true
		}
	}
	return false
}

func geminiContents(req llmtypes.GenerateSQLRequest) []content {
	promptMessages := llmcore.GenerateSQLMessages(req)
	contents := make([]content, 0, len(promptMessages))
	for _, promptMessage := range promptMessages {
		contents = append(contents, content{
			Role:  normalizeGeminiRole(promptMessage.Role),
			Parts: []part{{Text: promptMessage.Content}},
		})
	}
	return contents
}

func geminiAnswerContents(req llmtypes.GenerateAnswerRequest) []content {
	promptMessages := llmcore.GenerateAnswerMessages(req)
	contents := make([]content, 0, len(promptMessages))
	for _, promptMessage := range promptMessages {
		contents = append(contents, content{
			Role:  normalizeGeminiRole(promptMessage.Role),
			Parts: []part{{Text: promptMessage.Content}},
		})
	}
	return contents
}

func geminiConversationTitleContents(req llmtypes.GenerateConversationTitleRequest) []content {
	promptMessages := llmcore.GenerateConversationTitleMessages(req)
	contents := make([]content, 0, len(promptMessages))
	for _, promptMessage := range promptMessages {
		contents = append(contents, content{
			Role:  normalizeGeminiRole(promptMessage.Role),
			Parts: []part{{Text: promptMessage.Content}},
		})
	}
	return contents
}

func geminiSchema(schema map[string]any) map[string]any {
	converted := make(map[string]any, len(schema))
	for key, value := range schema {
		switch key {
		case "type":
			schemaType, nullable := geminiSchemaType(value)
			if schemaType != "" {
				converted[key] = schemaType
			}
			if nullable {
				converted["nullable"] = true
			}
		case "properties":
			properties, ok := value.(map[string]any)
			if !ok {
				continue
			}
			convertedProperties := make(map[string]any, len(properties))
			for propertyName, propertySchema := range properties {
				propertyMap, ok := propertySchema.(map[string]any)
				if !ok {
					continue
				}
				convertedProperties[propertyName] = geminiSchema(propertyMap)
			}
			converted[key] = convertedProperties
		case "items":
			items, ok := value.(map[string]any)
			if !ok {
				continue
			}
			converted[key] = geminiSchema(items)
		case "additionalProperties":
			continue
		default:
			converted[key] = value
		}
	}
	return converted
}

func geminiSchemaType(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.ToUpper(typed), false
	case []string:
		nullable := containsString(typed, "null")
		for _, item := range typed {
			if item != "null" {
				return strings.ToUpper(item), nullable
			}
		}
	case []any:
		nullable := false
		for _, item := range typed {
			itemType, ok := item.(string)
			if !ok {
				continue
			}
			if itemType == "null" {
				nullable = true
				continue
			}
			return strings.ToUpper(itemType), nullable
		}
	}
	return "", false
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func normalizeGeminiRole(role string) string {
	switch role {
	case "assistant":
		return "model"
	default:
		return "user"
	}
}
