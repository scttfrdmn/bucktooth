package gateway

import "sync"

// UserPrefs stores per-user preferred response channel IDs and system prompt overrides.
type UserPrefs struct {
	mu            sync.RWMutex
	prefs         map[string]string // user_id → preferred channel_id
	systemPrompts map[string]string // user_id → system prompt override
}

// NewUserPrefs creates an empty UserPrefs store.
func NewUserPrefs() *UserPrefs {
	return &UserPrefs{
		prefs:         make(map[string]string),
		systemPrompts: make(map[string]string),
	}
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

// Delete removes the user's channel preference.
func (p *UserPrefs) Delete(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.prefs, userID)
}

// SetSystemPrompt stores a custom system prompt for the user.
func (p *UserPrefs) SetSystemPrompt(userID, prompt string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.systemPrompts[userID] = prompt
}

// GetSystemPrompt returns the user's custom system prompt, or "" if none is set.
func (p *UserPrefs) GetSystemPrompt(userID string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.systemPrompts[userID]
}

// DeleteSystemPrompt clears the user's custom system prompt.
func (p *UserPrefs) DeleteSystemPrompt(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.systemPrompts, userID)
}
