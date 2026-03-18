package signal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
)

// newTestLogger returns a discarding zerolog logger for tests.
func newTestLogger() zerolog.Logger {
	return zerolog.Nop()
}

// mockSignaldServer creates a test HTTP server that upgrades to WebSocket and
// runs serveFunc in a goroutine for each connected client.
func mockSignaldServer(t *testing.T, serveFunc func(conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close() //nolint:errcheck
		serveFunc(conn)
	}))
	return srv
}

// wsURL converts an http:// test server URL to a ws:// URL.
func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestConnect(t *testing.T) {
	srv := mockSignaldServer(t, func(conn *websocket.Conn) {
		time.Sleep(200 * time.Millisecond)
	})
	defer srv.Close()

	ch := New("+15550000001", wsURL(srv), newTestLogger())
	ctx := context.Background()
	if err := ch.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if !ch.IsConnected() {
		t.Fatal("expected IsConnected() == true after Connect")
	}
}

func TestConnectAlreadyConnected(t *testing.T) {
	srv := mockSignaldServer(t, func(conn *websocket.Conn) {
		time.Sleep(500 * time.Millisecond)
	})
	defer srv.Close()

	ch := New("+15550000002", wsURL(srv), newTestLogger())
	ctx := context.Background()
	if err := ch.Connect(ctx); err != nil {
		t.Fatalf("first Connect failed: %v", err)
	}
	if err := ch.Connect(ctx); err == nil {
		t.Fatal("expected error on second Connect, got nil")
	}
}

func TestDisconnect(t *testing.T) {
	srv := mockSignaldServer(t, func(conn *websocket.Conn) {
		time.Sleep(500 * time.Millisecond)
	})
	defer srv.Close()

	ch := New("+15550000003", wsURL(srv), newTestLogger())
	ctx := context.Background()
	if err := ch.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if err := ch.Disconnect(); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}
	if ch.IsConnected() {
		t.Fatal("expected IsConnected() == false after Disconnect")
	}
}

func TestReceiveMessage(t *testing.T) {
	notif := jsonrpcNotification{
		JSONRPC: "2.0",
		Method:  "receive",
		Params: map[string]any{
			"envelope": map[string]any{
				"source": "+15559876543",
				"dataMessage": map[string]any{
					"message": "hello from signal",
				},
			},
		},
	}
	notifBytes, _ := json.Marshal(notif)

	srv := mockSignaldServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, notifBytes)
		time.Sleep(500 * time.Millisecond)
	})
	defer srv.Close()

	ch := New("+15550000004", wsURL(srv), newTestLogger())
	ctx := context.Background()
	if err := ch.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	msgChan, err := ch.ReceiveMessages(ctx)
	if err != nil {
		t.Fatalf("ReceiveMessages: %v", err)
	}

	select {
	case msg := <-msgChan:
		if msg.Content != "hello from signal" {
			t.Fatalf("unexpected content: %q", msg.Content)
		}
		if msg.UserID != "+15559876543" {
			t.Fatalf("unexpected user ID: %q", msg.UserID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestSendMessageProducesCorrectPayload(t *testing.T) {
	received := make(chan []byte, 1)

	srv := mockSignaldServer(t, func(conn *websocket.Conn) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Logf("read error: %v", err)
			return
		}
		received <- data
		time.Sleep(200 * time.Millisecond)
	})
	defer srv.Close()

	ch := New("+15550000005", wsURL(srv), newTestLogger())
	ctx := context.Background()
	if err := ch.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	msg := &channels.Message{
		UserID:    "+15551112222",
		ChannelID: "signal",
		Content:   "test payload",
	}
	if err := ch.SendMessage(ctx, msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	select {
	case data := <-received:
		var req jsonrpcRequest
		if err := json.Unmarshal(data, &req); err != nil {
			t.Fatalf("unmarshal sent payload: %v", err)
		}
		if req.Method != "send" {
			t.Fatalf("expected method=send, got %q", req.Method)
		}
		if req.JSONRPC != "2.0" {
			t.Fatalf("expected jsonrpc=2.0, got %q", req.JSONRPC)
		}
		if req.Params["recipient"] != "+15551112222" {
			t.Fatalf("unexpected recipient: %v", req.Params["recipient"])
		}
		if req.Params["message"] != "test payload" {
			t.Fatalf("unexpected message: %v", req.Params["message"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for sent payload")
	}
}

func TestReceiveNonMessageNotification(t *testing.T) {
	typing := jsonrpcNotification{
		JSONRPC: "2.0",
		Method:  "typing",
		Params:  map[string]any{"account": "+15550000006"},
	}
	typingBytes, _ := json.Marshal(typing)

	srv := mockSignaldServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, typingBytes)
		time.Sleep(300 * time.Millisecond)
	})
	defer srv.Close()

	ch := New("+15550000006", wsURL(srv), newTestLogger())
	ctx := context.Background()
	if err := ch.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	msgChan, _ := ch.ReceiveMessages(ctx)

	select {
	case msg := <-msgChan:
		t.Fatalf("expected no message queued, got: %+v", msg)
	case <-time.After(400 * time.Millisecond):
		// correct: typing notification was silently ignored
	}
}

func TestConnectUnreachable(t *testing.T) {
	ch := New("+15550000007", "ws://localhost:19999", newTestLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := ch.Connect(ctx)
	if err == nil {
		t.Fatal("expected Connect to fail for unreachable signald")
	}
}

func TestReconnectOnWSClose(t *testing.T) {
	// The mock server closes immediately on the first connection; the receive loop
	// should attempt reconnect. We just verify it doesn't panic or block permanently.
	var attempt int32
	srv := mockSignaldServer(t, func(conn *websocket.Conn) {
		n := atomic.AddInt32(&attempt, 1)
		// Let the second+ connection stay open so the loop stabilises.
		if n >= 2 {
			time.Sleep(300 * time.Millisecond)
		}
		// First connection closes immediately (conn is closed by defer).
	})
	defer srv.Close()

	ch := New("+15550000008", wsURL(srv), newTestLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := ch.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if _, err := ch.ReceiveMessages(ctx); err != nil {
		t.Fatalf("ReceiveMessages: %v", err)
	}

	// Wait for context to expire — the loop should have tried to reconnect at least once.
	<-ctx.Done()
}
