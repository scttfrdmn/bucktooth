package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

const pdfMaxBytes = 32 * 1024 * 1024 // 32 MB

// PDFAnalysisTool analyses PDF documents via Claude's native document API.
// It POSTs directly to the Anthropic /messages endpoint with base64-encoded PDF
// content, bypassing the agenkit LLM wrapper which does not support multimodal input.
type PDFAnalysisTool struct {
	apiKey     string
	apiBase    string
	model      string
	sandboxDir string
	httpClient *http.Client
}

// NewPDFAnalysisTool creates a PDFAnalysisTool.
// apiBase defaults to "https://api.anthropic.com/v1" when empty.
func NewPDFAnalysisTool(apiKey, apiBase, model, sandboxDir string) *PDFAnalysisTool {
	if apiBase == "" {
		apiBase = "https://api.anthropic.com/v1"
	}
	if model == "" {
		model = "claude-sonnet-4-5-20250220"
	}
	return &PDFAnalysisTool{
		apiKey:     apiKey,
		apiBase:    apiBase,
		model:      model,
		sandboxDir: sandboxDir,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (t *PDFAnalysisTool) Name() string { return "pdf_analyze" }

func (t *PDFAnalysisTool) Description() string {
	return `Analyses a PDF document using Claude's document API. Pass parameters as JSON: {"source":"<file-path|url>","prompt":"<question about the PDF>"}`
}

// Execute handles the pdf_analyze tool call.
// params: {"source":"<path|url>","prompt":"<question>"}
func (t *PDFAnalysisTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
	// ReActAgent passes input as {"input": "<json string>"} — unwrap if present.
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		}
	}

	source, _ := params["source"].(string)
	prompt, _ := params["prompt"].(string)
	if source == "" {
		return agenkit.NewToolError("missing required parameter: source"), nil
	}
	if prompt == "" {
		prompt = "Please summarise this document."
	}

	pdfBytes, err := t.loadPDF(ctx, source)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to load PDF: %v", err)), nil
	}

	b64Data := base64.StdEncoding.EncodeToString(pdfBytes)

	reqBody := map[string]any{
		"model":      t.model,
		"max_tokens": 4096,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "document",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "application/pdf",
							"data":       b64Data,
						},
					},
					{
						"type": "text",
						"text": prompt,
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.apiBase+"/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", t.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "pdfs-2024-09-25")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return agenkit.NewToolError(fmt.Sprintf("Anthropic API returned %d: %s", resp.StatusCode, string(b))), nil
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	for _, block := range result.Content {
		if block.Type == "text" {
			return agenkit.NewToolResult(block.Text), nil
		}
	}
	return agenkit.NewToolError("no text content in API response"), nil
}

// loadPDF fetches PDF bytes from a URL or a sandbox-constrained file path.
func (t *PDFAnalysisTool) loadPDF(ctx context.Context, source string) ([]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := t.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		ct := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/pdf") {
			return nil, fmt.Errorf("URL did not return a PDF (Content-Type: %s)", ct)
		}
		return io.ReadAll(io.LimitReader(resp.Body, pdfMaxBytes))
	}

	sandboxDir := t.sandboxDir
	if sandboxDir == "" {
		sandboxDir = filepath.Join(os.TempDir(), "bucktooth-sandbox")
	}
	if strings.HasPrefix(sandboxDir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			sandboxDir = filepath.Join(home, sandboxDir[2:])
		}
	}

	clean := filepath.Clean(filepath.Join(sandboxDir, source))
	if !strings.HasPrefix(clean, filepath.Clean(sandboxDir)+string(os.PathSeparator)) {
		return nil, fmt.Errorf("path traversal rejected")
	}

	f, err := os.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, pdfMaxBytes))
}
