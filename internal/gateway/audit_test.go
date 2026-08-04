package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
)

func newTestHTTPServer(cfg *config.Config) *HTTPServer {
	registry := channels.NewChannelRegistry()
	stats := NewStats()
	logger := zerolog.Nop()
	h := NewHTTPServer(8080, registry, nil, stats, logger)
	h.SetConfig(cfg)
	return h
}

func auditRequest(t *testing.T, h *HTTPServer) auditResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/audit", nil)
	w := httptest.NewRecorder()
	h.handleAdminAudit(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp auditResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

func findCheck(checks []auditCheck, name string) *auditCheck {
	for i := range checks {
		if checks[i].Name == name {
			return &checks[i]
		}
	}
	return nil
}

func TestAuditEndpoint_AllPass(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.DedupEnabled = true
	cfg.Gateway.DedupWindowSize = 256
	cfg.RateLimit.Enabled = true
	cfg.RateLimit.RequestsPerMinute = 60

	h := newTestHTTPServer(cfg)
	h.SetAPIToken("secret-token")
	h.SetDashboardAuth("hunter2")

	resp := auditRequest(t, h)

	if c := findCheck(resp.Checks, "api_token"); c == nil || c.Status != "pass" {
		t.Errorf("api_token: want pass, got %+v", c)
	}
	if c := findCheck(resp.Checks, "dashboard_auth"); c == nil || c.Status != "pass" {
		t.Errorf("dashboard_auth: want pass, got %+v", c)
	}
	if c := findCheck(resp.Checks, "rate_limiting"); c == nil || c.Status != "pass" {
		t.Errorf("rate_limiting: want pass, got %+v", c)
	}
	if c := findCheck(resp.Checks, "message_dedup"); c == nil || c.Status != "pass" {
		t.Errorf("message_dedup: want pass, got %+v", c)
	}
	if c := findCheck(resp.Checks, "shell_approval"); c == nil || c.Status != "info" {
		t.Errorf("shell_approval: want info (shell disabled), got %+v", c)
	}
	if c := findCheck(resp.Checks, "tls"); c == nil || c.Status != "info" {
		t.Errorf("tls: want info, got %+v", c)
	}
	// With 4 pass + 0 warn + 0 fail = score 100
	if resp.Score != 100 {
		t.Errorf("score: want 100, got %d", resp.Score)
	}
}

func TestAuditEndpoint_AllWarn(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gateway.DedupEnabled = false
	cfg.RateLimit.Enabled = false

	h := newTestHTTPServer(cfg)
	// no api token, no dashboard auth

	resp := auditRequest(t, h)

	if c := findCheck(resp.Checks, "api_token"); c == nil || c.Status != "warn" {
		t.Errorf("api_token: want warn, got %+v", c)
	}
	if c := findCheck(resp.Checks, "dashboard_auth"); c == nil || c.Status != "warn" {
		t.Errorf("dashboard_auth: want warn, got %+v", c)
	}
	if c := findCheck(resp.Checks, "rate_limiting"); c == nil || c.Status != "warn" {
		t.Errorf("rate_limiting: want warn, got %+v", c)
	}
	if c := findCheck(resp.Checks, "message_dedup"); c == nil || c.Status != "warn" {
		t.Errorf("message_dedup: want warn, got %+v", c)
	}
	// Score = 0 pass / (0 + 4 warn + 0 fail) = 0
	if resp.Score != 0 {
		t.Errorf("score: want 0, got %d", resp.Score)
	}
}

func TestAuditEndpoint_ShellNoApproval(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Shell.Enabled = true
	cfg.Tools.Shell.Options = map[string]any{"require_approval": false}

	h := newTestHTTPServer(cfg)

	resp := auditRequest(t, h)

	if c := findCheck(resp.Checks, "shell_approval"); c == nil || c.Status != "fail" {
		t.Errorf("shell_approval: want fail, got %+v", c)
	}
}

func TestAuditEndpoint_ShellWithApproval(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Tools.Shell.Enabled = true
	cfg.Tools.Shell.Options = map[string]any{"require_approval": true}
	cfg.Gateway.DedupEnabled = true
	cfg.RateLimit.Enabled = true

	h := newTestHTTPServer(cfg)
	h.SetAPIToken("tok")
	h.SetDashboardAuth("pw")

	resp := auditRequest(t, h)

	if c := findCheck(resp.Checks, "shell_approval"); c == nil || c.Status != "pass" {
		t.Errorf("shell_approval: want pass, got %+v", c)
	}
}

func TestAuditEndpoint_ScoreCalculation(t *testing.T) {
	// 2 pass, 2 warn, 1 fail → score = 2/5 * 100 = 40
	cfg := config.DefaultConfig()
	cfg.Gateway.DedupEnabled = true // pass
	cfg.RateLimit.Enabled = false   // warn
	cfg.Tools.Shell.Enabled = true  // fail (require_approval=false)
	cfg.Tools.Shell.Options = map[string]any{"require_approval": false}

	h := newTestHTTPServer(cfg)
	h.SetAPIToken("tok") // pass
	// no dashboard auth → warn

	resp := auditRequest(t, h)
	// passes: api_token, message_dedup = 2
	// warns: dashboard_auth, rate_limiting = 2
	// fails: shell_approval = 1
	// score = 2 * 100 / (2+2+1) = 40
	if resp.Score != 40 {
		t.Errorf("score: want 40, got %d", resp.Score)
	}
}
