package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// MessageFormatterTool formats text for different messaging platforms.
type MessageFormatterTool struct{}

// NewMessageFormatterTool creates a new message formatter tool.
func NewMessageFormatterTool() *MessageFormatterTool {
	return &MessageFormatterTool{}
}

func (t *MessageFormatterTool) Name() string { return "message_formatter" }

func (t *MessageFormatterTool) Description() string {
	return `Formats text for messaging platforms. Pass parameters as JSON: {"text":"<content>","format":"discord|plain|markdown"}`
}

func (t *MessageFormatterTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		}
	}

	text, ok := params["text"].(string)
	if !ok || text == "" {
		return agenkit.NewToolError("missing required parameter: text"), nil
	}

	format, _ := params["format"].(string)
	if format == "" {
		format = "plain"
	}

	var result string
	switch format {
	case "discord":
		result = toDiscordFormat(text)
	case "plain":
		result = stripMarkup(text)
	case "markdown":
		result = normalizeMarkdown(text)
	default:
		return agenkit.NewToolError(fmt.Sprintf("unknown format: %s (must be discord|plain|markdown)", format)), nil
	}

	return agenkit.NewToolResult(result), nil
}

// toDiscordFormat converts standard markdown to Discord-compatible formatting.
func toDiscordFormat(text string) string {
	// **bold** → **bold** (already Discord-compatible)
	// *italic* or _italic_ → *italic*
	// `code` → `code`
	// ```code block``` → ```code block```
	// These are already compatible; just normalize some edge cases.

	// Normalize heading markers to bold
	headingRe := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	text = headingRe.ReplaceAllString(text, "**$1**")

	// Normalize _italic_ to *italic* (Discord uses * not _)
	underscoreItalicRe := regexp.MustCompile(`(?:^|[^_])_([^_\n]+)_(?:[^_]|$)`)
	text = underscoreItalicRe.ReplaceAllStringFunc(text, func(s string) string {
		inner := underscoreItalicRe.FindStringSubmatch(s)
		if len(inner) > 1 {
			return strings.Replace(s, "_"+inner[1]+"_", "*"+inner[1]+"*", 1)
		}
		return s
	})

	return text
}

// stripMarkup removes all markdown/formatting markup.
func stripMarkup(text string) string {
	// Remove code blocks
	codeBlockRe := regexp.MustCompile("```[\\s\\S]*?```")
	text = codeBlockRe.ReplaceAllString(text, "")

	// Remove inline code
	inlineCodeRe := regexp.MustCompile("`[^`]+`")
	text = inlineCodeRe.ReplaceAllString(text, "")

	// Remove bold/italic markers
	text = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile(`_{1,3}([^_]+)_{1,3}`).ReplaceAllString(text, "$1")

	// Remove headings
	text = regexp.MustCompile(`(?m)^#{1,6}\s+`).ReplaceAllString(text, "")

	// Remove links [text](url) → text
	text = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(text, "$1")

	// Clean up extra whitespace
	text = strings.TrimSpace(text)
	return text
}

// normalizeMarkdown validates and normalizes markdown.
func normalizeMarkdown(text string) string {
	// Ensure proper spacing after headers
	text = regexp.MustCompile(`(?m)^(#{1,6})([^ #])`).ReplaceAllString(text, "$1 $2")

	// Ensure blank line before headers (except at start)
	text = regexp.MustCompile(`(?m)([^\n])\n(#{1,6} )`).ReplaceAllString(text, "$1\n\n$2")

	return strings.TrimSpace(text)
}
