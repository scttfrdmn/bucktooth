package gateway

import (
	"strings"
	"testing"
)

func TestChunker_NoLimit(t *testing.T) {
	c := NewChunker(nil)
	text := strings.Repeat("word ", 100)
	chunks := c.Split(text, "websocket")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for no-limit channel, got %d", len(chunks))
	}
}

func TestChunker_FitsInLimit(t *testing.T) {
	c := NewChunker(nil)
	chunks := c.Split("hello world", "discord")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for short text, got %d", len(chunks))
	}
}

func TestChunker_SplitsParagraph(t *testing.T) {
	c := NewChunker(map[string]int{"discord": 50})
	// Two paragraphs; first fits, boundary at \n\n.
	text := "First paragraph here.\n\nSecond paragraph is also here and quite long."
	chunks := c.Split(text, "discord")
	if len(chunks) < 2 {
		t.Fatalf("expected ≥2 chunks, got %d: %v", len(chunks), chunks)
	}
	for i, ch := range chunks {
		if len(ch) > 50 {
			t.Errorf("chunk %d exceeds limit (%d chars): %q", i, len(ch), ch)
		}
	}
}

func TestChunker_SentenceBoundary(t *testing.T) {
	c := NewChunker(map[string]int{"slack": 40})
	text := "First sentence. Second sentence goes here."
	chunks := c.Split(text, "slack")
	if len(chunks) < 2 {
		t.Fatalf("expected ≥2 chunks, got %d", len(chunks))
	}
}

func TestChunker_HardCut(t *testing.T) {
	c := NewChunker(map[string]int{"teams": 10})
	text := "abcdefghijklmnopqrstuvwxyz"
	chunks := c.Split(text, "teams")
	for i, ch := range chunks {
		if len(ch) > 10 {
			t.Errorf("chunk %d exceeds hard limit: %q", i, ch)
		}
	}
	// Reassembled text must match original.
	if got := strings.Join(chunks, ""); got != text {
		t.Errorf("reassembled text mismatch: got %q want %q", got, text)
	}
}

func TestChunker_UnknownChannel(t *testing.T) {
	c := NewChunker(nil)
	long := strings.Repeat("x", 10000)
	chunks := c.Split(long, "unknown_platform")
	// No limit configured → single chunk.
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for unknown channel, got %d", len(chunks))
	}
}
