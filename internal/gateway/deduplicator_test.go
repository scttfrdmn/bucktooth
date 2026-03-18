package gateway

import (
	"testing"
	"time"

	"github.com/scttfrdmn/bucktooth/internal/channels"
)

func makeMsg(id, channelID, userID, content string) *channels.Message {
	return &channels.Message{
		ID:        id,
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
	}
}

func TestDeduplicator_FirstSeenReturnsFalse(t *testing.T) {
	d := NewDeduplicator(16)
	msg := makeMsg("msg1", "ch", "u1", "hello")
	if d.Seen(msg) {
		t.Fatal("expected false for first-seen message")
	}
}

func TestDeduplicator_DuplicateReturnsTrue(t *testing.T) {
	d := NewDeduplicator(16)
	msg := makeMsg("msg1", "ch", "u1", "hello")
	d.Seen(msg)
	if !d.Seen(msg) {
		t.Fatal("expected true for duplicate message")
	}
}

func TestDeduplicator_DifferentIDsNotDuplicate(t *testing.T) {
	d := NewDeduplicator(16)
	d.Seen(makeMsg("msg1", "ch", "u1", "hello"))
	if d.Seen(makeMsg("msg2", "ch", "u1", "hello")) {
		t.Fatal("different IDs should not be considered duplicates")
	}
}

func TestDeduplicator_ContentKeyFallback(t *testing.T) {
	// No ID set — falls back to content hash.
	d := NewDeduplicator(16)
	msg := makeMsg("", "ch", "u1", "hello world")
	d.Seen(msg)
	if !d.Seen(makeMsg("", "ch", "u1", "hello world")) {
		t.Fatal("same content+channel+user should be detected as duplicate")
	}
}

func TestDeduplicator_TTLExpiry(t *testing.T) {
	d := NewDeduplicator(16)
	d.ttl = 10 * time.Millisecond
	msg := makeMsg("msg1", "ch", "u1", "hello")
	d.Seen(msg)
	time.Sleep(20 * time.Millisecond)
	if d.Seen(msg) {
		t.Fatal("expected false after TTL expiry")
	}
}

func TestDeduplicator_RingEviction(t *testing.T) {
	d := NewDeduplicator(2) // tiny buffer
	d.Seen(makeMsg("a", "ch", "u", "a"))
	d.Seen(makeMsg("b", "ch", "u", "b"))
	// "a" should have been evicted by now (ring is full after "c").
	d.Seen(makeMsg("c", "ch", "u", "c"))
	// "a" was the oldest; it's gone from index, so should be not-seen.
	if d.Seen(makeMsg("a", "ch", "u", "a")) {
		t.Fatal("evicted entry should not be considered duplicate")
	}
}
