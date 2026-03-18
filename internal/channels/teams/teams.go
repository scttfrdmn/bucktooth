// Package teams implements a Microsoft Teams channel adapter using the
// Bot Framework REST API.
//
// Incoming messages arrive via an HTTP webhook that Teams POSTs to
// /channels/teams/messages. Replies are sent back via the Bot Framework
// REST API using an OAuth2 bearer token obtained from Microsoft Identity.
//
// Required configuration (auth block in the channel config):
//
//	app_id       — Azure AD app (client) ID
//	app_password — Azure AD app (client) secret
//
// Optional configuration (options block):
//
//	tenant_id — Azure AD tenant; defaults to "botframework.com"
package teams

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

// botTokenURL is the Microsoft Identity token endpoint for Bot Framework.
const botTokenURL = "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token"

// botScope is the OAuth2 scope required for Bot Framework REST API calls.
const botScope = "https://api.botframework.com/.default"

// TeamsChannel implements the channels.Channel interface for Microsoft Teams.
type TeamsChannel struct {
	*channels.BaseChannel
	appID         string
	appPassword   string
	securityToken string // optional; when non-empty, inbound activities must carry a valid HMAC-SHA256 signature
	logger        zerolog.Logger
	httpClient    *http.Client

	// tokenMu guards accessToken and tokenExpiry.
	tokenMu     sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewTeamsChannel creates a new TeamsChannel from the provided config.
func NewTeamsChannel(cfg config.ChannelConfig, logger zerolog.Logger) (*TeamsChannel, error) {
	appID, _ := cfg.Auth["app_id"].(string)
	appPassword, _ := cfg.Auth["app_password"].(string)
	if appID == "" || appPassword == "" {
		return nil, fmt.Errorf("teams: app_id and app_password are required")
	}

	securityToken, _ := cfg.Auth["security_token"].(string)

	base := channels.NewBaseChannel("teams", logger, 100)

	return &TeamsChannel{
		BaseChannel:   base,
		appID:         appID,
		appPassword:   appPassword,
		securityToken: securityToken,
		logger:        logger.With().Str("channel", "teams").Logger(),
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// validateHMAC checks the Bot Framework outgoing-webhook HMAC-SHA256 signature.
// The Authorization header must be "HMAC <base64-encoded-signature>" where the
// signature is HMAC-SHA256(body, securityToken).
// Returns true when securityToken is empty (opt-in, backward compatible).
func (t *TeamsChannel) validateHMAC(r *http.Request, body []byte) bool {
	if t.securityToken == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "HMAC ") {
		return false
	}
	sigB64 := strings.TrimPrefix(auth, "HMAC ")
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(t.securityToken))
	mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}

// Connect marks the channel as connected. Actual message reception is driven
// by the HTTP webhook registered externally via HandleMessage.
func (t *TeamsChannel) Connect(_ context.Context) error {
	t.SetConnected(true)
	t.logger.Info().Msg("teams channel connected (awaiting webhook messages)")
	return nil
}

// Disconnect marks the channel as disconnected and drains the message queue.
func (t *TeamsChannel) Disconnect() error {
	t.SetConnected(false)
	t.Close()
	t.logger.Info().Msg("teams channel disconnected")
	return nil
}

// ReceiveMessages returns the inbound message queue channel.
func (t *TeamsChannel) ReceiveMessages(_ context.Context) (<-chan *channels.Message, error) {
	if !t.IsConnected() {
		return nil, channels.ErrNotConnected
	}
	return t.MessageQueue(), nil
}

// SendMessage sends a reply to the Teams conversation via the Bot Framework REST API.
// msg.Metadata must contain:
//
//	"teams_service_url"    — the serviceUrl from the incoming Activity
//	"teams_conversation_id" — the conversation.id from the incoming Activity
//	"teams_activity_id"    — the id of the incoming Activity (used for the reply URL)
func (t *TeamsChannel) SendMessage(ctx context.Context, msg *channels.Message) error {
	if !t.IsConnected() {
		return channels.ErrNotConnected
	}

	serviceURL, _ := msg.Metadata["teams_service_url"].(string)
	conversationID, _ := msg.Metadata["teams_conversation_id"].(string)
	activityID, _ := msg.Metadata["teams_activity_id"].(string)
	if serviceURL == "" || conversationID == "" {
		return fmt.Errorf("teams: missing teams_service_url or teams_conversation_id in message metadata")
	}

	token, err := t.getBearerToken(ctx)
	if err != nil {
		return fmt.Errorf("teams: failed to get bearer token: %w", err)
	}

	// Build the reply URL.
	replyURL := fmt.Sprintf("%s/v3/conversations/%s/activities",
		strings.TrimRight(serviceURL, "/"),
		url.PathEscape(conversationID),
	)
	if activityID != "" {
		replyURL += "/" + url.PathEscape(activityID)
	}

	activity := map[string]interface{}{
		"type": "message",
		"text": msg.Content,
	}

	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("teams: marshal activity: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, replyURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("teams: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("teams: post activity: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("teams: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// HandleMessage is the HTTP handler for POST /channels/teams/messages.
// It should be registered with the gateway's HTTP server before Start().
func (t *TeamsChannel) HandleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body once so it can be used for both HMAC validation and JSON decoding.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Validate HMAC signature when a security token is configured.
	if !t.validateHMAC(r, body) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var activity struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Text string `json:"text"`
		From struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"from"`
		Conversation struct {
			ID string `json:"id"`
		} `json:"conversation"`
		ServiceURL string `json:"serviceUrl"`
	}

	if err := json.Unmarshal(body, &activity); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Only handle message activities.
	if !strings.EqualFold(activity.Type, "message") {
		w.WriteHeader(http.StatusOK)
		return
	}

	if strings.TrimSpace(activity.Text) == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	msg := &channels.Message{
		ID:        activity.ID,
		ChannelID: "teams",
		UserID:    activity.From.ID,
		Username:  activity.From.Name,
		Content:   activity.Text,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"teams_service_url":     activity.ServiceURL,
			"teams_conversation_id": activity.Conversation.ID,
			"teams_activity_id":     activity.ID,
		},
	}

	if err := t.QueueMessage(msg); err != nil {
		t.logger.Error().Err(err).Str("activity_id", activity.ID).Msg("failed to queue teams message")
		http.Error(w, "queue full", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// getBearerToken returns a cached OAuth2 token, refreshing it if it has expired.
func (t *TeamsChannel) getBearerToken(ctx context.Context) (string, error) {
	t.tokenMu.Lock()
	defer t.tokenMu.Unlock()

	if t.accessToken != "" && time.Now().Before(t.tokenExpiry) {
		return t.accessToken, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", t.appID)
	data.Set("client_secret", t.appPassword)
	data.Set("scope", botScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, botTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	t.accessToken = tokenResp.AccessToken
	// Refresh 60s before actual expiry to avoid races.
	t.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return t.accessToken, nil
}
