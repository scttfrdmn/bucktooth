package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTempCalendar(t *testing.T) (*CalendarTool, string) {
	t.Helper()
	dir := t.TempDir()
	storePath := filepath.Join(dir, "calendar.json")
	tool, err := NewCalendarTool(storePath)
	if err != nil {
		t.Fatalf("NewCalendarTool: %v", err)
	}
	return tool, storePath
}

func TestCalendarTool_ListEmpty(t *testing.T) {
	tool, _ := newTempCalendar(t)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "list",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	events, ok := result.Data.([]CalendarEvent)
	if !ok {
		t.Fatalf("expected []CalendarEvent, got %T", result.Data)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestCalendarTool_CreateAndList(t *testing.T) {
	tool, _ := newTempCalendar(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	start := now.Format(time.RFC3339)
	end := now.Add(time.Hour).Format(time.RFC3339)

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation":   "create",
		"title":       "Team standup",
		"start":       start,
		"end":         end,
		"description": "Daily sync",
	})
	if err != nil {
		t.Fatalf("create error: %v", err)
	}
	if !result.Success {
		t.Fatalf("create failed: %s", result.Error)
	}

	created, ok := result.Data.(CalendarEvent)
	if !ok {
		t.Fatalf("expected CalendarEvent, got %T", result.Data)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if created.Title != "Team standup" {
		t.Fatalf("unexpected title: %s", created.Title)
	}

	// List should now contain one event.
	listResult, err := tool.Execute(ctx, map[string]interface{}{"operation": "list"})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	events, ok := listResult.Data.([]CalendarEvent)
	if !ok {
		t.Fatalf("expected []CalendarEvent, got %T", listResult.Data)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestCalendarTool_Delete(t *testing.T) {
	tool, _ := newTempCalendar(t)
	ctx := context.Background()

	now := time.Now().UTC()
	createResult, _ := tool.Execute(ctx, map[string]interface{}{
		"operation": "create",
		"title":     "Temp event",
		"start":     now.Format(time.RFC3339),
		"end":       now.Add(time.Hour).Format(time.RFC3339),
	})
	if !createResult.Success {
		t.Fatalf("create failed: %s", createResult.Error)
	}

	created := createResult.Data.(CalendarEvent)

	delResult, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "delete",
		"id":        created.ID,
	})
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if !delResult.Success {
		t.Fatalf("delete failed: %s", delResult.Error)
	}

	// List should be empty again.
	listResult, _ := tool.Execute(ctx, map[string]interface{}{"operation": "list"})
	events := listResult.Data.([]CalendarEvent)
	if len(events) != 0 {
		t.Fatalf("expected 0 events after delete, got %d", len(events))
	}
}

func TestCalendarTool_DeleteNotFound(t *testing.T) {
	tool, _ := newTempCalendar(t)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "delete",
		"id":        "nonexistent-id",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure for nonexistent ID")
	}
}

func TestCalendarTool_UnknownOperation(t *testing.T) {
	tool, _ := newTempCalendar(t)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "update",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure for unknown operation")
	}
}

func TestCalendarTool_JSONInputParsing(t *testing.T) {
	tool, _ := newTempCalendar(t)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"input": `{"operation":"list"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success via JSON input: %s", result.Error)
	}
}

func TestCalendarTool_ExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	path := "~/.bucktooth-test-calendar-remove-me.json"
	expanded, err := expandPath(path)
	if err != nil {
		t.Fatalf("expandPath: %v", err)
	}

	expected := filepath.Join(home, ".bucktooth-test-calendar-remove-me.json")
	if expanded != expected {
		t.Fatalf("expected %s, got %s", expected, expanded)
	}
}
