package gateway

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/scttfrdmn/bucktooth/internal/channels"
)

const recentMessageCap = 50

// RecentMessage is a compact record stored in the ring buffer.
type RecentMessage struct {
	Timestamp time.Time `json:"timestamp"`
	ChannelID string    `json:"channel_id"`
	UserID    string    `json:"user_id"`
	Preview   string    `json:"content_preview"` // first 120 runes
	Direction string    `json:"direction"`        // "in" or "out"
}

// Stats tracks message counters and a fixed-size recent-message ring buffer.
type Stats struct {
	messagesIn  atomic.Uint64
	messagesOut atomic.Uint64
	startTime   time.Time
	recentMu    sync.Mutex
	recent      []RecentMessage
}

// NewStats creates a Stats instance with the clock started.
func NewStats() *Stats {
	return &Stats{startTime: time.Now()}
}

// RecordInbound increments the inbound counter and adds an entry to the ring buffer.
func (s *Stats) RecordInbound(msg *channels.Message) {
	s.messagesIn.Add(1)
	s.addRecent(RecentMessage{
		Timestamp: time.Now(),
		ChannelID: msg.ChannelID,
		UserID:    msg.UserID,
		Preview:   truncatePreview(msg.Content),
		Direction: "in",
	})
}

// RecordOutbound increments the outbound counter and adds an entry to the ring buffer.
func (s *Stats) RecordOutbound(msg *channels.Message) {
	s.messagesOut.Add(1)
	s.addRecent(RecentMessage{
		Timestamp: time.Now(),
		ChannelID: msg.ChannelID,
		UserID:    msg.UserID,
		Preview:   truncatePreview(msg.Content),
		Direction: "out",
	})
}

func (s *Stats) addRecent(r RecentMessage) {
	s.recentMu.Lock()
	defer s.recentMu.Unlock()
	s.recent = append([]RecentMessage{r}, s.recent...) // newest first
	if len(s.recent) > recentMessageCap {
		s.recent = s.recent[:recentMessageCap]
	}
}

// StatsSnapshot is a point-in-time copy of all stats.
type StatsSnapshot struct {
	MessagesIn    uint64          `json:"messages_in"`
	MessagesOut   uint64          `json:"messages_out"`
	UptimeSeconds uint64          `json:"uptime_seconds"`
	Recent        []RecentMessage `json:"recent_messages"`
}

// Snapshot returns a consistent point-in-time copy of all stats.
func (s *Stats) Snapshot() StatsSnapshot {
	s.recentMu.Lock()
	recent := make([]RecentMessage, len(s.recent))
	copy(recent, s.recent)
	s.recentMu.Unlock()

	return StatsSnapshot{
		MessagesIn:    s.messagesIn.Load(),
		MessagesOut:   s.messagesOut.Load(),
		UptimeSeconds: uint64(time.Since(s.startTime).Seconds()),
		Recent:        recent,
	}
}

func truncatePreview(s string) string {
	runes := []rune(s)
	if len(runes) > 120 {
		return string(runes[:120])
	}
	return s
}
