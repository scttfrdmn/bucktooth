package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	agenmem "github.com/scttfrdmn/agenkit/agenkit-go/memory"
)

// OpenAIEmbeddingProvider implements agenmem.EmbeddingProvider via the OpenAI embeddings API.
// It is compatible with any OpenAI-compatible embeddings endpoint.
type OpenAIEmbeddingProvider struct {
	baseURL string
	apiKey  string
	model   string
	dim     int
	client  *http.Client
}

// NewOpenAIEmbeddingProvider creates a new embedding provider.
// baseURL defaults to "https://api.openai.com/v1" when empty.
// model defaults to "text-embedding-3-small" when empty.
func NewOpenAIEmbeddingProvider(baseURL, apiKey, model string) *OpenAIEmbeddingProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIEmbeddingProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Embed generates an embedding vector for the given text.
func (p *OpenAIEmbeddingProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	body, err := json.Marshal(map[string]any{
		"input": text,
		"model": p.model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("embed API returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode embed response: %w", err)
	}
	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embed response contained no embedding")
	}

	embedding := result.Data[0].Embedding
	p.dim = len(embedding)
	return embedding, nil
}

// Dimension returns the embedding dimension (set after the first Embed call).
func (p *OpenAIEmbeddingProvider) Dimension() int {
	return p.dim
}

// VectorStore implements BuckTooth's memory.Store backed by agenkit-go VectorMemory.
// It provides semantic retrieval using cosine similarity over stored message embeddings.
type VectorStore struct {
	vm *agenmem.VectorMemory
}

// NewVectorStore creates a VectorStore using the given embedding provider.
// An in-memory cosine-similarity backend is used for vector storage.
func NewVectorStore(embedProvider agenmem.EmbeddingProvider) *VectorStore {
	return &VectorStore{
		vm: agenmem.NewVectorMemory(embedProvider, nil),
	}
}

// AddMessage stores a message in the vector memory under the given userID.
func (s *VectorStore) AddMessage(ctx context.Context, userID string, msg Message) error {
	aMsg := agenkit.Message{
		Role:    msg.Role,
		Content: msg.Content,
	}
	return s.vm.Store(ctx, userID, aMsg, nil)
}

// GetHistory retrieves the most recent messages for a user via vector memory.
func (s *VectorStore) GetHistory(ctx context.Context, userID string, limit int) ([]Message, error) {
	msgs, err := s.vm.Retrieve(ctx, userID, agenmem.RetrieveOptions{Limit: &limit})
	if err != nil {
		return nil, fmt.Errorf("vector memory retrieve failed: %w", err)
	}
	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, Message{
			Role:      m.Role,
			Content:   m.ContentString(),
			Timestamp: time.Now(),
		})
	}
	return result, nil
}

// ClearHistory removes all stored messages for a user.
func (s *VectorStore) ClearHistory(ctx context.Context, userID string) error {
	return s.vm.Clear(ctx, userID)
}

// Close is a no-op for the in-memory vector backend.
func (s *VectorStore) Close() error {
	return nil
}
