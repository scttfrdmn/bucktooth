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

const defaultImageMaxBytes = 5 * 1024 * 1024 // 5 MB (Anthropic per-image limit)

// supportedImageMediaTypes lists the image formats accepted by Claude's vision API.
var supportedImageMediaTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// ImageAnalysisTool analyses images via Claude's vision API.
// Like PDFAnalysisTool, it POSTs directly to the Anthropic /messages endpoint.
type ImageAnalysisTool struct {
	apiKey     string
	apiBase    string
	model      string
	maxBytes   int64
	sandboxDir string
	httpClient *http.Client
}

// NewImageAnalysisTool creates an ImageAnalysisTool.
// apiBase defaults to "https://api.anthropic.com/v1" when empty.
// maxBytes defaults to 5 MB when <= 0.
func NewImageAnalysisTool(apiKey, apiBase, model string, maxBytes int64, sandboxDir string) *ImageAnalysisTool {
	if apiBase == "" {
		apiBase = "https://api.anthropic.com/v1"
	}
	if model == "" {
		model = "claude-sonnet-4-5-20250220"
	}
	if maxBytes <= 0 {
		maxBytes = defaultImageMaxBytes
	}
	return &ImageAnalysisTool{
		apiKey:     apiKey,
		apiBase:    apiBase,
		model:      model,
		maxBytes:   maxBytes,
		sandboxDir: sandboxDir,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (t *ImageAnalysisTool) Name() string { return "image_analyze" }

func (t *ImageAnalysisTool) Description() string {
	return `Analyses an image using Claude's vision API. Pass parameters as JSON: {"source":"<file-path|url>","prompt":"<question about the image>"}`
}

// Execute handles the image_analyze tool call.
// params: {"source":"<path|url>","prompt":"<question>"}
func (t *ImageAnalysisTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
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
		prompt = "Please describe this image."
	}

	imgBytes, mediaType, err := t.loadImage(ctx, source)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to load image: %v", err)), nil
	}

	b64Data := base64.StdEncoding.EncodeToString(imgBytes)

	reqBody := map[string]any{
		"model":      t.model,
		"max_tokens": 2048,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": mediaType,
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

// loadImage fetches image bytes and detects the media type, from either a URL
// or a sandbox-constrained local file path.
func (t *ImageAnalysisTool) loadImage(ctx context.Context, source string) ([]byte, string, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := t.httpClient.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		ct := strings.Split(resp.Header.Get("Content-Type"), ";")[0]
		if !supportedImageMediaTypes[ct] {
			return nil, "", fmt.Errorf("unsupported image type: %s", ct)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, t.maxBytes))
		return data, ct, err
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
		return nil, "", fmt.Errorf("path traversal rejected")
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}
	if int64(len(data)) > t.maxBytes {
		return nil, "", fmt.Errorf("image exceeds maximum size of %d bytes", t.maxBytes)
	}

	// Detect media type from file bytes.
	mediaType := strings.Split(http.DetectContentType(data), ";")[0]
	if !supportedImageMediaTypes[mediaType] {
		return nil, "", fmt.Errorf("unsupported image type: %s", mediaType)
	}
	return data, mediaType, nil
}
