package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// BrowserTool provides headless Chrome browser automation via the Chrome DevTools Protocol.
// Each Execute call creates and tears down its own chromedp context to avoid leaked browser
// processes.
type BrowserTool struct {
	timeout time.Duration
}

// NewBrowserTool creates a BrowserTool with the given per-action timeout in seconds.
// If timeoutSeconds is ≤ 0, a default of 30 seconds is used.
func NewBrowserTool(timeoutSeconds int) *BrowserTool {
	t := time.Duration(timeoutSeconds) * time.Second
	if t <= 0 {
		t = 30 * time.Second
	}
	return &BrowserTool{timeout: t}
}

func (b *BrowserTool) Name() string { return "browser" }

func (b *BrowserTool) Description() string {
	return `Control a headless Chrome browser. ` +
		`Actions: navigate (url), click (selector), type (selector, text), screenshot (), extract (selector). ` +
		`Parameters: {"action":"navigate","url":"<url>"} | {"action":"click","selector":"<css>"} | ` +
		`{"action":"type","selector":"<css>","text":"<text>"} | {"action":"screenshot"} | {"action":"extract","selector":"<css>"}`
}

// Execute dispatches on input["action"] and runs the corresponding browser operation.
// A fresh chromedp allocator and context are created and torn down for each call.
func (b *BrowserTool) Execute(ctx context.Context, input map[string]any) (*agenkit.ToolResult, error) {
	action, _ := input["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("browser: missing required parameter: action")
	}

	// Each call gets its own allocator + context to prevent leaked Chrome processes.
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(taskCtx, b.timeout)
	defer timeoutCancel()

	switch action {
	case "navigate":
		url, _ := input["url"].(string)
		if url == "" {
			return nil, fmt.Errorf("browser: navigate requires url")
		}
		if err := chromedp.Run(timeoutCtx, chromedp.Navigate(url)); err != nil {
			return nil, fmt.Errorf("browser: navigate failed: %w", err)
		}
		return &agenkit.ToolResult{Success: true, Data: "navigated to " + url}, nil

	case "click":
		selector, _ := input["selector"].(string)
		if selector == "" {
			return nil, fmt.Errorf("browser: click requires selector")
		}
		if err := chromedp.Run(timeoutCtx, chromedp.Click(selector, chromedp.ByQuery)); err != nil {
			return nil, fmt.Errorf("browser: click failed: %w", err)
		}
		return &agenkit.ToolResult{Success: true, Data: "clicked " + selector}, nil

	case "type":
		selector, _ := input["selector"].(string)
		if selector == "" {
			return nil, fmt.Errorf("browser: type requires selector")
		}
		text, _ := input["text"].(string)
		if err := chromedp.Run(timeoutCtx, chromedp.SendKeys(selector, text, chromedp.ByQuery)); err != nil {
			return nil, fmt.Errorf("browser: type failed: %w", err)
		}
		return &agenkit.ToolResult{Success: true, Data: "typed into " + selector}, nil

	case "screenshot":
		var buf []byte
		if err := chromedp.Run(timeoutCtx, chromedp.CaptureScreenshot(&buf)); err != nil {
			return nil, fmt.Errorf("browser: screenshot failed: %w", err)
		}
		return &agenkit.ToolResult{Success: true, Data: base64.StdEncoding.EncodeToString(buf)}, nil

	case "extract":
		selector, _ := input["selector"].(string)
		if selector == "" {
			return nil, fmt.Errorf("browser: extract requires selector")
		}
		var out string
		if err := chromedp.Run(timeoutCtx, chromedp.Text(selector, &out, chromedp.ByQuery)); err != nil {
			return nil, fmt.Errorf("browser: extract failed: %w", err)
		}
		return &agenkit.ToolResult{Success: true, Data: out}, nil

	default:
		return nil, fmt.Errorf("browser: unknown action %q (want navigate, click, type, screenshot, extract)", action)
	}
}
