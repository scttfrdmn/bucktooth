package gateway

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/scttfrdmn/bucktooth/internal/channels"
)

const dedupDefaultTTL = 5 * time.Minute

// Deduplicator uses a ring buffer + hash index to detect duplicate messages
// within a rolling TTL window.
type Deduplicator struct {
	mu    sync.Mutex
	index map[string]time.Time // key → time added
	ring  []string             // circular eviction buffer
	head  int                  // next write position in ring
	size  int                  // capacity of ring
	ttl   time.Duration
}

// NewDeduplicator creates a Deduplicator with the given ring buffer size.
func NewDeduplicator(size int) *Deduplicator {
	if size <= 0 {
		size = 256
	}
	return &Deduplicator{
		index: make(map[string]time.Time, size),
		ring:  make([]string, size),
		size:  size,
		ttl:   dedupDefaultTTL,
	}
}

// Seen returns true if the message is a duplicate (seen within TTL).
// On a first-seen message it records the key and returns false.
func (d *Deduplicator) Seen(msg *channels.Message) bool {
	key := dedupKey(msg)

	d.mu.Lock()
	defer d.mu.Unlock()

	if t, exists := d.index[key]; exists {
		if time.Since(t) < d.ttl {
			return true
		}
		// Expired — treat as unseen; will be refreshed below.
		delete(d.index, key)
	}

	// Evict oldest ring slot if full.
	if old := d.ring[d.head]; old != "" {
		delete(d.index, old)
	}
	d.ring[d.head] = key
	d.head = (d.head + 1) % d.size
	d.index[key] = time.Now()
	return false
}

// dedupKey derives a short stable key for a message.
// Prefers msg.ID when set; falls back to a hash of channel+user+content.
func dedupKey(msg *channels.Message) string {
	if msg.ID != "" {
		return msg.ID
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", msg.ChannelID, msg.UserID, msg.Content)))
	return fmt.Sprintf("%x", h[:8])
}
