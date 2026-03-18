package gateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/memory"
)

// buildHTTPServerForCompletions returns an HTTPServer with a stub AgentRouter attached.
func buildHTTPServerForCompletions(t *testing.T, apiToken string) *HTTPServer {
	t.Helper()
	store := memory.NewInMemoryStore()
	stats := NewStats()
	logger := zerolog.Nop()
	router, _, err := NewAgentRouter(
		config.AgentConfig{LLMProvider: "stub", StubResponse: "test reply", MaxHistory: 10},
		config.GatewayConfig{},
		config.SkillsConfig{},
		store, nil, stats, logger,
	)
	if err != nil {
		t.Fatalf("NewAgentRouter: %v", err)
	}
	registry := channels.NewChannelRegistry()
	srv := NewHTTPServer(0, registry, router, stats, logger)
	if apiToken != "" {
		srv.SetAPIToken(apiToken)
	}
	return srv
}

func TestCompletions_NonStreaming_ValidResponse(t *testing.T) {
	srv := buildHTTPServerForCompletions(t, "")

	body, _ := json.Marshal(openAICompletionRequest{
		Model: "claude-sonnet",
		Messages: []openAIChatMessage{
			{Role: "user", Content: "hello"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp openAICompletionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Object != "chat.completion" {
		t.Errorf("expected object=chat.completion, got %q", resp.Object)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice")
	}
	if resp.Choices[0].Message.Role != "assistant" {
		t.Errorf("expected role=assistant, got %q", resp.Choices[0].Message.Role)
	}
	if resp.Choices[0].Message.Content == "" {
		t.Error("expected non-empty content in choice")
	}
	if !strings.HasPrefix(resp.ID, "chatcmpl-") {
		t.Errorf("expected ID to start with chatcmpl-, got %q", resp.ID)
	}
}

func TestCompletions_Streaming_SSELines(t *testing.T) {
	srv := buildHTTPServerForCompletions(t, "")

	body, _ := json.Marshal(openAICompletionRequest{
		Model:  "claude-sonnet",
		Stream: true,
		Messages: []openAIChatMessage{
			{Role: "user", Content: "count"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Content-Type must be text/event-stream
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %q", ct)
	}

	rawBody := w.Body.String()
	if !strings.Contains(rawBody, "data: [DONE]") {
		t.Errorf("expected data: [DONE] sentinel, got: %q", rawBody)
	}

	// Parse non-DONE data lines as openAIStreamChunk.
	scanner := bufio.NewScanner(strings.NewReader(rawBody))
	chunkCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("parse chunk %q: %v", payload, err)
		}
		if chunk.Object != "chat.completion.chunk" {
			t.Errorf("expected object=chat.completion.chunk, got %q", chunk.Object)
		}
		chunkCount++
	}
	if chunkCount == 0 {
		t.Error("expected at least one chunk before [DONE]")
	}
}

func TestCompletions_Unauthorized_WhenTokenConfigured(t *testing.T) {
	srv := buildHTTPServerForCompletions(t, "secret-token")

	body, _ := json.Marshal(openAICompletionRequest{
		Model:    "claude-sonnet",
		Messages: []openAIChatMessage{{Role: "user", Content: "hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header.
	w := httptest.NewRecorder()

	// Route through apiTokenMiddleware as Start() does.
	handler := srv.apiTokenMiddleware(http.HandlerFunc(srv.handleCompletions))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCompletions_Authorized_WithValidToken(t *testing.T) {
	srv := buildHTTPServerForCompletions(t, "secret-token")

	body, _ := json.Marshal(openAICompletionRequest{
		Model:    "claude-sonnet",
		Messages: []openAIChatMessage{{Role: "user", Content: "hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	w := httptest.NewRecorder()

	handler := srv.apiTokenMiddleware(http.HandlerFunc(srv.handleCompletions))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCompletions_NoUserMessage_BadRequest(t *testing.T) {
	srv := buildHTTPServerForCompletions(t, "")

	body, _ := json.Marshal(openAICompletionRequest{
		Model:    "claude-sonnet",
		Messages: []openAIChatMessage{{Role: "system", Content: "you are helpful"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for no user message, got %d", w.Code)
	}
}
