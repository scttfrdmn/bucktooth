package gateway

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/memory"
)

// newTestRouter creates a minimal AgentRouter backed by a StubLLM using the real constructor.
func newTestRouter(t *testing.T, fixedResponse string) *AgentRouter {
	t.Helper()
	store := memory.NewInMemoryStore()
	stats := NewStats()
	logger := zerolog.Nop()
	router, _, err := NewAgentRouter(
		config.AgentConfig{LLMProvider: "stub", StubResponse: fixedResponse, MaxHistory: 10},
		config.GatewayConfig{},
		config.SkillsConfig{},
		store, nil, stats, logger,
	)
	if err != nil {
		t.Fatalf("NewAgentRouter: %v", err)
	}
	return router
}

// dialTestServer starts an httptest.Server with the WebSocket handler and dials it.
func dialTestServer(t *testing.T, ws *WebSocketServer) (*websocket.Conn, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.handleWebSocket)
	srv := httptest.NewServer(mux)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		srv.Close()
		// Try without DefaultDialer to get a better error.
		dialer := &websocket.Dialer{NetDial: func(_, addr string) (net.Conn, error) { return net.Dial("tcp", addr) }}
		conn, _, err = dialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial ws: %v", err)
		}
	}
	return conn, srv
}

func TestWebSocket_NonStreaming_LegacyPath(t *testing.T) {
	router := newTestRouter(t, "hello from stub")
	ws := NewWebSocketServer(0, nil, router, false, zerolog.Nop())

	conn, srv := dialTestServer(t, ws)
	defer srv.Close()
	defer conn.Close()

	raw, _ := json.Marshal(wsInboundMsg{UserID: "alice", Content: "hi"})
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, resp, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var out wsOutboundMsg
	if err := json.Unmarshal(resp, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.UserID != "alice" {
		t.Errorf("expected user_id=alice, got %q", out.UserID)
	}
	if !strings.Contains(out.Content, "hello from stub") {
		t.Errorf("expected content to contain %q, got %q", "hello from stub", out.Content)
	}
}

func TestWebSocket_StreamFlagIgnoredWhenDisabled(t *testing.T) {
	router := newTestRouter(t, "sync response")
	// streamingEnabled=false — stream:true in message should produce wsOutboundMsg (legacy path).
	ws := NewWebSocketServer(0, nil, router, false, zerolog.Nop())

	conn, srv := dialTestServer(t, ws)
	defer srv.Close()
	defer conn.Close()

	raw, _ := json.Marshal(wsInboundMsg{UserID: "bob", Content: "test", Stream: true})
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, resp, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Should be a wsOutboundMsg (legacy), not a chunk/done frame.
	var out wsOutboundMsg
	if err := json.Unmarshal(resp, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Content == "" && out.Error == "" {
		t.Error("expected non-empty content or error in legacy response")
	}
}

func TestWebSocket_StreamingEnabled_ChunkThenDone(t *testing.T) {
	router := newTestRouter(t, "streamed answer")
	ws := NewWebSocketServer(0, nil, router, true, zerolog.Nop())

	conn, srv := dialTestServer(t, ws)
	defer srv.Close()
	defer conn.Close()

	raw, _ := json.Marshal(wsInboundMsg{UserID: "carol", Content: "hello", Stream: true})
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	gotChunk := false
	gotDone := false

	for !gotDone {
		_, resp, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var typed struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(resp, &typed); err != nil {
			t.Fatalf("unmarshal type: %v", err)
		}
		switch typed.Type {
		case "chunk":
			var chunk wsChunkMsg
			if err := json.Unmarshal(resp, &chunk); err != nil {
				t.Fatalf("unmarshal chunk: %v", err)
			}
			if chunk.UserID != "carol" {
				t.Errorf("chunk user_id: want carol, got %q", chunk.UserID)
			}
			gotChunk = true
		case "done":
			var done wsDoneMsg
			if err := json.Unmarshal(resp, &done); err != nil {
				t.Fatalf("unmarshal done: %v", err)
			}
			if done.UserID != "carol" {
				t.Errorf("done user_id: want carol, got %q", done.UserID)
			}
			gotDone = true
		case "error":
			t.Fatalf("unexpected error frame: %s", resp)
		default:
			// Non-typed frame (legacy wsOutboundMsg from non-streaming fallback) — tolerate.
			gotDone = true
		}
	}

	if !gotChunk {
		t.Error("expected at least one chunk frame before done")
	}
}

func TestWebSocket_ErrorFrameStructure(t *testing.T) {
	// Verify the wsErrorMsg JSON structure without a full network round-trip.
	frame := wsErrorMsg{Type: "error", UserID: "dave", Error: "test error"}
	b, _ := json.Marshal(frame)
	var parsed struct {
		Type   string `json:"type"`
		UserID string `json:"user_id"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Type != "error" {
		t.Errorf("expected type=error, got %q", parsed.Type)
	}
	if parsed.Error != "test error" {
		t.Errorf("expected error=test error, got %q", parsed.Error)
	}
	if parsed.UserID != "dave" {
		t.Errorf("expected user_id=dave, got %q", parsed.UserID)
	}
}
