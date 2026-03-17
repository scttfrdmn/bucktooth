package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// CalendarEvent represents a single calendar event stored in the local JSON file.
type CalendarEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Description string    `json:"description"`
}

// CalendarTool manages calendar events in a local JSON file — no OAuth or external
// credentials required.
type CalendarTool struct {
	storePath string
}

// NewCalendarTool creates a CalendarTool that persists events to storePath.
// The path may contain a leading "~" which is expanded to the home directory.
// Parent directories are created if they do not already exist.
func NewCalendarTool(storePath string) (*CalendarTool, error) {
	if storePath == "" {
		storePath = "~/.bucktooth/calendar.json"
	}

	expanded, err := expandPath(storePath)
	if err != nil {
		return nil, fmt.Errorf("calendar: failed to expand path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(expanded), 0o755); err != nil {
		return nil, fmt.Errorf("calendar: failed to create store directory: %w", err)
	}

	return &CalendarTool{storePath: expanded}, nil
}

func (t *CalendarTool) Name() string { return "calendar" }

func (t *CalendarTool) Description() string {
	return `Manage calendar events. JSON params: {"operation":"list|create|delete","title":"","start":"RFC3339","end":"RFC3339","description":"","id":"<for delete>"}`
}

// Execute dispatches to list, create, or delete based on the "operation" parameter.
func (t *CalendarTool) Execute(ctx context.Context, params map[string]interface{}) (*agenkit.ToolResult, error) {
	// Support ReActAgent wrapping params in {"input": "<json string>"}
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		}
	}

	operation, _ := params["operation"].(string)

	switch operation {
	case "list":
		return t.list()
	case "create":
		return t.create(params)
	case "delete":
		return t.delete(params)
	default:
		return agenkit.NewToolError(fmt.Sprintf("unknown operation %q — must be list|create|delete", operation)), nil
	}
}

func (t *CalendarTool) list() (*agenkit.ToolResult, error) {
	events, err := t.loadEvents()
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to load events: %v", err)), nil
	}

	// Sort by start time, upcoming first.
	sort.Slice(events, func(i, j int) bool {
		return events[i].Start.Before(events[j].Start)
	})

	return agenkit.NewToolResult(events), nil
}

func (t *CalendarTool) create(params map[string]interface{}) (*agenkit.ToolResult, error) {
	title, _ := params["title"].(string)
	if title == "" {
		return agenkit.NewToolError("create requires a non-empty title"), nil
	}

	startStr, _ := params["start"].(string)
	endStr, _ := params["end"].(string)

	if startStr == "" || endStr == "" {
		return agenkit.NewToolError("create requires start and end in RFC3339 format"), nil
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("invalid start time %q: %v", startStr, err)), nil
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("invalid end time %q: %v", endStr, err)), nil
	}

	description, _ := params["description"].(string)

	event := CalendarEvent{
		ID:          uuid.New().String(),
		Title:       title,
		Start:       start,
		End:         end,
		Description: description,
	}

	events, err := t.loadEvents()
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to load events: %v", err)), nil
	}

	events = append(events, event)

	if err := t.saveEvents(events); err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to save events: %v", err)), nil
	}

	return agenkit.NewToolResult(event), nil
}

func (t *CalendarTool) delete(params map[string]interface{}) (*agenkit.ToolResult, error) {
	id, _ := params["id"].(string)
	if id == "" {
		return agenkit.NewToolError("delete requires an event id"), nil
	}

	events, err := t.loadEvents()
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to load events: %v", err)), nil
	}

	remaining := make([]CalendarEvent, 0, len(events))
	found := false
	for _, e := range events {
		if e.ID == id {
			found = true
			continue
		}
		remaining = append(remaining, e)
	}

	if !found {
		return agenkit.NewToolError(fmt.Sprintf("event %q not found", id)), nil
	}

	if err := t.saveEvents(remaining); err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to save events: %v", err)), nil
	}

	return agenkit.NewToolResult(map[string]string{
		"status":  "deleted",
		"id":      id,
	}), nil
}

// loadEvents reads events from the JSON store file.
// Returns an empty slice if the file does not exist yet.
func (t *CalendarTool) loadEvents() ([]CalendarEvent, error) {
	data, err := os.ReadFile(t.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []CalendarEvent{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", t.storePath, err)
	}

	var events []CalendarEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parse %s: %w", t.storePath, err)
	}

	return events, nil
}

// saveEvents writes events to the JSON store file.
func (t *CalendarTool) saveEvents(events []CalendarEvent) error {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal events: %w", err)
	}

	if err := os.WriteFile(t.storePath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", t.storePath, err)
	}

	return nil
}

// expandPath expands a leading "~" to the user's home directory.
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, path[1:]), nil
}
