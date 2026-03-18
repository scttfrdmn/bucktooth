package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	"golang.org/x/net/html"
)

const defaultWebFetchMaxBytes = 512 * 1024

// WebFetchTool fetches a URL and returns its text content.
// HTML pages are stripped of tags; other content types are returned as-is (truncated).
type WebFetchTool struct {
	httpClient *http.Client
	maxBytes   int
}

// NewWebFetchTool creates a new WebFetchTool.
// maxBytes controls the response size limit; 0 uses the default (512 KB).
func NewWebFetchTool(maxBytes int) *WebFetchTool {
	if maxBytes <= 0 {
		maxBytes = defaultWebFetchMaxBytes
	}
	return &WebFetchTool{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		maxBytes:   maxBytes,
	}
}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return `Fetch a URL and return its text content. Parameters: {"url":"<url>","max_bytes":<n>}`
}

// Execute fetches the given URL and returns its text content.
func (t *WebFetchTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
	// Support ReActAgent wrapping params in {"input": "<json string>"}
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		} else {
			params = map[string]any{"url": raw}
		}
	}

	rawURL, _ := params["url"].(string)
	if rawURL == "" {
		return agenkit.NewToolError("missing required parameter: url"), nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return agenkit.NewToolError(fmt.Sprintf("invalid URL (must be http or https): %s", rawURL)), nil
	}

	maxBytes := t.maxBytes
	if v, ok := params["max_bytes"].(float64); ok && v > 0 {
		maxBytes = int(v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to create request: %v", err)), nil
	}
	req.Header.Set("User-Agent", "BuckTooth/1.0 (web_fetch tool)")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("HTTP request failed: %v", err)), nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return agenkit.NewToolError(fmt.Sprintf("HTTP %d from %s", resp.StatusCode, rawURL)), nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to read response body: %v", err)), nil
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		text := extractHTMLText(body)
		return agenkit.NewToolResult(text), nil
	}

	return agenkit.NewToolResult(string(body)), nil
}

// extractHTMLText strips HTML tags and returns plain text, skipping <script> and <style>.
func extractHTMLText(data []byte) string {
	tokenizer := html.NewTokenizer(strings.NewReader(string(data)))
	var sb strings.Builder
	skip := 0 // depth counter for script/style subtrees

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return strings.TrimSpace(sb.String())

		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokenizer.TagName()
			tag := string(name)
			if tag == "script" || tag == "style" {
				if tt == html.StartTagToken {
					skip++
				}
			}

		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			tag := string(name)
			if tag == "script" || tag == "style" {
				if skip > 0 {
					skip--
				}
			}

		case html.TextToken:
			if skip == 0 {
				text := strings.TrimSpace(string(tokenizer.Text()))
				if text != "" {
					sb.WriteString(text)
					sb.WriteByte('\n')
				}
			}
		}
	}
}
