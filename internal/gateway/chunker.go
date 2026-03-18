package gateway

import "strings"

// defaultChunkLimits maps channel type names to their character limits.
// A limit of 0 means no chunking.
var defaultChunkLimits = map[string]int{
	"discord":   2000,
	"slack":     4000,
	"teams":     1024,
	"telegram":  4096,
	"whatsapp":  4096,
	"websocket": 0,
}

// Chunker splits text into platform-appropriate chunks.
type Chunker struct {
	limits map[string]int
}

// NewChunker creates a Chunker with the given overrides merged over the defaults.
func NewChunker(overrides map[string]int) *Chunker {
	merged := make(map[string]int, len(defaultChunkLimits)+len(overrides))
	for k, v := range defaultChunkLimits {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	return &Chunker{limits: merged}
}

// Split splits text into chunks that fit within the channel's character limit.
// Returns []string{text} when the channel has no limit or text already fits.
// Splits on paragraph boundaries first (\n\n), then sentence boundaries (". "),
// then hard-cuts at the limit. Never splits mid-word.
func (c *Chunker) Split(text, channelType string) []string {
	limit, ok := c.limits[channelType]
	if !ok || limit <= 0 || len(text) <= limit {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > limit {
		chunk := remaining[:limit]

		// Try paragraph boundary (\n\n)
		if idx := strings.LastIndex(chunk, "\n\n"); idx > 0 {
			chunks = append(chunks, strings.TrimRight(remaining[:idx], " \t"))
			remaining = strings.TrimLeft(remaining[idx+2:], " \t")
			continue
		}

		// Try sentence boundary (". ")
		if idx := strings.LastIndex(chunk, ". "); idx > 0 {
			chunks = append(chunks, remaining[:idx+1])
			remaining = strings.TrimLeft(remaining[idx+2:], " \t")
			continue
		}

		// Try word boundary (space)
		if idx := strings.LastIndex(chunk, " "); idx > 0 {
			chunks = append(chunks, remaining[:idx])
			remaining = strings.TrimLeft(remaining[idx+1:], "")
			continue
		}

		// Hard cut at limit (no word boundary found)
		chunks = append(chunks, remaining[:limit])
		remaining = remaining[limit:]
	}

	if len(remaining) > 0 {
		chunks = append(chunks, remaining)
	}
	return chunks
}
