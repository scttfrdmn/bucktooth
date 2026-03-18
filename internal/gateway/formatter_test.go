package gateway

import (
	"strings"
	"testing"
)

func TestFormatter_Passthrough(t *testing.T) {
	f := ResponseFormatter{}
	text := "**Hello** world"
	for _, ch := range []string{"discord", "websocket", "unknown"} {
		if got := f.Format(text, ch); got != text {
			t.Errorf("channel %q: expected passthrough, got %q", ch, got)
		}
	}
}

func TestFormatter_Slack_Bold(t *testing.T) {
	f := ResponseFormatter{}
	got := f.Format("**bold text**", "slack")
	if !strings.Contains(got, "*bold text*") {
		t.Errorf("slack bold conversion failed: %q", got)
	}
	if strings.Contains(got, "**") {
		t.Errorf("slack output should not contain **: %q", got)
	}
}

func TestFormatter_Slack_Italic(t *testing.T) {
	f := ResponseFormatter{}
	// Use a non-bold italic (single *).
	got := f.Format("*italic*", "slack")
	if !strings.Contains(got, "_italic_") {
		t.Errorf("slack italic conversion failed: %q", got)
	}
}

func TestFormatter_Slack_Heading(t *testing.T) {
	f := ResponseFormatter{}
	got := f.Format("### Section Title", "slack")
	if !strings.Contains(got, "*Section Title*") {
		t.Errorf("slack heading conversion failed: %q", got)
	}
}

func TestFormatter_Slack_CodeBlockPreserved(t *testing.T) {
	f := ResponseFormatter{}
	code := "```go\nfmt.Println(\"**not bold**\")\n```"
	got := f.Format(code, "slack")
	// The content inside the code block should not have been rewritten.
	if !strings.Contains(got, "**not bold**") {
		t.Errorf("slack formatter should not rewrite code block contents: %q", got)
	}
}

func TestFormatter_WhatsApp_Bold(t *testing.T) {
	f := ResponseFormatter{}
	got := f.Format("**hello**", "whatsapp")
	if !strings.Contains(got, "*hello*") {
		t.Errorf("whatsapp bold failed: %q", got)
	}
}

func TestFormatter_WhatsApp_StripHeading(t *testing.T) {
	f := ResponseFormatter{}
	got := f.Format("## My Title\nsome text", "whatsapp")
	if strings.Contains(got, "##") {
		t.Errorf("whatsapp should strip heading markers: %q", got)
	}
	if !strings.Contains(got, "My Title") {
		t.Errorf("whatsapp should keep heading text: %q", got)
	}
}

func TestFormatter_Telegram_EscapesSpecials(t *testing.T) {
	f := ResponseFormatter{}
	got := f.Format("hello.world!", "telegram")
	if !strings.Contains(got, `\.`) {
		t.Errorf("telegram should escape dots: %q", got)
	}
	if !strings.Contains(got, `\!`) {
		t.Errorf("telegram should escape exclamation: %q", got)
	}
}

func TestFormatter_Telegram_CodeBlockNotEscaped(t *testing.T) {
	f := ResponseFormatter{}
	got := f.Format("```\nhello.world\n```", "telegram")
	// Inside code block, dots should not be escaped.
	if strings.Contains(got, `\.`) {
		t.Errorf("telegram should not escape inside code blocks: %q", got)
	}
}

func TestFormatter_Teams_StripHTML(t *testing.T) {
	f := ResponseFormatter{}
	got := f.Format("hello <br> world <p>para</p>", "teams")
	if strings.Contains(got, "<") {
		t.Errorf("teams formatter should strip HTML tags: %q", got)
	}
}
