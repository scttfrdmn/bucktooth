package gateway

import "sync"

// UserPrefs stores per-user preferred response channel IDs.
type UserPrefs struct {
	mu    sync.RWMutex
	prefs map[string]string // user_id → preferred channel_id
}

// NewUserPrefs creates an empty UserPrefs store.
func NewUserPrefs() *UserPrefs {
	return &UserPrefs{prefs: make(map[string]string)}
}

// Set sets the preferred channel for a user.
func (p *UserPrefs) Set(userID, channelID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prefs[userID] = channelID
}

// Get returns the preferred channel for a user, or "" if none is set.
func (p *UserPrefs) Get(userID string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.prefs[userID]
}

// Delete removes the user's preference.
func (p *UserPrefs) Delete(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.prefs, userID)
}
