package teams

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

func makeChannel(t *testing.T, securityToken string) *TeamsChannel {
	t.Helper()
	auth := map[string]any{
		"app_id":       "test-app-id",
		"app_password": "test-app-password",
	}
	if securityToken != "" {
		auth["security_token"] = securityToken
	}
	ch, err := NewTeamsChannel(config.ChannelConfig{Auth: auth}, zerolog.Nop())
	if err != nil {
		t.Fatalf("NewTeamsChannel: %v", err)
	}
	ch.SetConnected(true)
	return ch
}

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "HMAC " + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func buildActivityBody(t *testing.T) []byte {
	t.Helper()
	activity := map[string]any{
		"type": "message",
		"id":   "act-1",
		"text": "hello teams",
		"from": map[string]string{"id": "u1", "name": "User One"},
		"conversation": map[string]string{"id": "conv-1"},
		"serviceUrl":   "https://smba.trafficmanager.net/",
	}
	b, _ := json.Marshal(activity)
	return b
}

func TestTeams_HMAC_MatchingSignature_Returns202(t *testing.T) {
	const secret = "my-secret-token"
	ch := makeChannel(t, secret)

	body := buildActivityBody(t)
	req := httptest.NewRequest(http.MethodPost, "/channels/teams/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", signBody(body, secret))
	w := httptest.NewRecorder()

	ch.HandleMessage(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTeams_HMAC_WrongSignature_Returns401(t *testing.T) {
	ch := makeChannel(t, "correct-secret")

	body := buildActivityBody(t)
	req := httptest.NewRequest(http.MethodPost, "/channels/teams/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", signBody(body, "wrong-secret"))
	w := httptest.NewRecorder()

	ch.HandleMessage(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestTeams_HMAC_MissingHeader_Returns401(t *testing.T) {
	ch := makeChannel(t, "some-secret")

	body := buildActivityBody(t)
	req := httptest.NewRequest(http.MethodPost, "/channels/teams/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()

	ch.HandleMessage(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when header absent, got %d", w.Code)
	}
}

func TestTeams_HMAC_NotConfigured_BackwardCompat_Returns202(t *testing.T) {
	// No security_token configured — HMAC check is skipped (opt-in, backward compat).
	ch := makeChannel(t, "")

	body := buildActivityBody(t)
	req := httptest.NewRequest(http.MethodPost, "/channels/teams/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header — should still succeed
	w := httptest.NewRecorder()

	ch.HandleMessage(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202 when security_token not configured, got %d: %s", w.Code, w.Body.String())
	}
}
