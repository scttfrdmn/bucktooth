package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// WebSearchTool searches the web using the Brave Search REST API.
// If no API key is configured the tool returns a graceful no-op error.
type WebSearchTool struct {
	apiKey     string
	httpClient *http.Client
	maxResults int
}

// NewWebSearchTool creates a new WebSearchTool.
// apiKey may be empty; in that case Execute() returns a configuration error.
func NewWebSearchTool(apiKey string, maxResults int) *WebSearchTool {
	if maxResults <= 0 {
		maxResults = 5
	}
	return &WebSearchTool{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		maxResults: maxResults,
	}
}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return `Search the web. Parameters as JSON: {"query":"<search terms>","max_results":<n>}`
}

// Execute performs a web search and returns the top results.
func (t *WebSearchTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
	// Support ReActAgent wrapping params in {"input": "<json string>"}
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		} else {
			// Treat raw string as the query directly.
			params = map[string]any{"query": raw}
		}
	}

	if t.apiKey == "" {
		return agenkit.NewToolError("web search not configured: BRAVE_SEARCH_API_KEY is not set"), nil
	}

	query, _ := params["query"].(string)
	if query == "" {
		return agenkit.NewToolError("missing required parameter: query"), nil
	}

	maxResults := t.maxResults
	if v, ok := params["max_results"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	results, err := t.search(ctx, query, maxResults)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("search failed: %v", err)), nil
	}

	return agenkit.NewToolResult(results), nil
}

// braveSearchResponse is the minimal subset of the Brave Search API response.
type braveSearchResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

func (t *WebSearchTool) search(ctx context.Context, query string, maxResults int) ([]map[string]string, error) {
	endpoint := fmt.Sprintf(
		"https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query),
		maxResults,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Subscription-Token", t.apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("brave search API returned %d: %s", resp.StatusCode, string(body))
	}

	var searchResp braveSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]map[string]string, 0, len(searchResp.Web.Results))
	for _, r := range searchResp.Web.Results {
		results = append(results, map[string]string{
			"title":       r.Title,
			"url":         r.URL,
			"description": r.Description,
		})
	}

	return results, nil
}
