package gateway

import (
	"regexp"
	"strings"
)

// ResponseFormatter converts LLM markdown output to the dialect expected by
// each messaging platform.
type ResponseFormatter struct{}

// telegramSpecialChars lists characters that must be escaped in Telegram MarkdownV2
// outside of code spans.
var telegramSpecialChars = []string{".", "!", "(", ")", "-", "+", "=", "|", "{", "}", "#"}

// headingRe matches Markdown ATX headings (### Heading).
var headingRe = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)

// codeFenceRe matches fenced code blocks (``` ... ```).
var codeFenceRe = regexp.MustCompile("(?s)```[^`]*```")

// boldRe matches **bold** markdown.
var boldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)

// italicRe matches *italic* markdown (single asterisk only — must not be adjacent to another *).
var italicRe = regexp.MustCompile(`(?:^|[^*])\*([^*\n]+?)\*(?:[^*]|$)`)

// htmlTagRe matches simple HTML-like tags.
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// boldPlaceholder is a temporary marker used to protect bold regions from italic
// conversion (never appears in normal text).
const boldPlaceholder = "\x01"

// Format converts text to the appropriate markdown dialect for channelType.
// Discord and WebSocket receive the raw CommonMark output unchanged.
func (f ResponseFormatter) Format(text, channelType string) string {
	switch channelType {
	case "slack":
		return formatSlack(text)
	case "telegram":
		return formatTelegram(text)
	case "whatsapp":
		return formatWhatsApp(text)
	case "teams":
		return htmlTagRe.ReplaceAllString(text, "")
	default:
		return text
	}
}

// formatSlack converts CommonMark to Slack mrkdwn.
// **bold** → *bold*, *italic* → _italic_, headings → *Heading*.
// Code blocks are passed through unchanged.
func formatSlack(text string) string {
	// Protect code blocks from rewriting.
	placeholders := make(map[string]string)
	codeIdx := 0
	text = codeFenceRe.ReplaceAllStringFunc(text, func(match string) string {
		key := "\x02CODE" + string(rune('0'+codeIdx)) + "\x02"
		placeholders[key] = match
		codeIdx++
		return key
	})

	// Phase 1: mark bold regions with a placeholder to protect them from italic conversion.
	// **text** → \x01text\x01
	text = boldRe.ReplaceAllString(text, boldPlaceholder+"$1"+boldPlaceholder)

	// Phase 2: convert bare *italic* → _italic_ (only single-star, non-bold).
	text = convertItalicSlack(text)

	// Phase 3: restore bold placeholder to Slack * notation.
	text = strings.ReplaceAll(text, boldPlaceholder, "*")

	// Phase 4: headings → *Heading*
	text = headingRe.ReplaceAllString(text, "*$1*")

	// Restore code blocks.
	for k, v := range placeholders {
		text = strings.ReplaceAll(text, k, v)
	}
	return text
}

// convertItalicSlack replaces *text* with _text_ only where the * is a single star
// (not adjacent to another *, and not adjacent to boldPlaceholder which has already
// been substituted for **).
func convertItalicSlack(text string) string {
	var out strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '*' {
			// Make sure it's not a bold placeholder boundary.
			// Find closing *.
			j := i + 1
			for j < len(text) && text[j] != '*' && text[j] != '\n' {
				j++
			}
			if j < len(text) && text[j] == '*' && j > i+1 {
				// Emit _content_ instead of *content*
				out.WriteByte('_')
				out.WriteString(text[i+1 : j])
				out.WriteByte('_')
				i = j + 1
				continue
			}
		}
		out.WriteByte(text[i])
		i++
	}
	return out.String()
}

// formatTelegram escapes MarkdownV2 special characters outside code spans.
func formatTelegram(text string) string {
	var out strings.Builder
	rest := text
	for len(rest) > 0 {
		// Find next code fence (```) or inline backtick.
		fenceStart := strings.Index(rest, "```")
		inlineStart := strings.Index(rest, "`")

		if fenceStart < 0 && inlineStart < 0 {
			break
		}

		var codeStart, codeEnd int
		if fenceStart >= 0 && (inlineStart < 0 || fenceStart <= inlineStart) {
			codeStart = fenceStart
			endIdx := strings.Index(rest[fenceStart+3:], "```")
			if endIdx < 0 {
				break
			}
			codeEnd = fenceStart + 3 + endIdx + 3
		} else {
			codeStart = inlineStart
			endIdx := strings.Index(rest[inlineStart+1:], "`")
			if endIdx < 0 {
				break
			}
			codeEnd = inlineStart + 1 + endIdx + 1
		}

		out.WriteString(telegramEscape(rest[:codeStart]))
		out.WriteString(rest[codeStart:codeEnd])
		rest = rest[codeEnd:]
	}
	out.WriteString(telegramEscape(rest))
	return out.String()
}

func telegramEscape(s string) string {
	for _, ch := range telegramSpecialChars {
		s = strings.ReplaceAll(s, ch, `\`+ch)
	}
	return s
}

// formatWhatsApp converts to WhatsApp markdown:
// **bold** → *bold*, *italic* → _italic_, strip code fences and heading markers.
func formatWhatsApp(text string) string {
	// Strip fenced code blocks (keep content, remove fences).
	text = codeFenceRe.ReplaceAllStringFunc(text, func(match string) string {
		inner := match[3 : len(match)-3]
		if nl := strings.Index(inner, "\n"); nl >= 0 {
			inner = inner[nl+1:]
		}
		return strings.TrimSpace(inner)
	})

	// Strip headings (keep text, remove # markers).
	text = headingRe.ReplaceAllString(text, "$1")

	// Phase 1: mark bold with placeholder so italic pass doesn't touch it.
	text = boldRe.ReplaceAllString(text, boldPlaceholder+"$1"+boldPlaceholder)

	// Phase 2: convert bare *italic* → _italic_.
	text = convertItalicSlack(text) // same single-star logic

	// Phase 3: restore bold placeholder to WhatsApp * notation.
	text = strings.ReplaceAll(text, boldPlaceholder, "*")

	return text
}
