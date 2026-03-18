//go:build integration

package harness_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// gatewayURL returns GATEWAY_URL or skips the test.
func gatewayURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("GATEWAY_URL")
	if u == "" {
		t.Skip("GATEWAY_URL not set — skipping integration tests")
	}
	return u
}

func TestHealthEndpoint(t *testing.T) {
	base := gatewayURL(t)

	resp, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Fatalf("expected status=healthy, got %v", body["status"])
	}
}

func TestSendAndReceiveEcho(t *testing.T) {
	base := gatewayURL(t)

	payload := `{"user_id":"tester","content":"hello harness"}`
	resp, err := http.Post(
		base+"/test/send",
		"application/json",
		bytes.NewBufferString(payload),
	)
	if err != nil {
		t.Fatalf("POST /test/send failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	// Poll /test/responses up to 5s.
	var responses []string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		responses = pollResponses(t, base)
		if len(responses) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if len(responses) == 0 {
		t.Fatal("no response received within 5s")
	}
	if !strings.Contains(responses[0], "echo: hello harness") {
		t.Fatalf("expected echo response, got: %q", responses[0])
	}
}

func TestMultipleMessages(t *testing.T) {
	base := gatewayURL(t)

	messages := []string{"msg-one", "msg-two", "msg-three"}
	for _, content := range messages {
		body := fmt.Sprintf(`{"user_id":"tester","content":"%s"}`, content)
		resp, err := http.Post(base+"/test/send", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("POST /test/send failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected 202, got %d", resp.StatusCode)
		}
	}

	// Poll until we have 3 responses (up to 10s for 3 sequential LLM calls).
	var all []string
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && len(all) < len(messages) {
		batch := pollResponses(t, base)
		all = append(all, batch...)
		if len(all) < len(messages) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if len(all) != len(messages) {
		t.Fatalf("expected %d responses, got %d: %v", len(messages), len(all), all)
	}
	for i, msg := range messages {
		expected := "echo: " + msg
		if !strings.Contains(all[i], expected) {
			t.Fatalf("response[%d]: expected %q, got %q", i, expected, all[i])
		}
	}
}

// pollResponses does a single GET /test/responses and returns the slice.
func pollResponses(t *testing.T, base string) []string {
	t.Helper()
	resp, err := http.Get(base + "/test/responses")
	if err != nil {
		t.Fatalf("GET /test/responses failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var responses []string
	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		t.Fatalf("failed to decode responses: %v", err)
	}
	return responses
}

func TestModelsEndpoint(t *testing.T) {
	base := gatewayURL(t)

	resp, err := http.Get(base + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Object string `json:"object"`
		Data   []struct {
			ID     string `json:"id"`
			Object string `json:"object"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode /v1/models response: %v", err)
	}
	if body.Object != "list" {
		t.Fatalf("expected object=list, got %q", body.Object)
	}
	if len(body.Data) == 0 {
		t.Fatal("expected at least one model in data array")
	}
	for i, m := range body.Data {
		if m.ID == "" {
			t.Fatalf("model[%d] has empty id", i)
		}
		if m.Object != "model" {
			t.Fatalf("model[%d] expected object=model, got %q", i, m.Object)
		}
	}
}

func TestSkillsDepsEndpoint(t *testing.T) {
	base := gatewayURL(t)

	resp, err := http.Get(base + "/admin/skills/deps")
	if err != nil {
		t.Fatalf("GET /admin/skills/deps failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Response must decode as a JSON array (empty is OK when skills are disabled).
	var result []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode /admin/skills/deps as array: %v", err)
	}
}

func TestUsageEndpoint(t *testing.T) {
	base := gatewayURL(t)

	// Send a message first to ensure at least one LLM call has been made.
	payload := `{"user_id":"harness-usage-test","content":"ping"}`
	_, _ = http.Post(base+"/test/send", "application/json", bytes.NewBufferString(payload))
	// Wait briefly for the response to be processed.
	time.Sleep(500 * time.Millisecond)

	resp, err := http.Get(base + "/v1/usage")
	if err != nil {
		t.Fatalf("GET /v1/usage failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		TotalCostUSD   float64 `json:"total_cost_usd"`
		TotalTokensIn  uint64  `json:"total_tokens_in"`
		TotalTokensOut uint64  `json:"total_tokens_out"`
		ByModel        []any   `json:"by_model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode /v1/usage response: %v", err)
	}
	if body.TotalCostUSD < 0 {
		t.Fatalf("total_cost_usd must be >= 0, got %f", body.TotalCostUSD)
	}
	if body.ByModel == nil {
		t.Fatal("by_model must be a JSON array (empty is OK)")
	}
}

func TestSignalChannelUnavailable(t *testing.T) {
	// If SIGNAL_PHONE is not set, skip — this test requires explicit opt-in.
	if os.Getenv("SIGNAL_PHONE") == "" {
		t.Skip("SIGNAL_PHONE not set — skipping Signal channel test")
	}

	base := gatewayURL(t)

	// The gateway should still be healthy even if the Signal channel cannot connect.
	resp, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// A non-200 status is acceptable; what we're checking is that the gateway
	// did not crash and is still accepting HTTP traffic.
	if resp.StatusCode == 0 {
		t.Fatal("expected a valid HTTP status code, got 0")
	}
}
