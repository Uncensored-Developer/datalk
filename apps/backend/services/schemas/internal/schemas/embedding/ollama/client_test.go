package ollama

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_EmbedTexts(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, err := NewClient("http://ollama.local", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "/api/embed", req.URL.Path)
				return jsonResponse(http.StatusOK, fmt.Sprintf(`{"embeddings":[%s,%s]}`, vectorJSON(0.1), vectorJSON(0.2))), nil
			}),
		})
		require.NoError(t, err)

		embeddings, err := client.EmbedTexts(t.Context(), []string{"users", "orders"})
		require.NoError(t, err)
		require.Len(t, embeddings, 2)
		assert.Len(t, embeddings[0], EmbeddingDimension)
		assert.Len(t, embeddings[1], EmbeddingDimension)
	})

	t.Run("unexpected dimension", func(t *testing.T) {
		client, err := NewClient("http://ollama.local", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusOK, `{"embeddings":[[0.1,0.2]]}`), nil
			}),
		})
		require.NoError(t, err)

		_, err = client.EmbedTexts(t.Context(), []string{"users"})
		require.ErrorIs(t, err, ErrUnexpectedEmbeddingLength)
	})

	t.Run("provider error", func(t *testing.T) {
		client, err := NewClient("http://ollama.local", &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusBadRequest, `{"error":"model not found"}`), nil
			}),
		})
		require.NoError(t, err)

		_, err = client.EmbedTexts(t.Context(), []string{"users"})
		require.ErrorContains(t, err, "model not found")
	})
}

func TestClient_Check(t *testing.T) {
	client, err := NewClient("http://ollama.local", &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, fmt.Sprintf(`{"embeddings":[%s]}`, vectorJSON(0.3))), nil
		}),
		Timeout: time.Second,
	})
	require.NoError(t, err)

	require.NoError(t, client.Check(context.Background()))
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func vectorJSON(value float32) string {
	vector := make([]byte, 0, EmbeddingDimension*4)
	vector = append(vector, '[')
	for i := 0; i < EmbeddingDimension; i++ {
		if i > 0 {
			vector = append(vector, ',')
		}
		vector = append(vector, []byte(fmt.Sprintf("%0.1f", value))...)
	}
	vector = append(vector, ']')
	return string(vector)
}
