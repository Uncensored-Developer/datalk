package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mdobak/go-xerrors"
)

const (
	ModelName          = "nomic-embed-text"
	EmbeddingDimension = 768
)

var (
	ErrInvalidBaseURL            = xerrors.New("ollama base url cannot be empty")
	ErrUnexpectedEmbeddingCount  = xerrors.New("unexpected embedding count")
	ErrUnexpectedEmbeddingLength = xerrors.New("unexpected embedding length")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
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

func (c *Client) Check(ctx context.Context) error {
	embeddings, err := c.EmbedTexts(ctx, []string{"healthcheck"})
	if err != nil {
		return err
	}
	if len(embeddings) != 1 {
		return xerrors.Newf("%w: got %d embeddings", ErrUnexpectedEmbeddingCount, len(embeddings))
	}
	return nil
}

func (c *Client) EmbedTexts(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return [][]float32{}, nil
	}

	reqBody, err := json.Marshal(embedRequest{
		Model: ModelName,
		Input: inputs,
	})
	if err != nil {
		return nil, xerrors.Newf("failed to marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(reqBody))
	if err != nil {
		return nil, xerrors.Newf("failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, xerrors.Newf("failed to call ollama embed api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Newf("failed to read embed response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var errResp errorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, xerrors.Newf("ollama embed api failed: %s", errResp.Error)
		}
		return nil, xerrors.Newf("ollama embed api failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload embedResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, xerrors.Newf("failed to decode embed response: %w", err)
	}

	if len(payload.Embeddings) != len(inputs) {
		return nil, xerrors.Newf("%w: expected %d, got %d", ErrUnexpectedEmbeddingCount, len(inputs), len(payload.Embeddings))
	}

	for index, embedding := range payload.Embeddings {
		if len(embedding) != EmbeddingDimension {
			return nil, xerrors.Newf("%w for item %d: expected %d, got %d", ErrUnexpectedEmbeddingLength, index, EmbeddingDimension, len(embedding))
		}
	}

	return payload.Embeddings, nil
}

func QualifiedName(namespace, objectName string) string {
	if namespace == "" {
		return objectName
	}
	return fmt.Sprintf("%s.%s", namespace, objectName)
}
