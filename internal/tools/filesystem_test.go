package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemTool(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystemTool(dir, 1024)
	if err != nil {
		t.Fatalf("NewFilesystemTool: %v", err)
	}
	ctx := context.Background()

	t.Run("write and read", func(t *testing.T) {
		res, err := fs.Execute(ctx, map[string]any{
			"operation": "write",
			"path":      "hello.txt",
			"content":   "Hello, World!",
		})
		if err != nil || !res.Success {
			t.Fatalf("write failed: err=%v result=%v", err, res)
		}

		res, err = fs.Execute(ctx, map[string]any{
			"operation": "read",
			"path":      "hello.txt",
		})
		if err != nil || !res.Success {
			t.Fatalf("read failed: err=%v result=%v", err, res)
		}
		if res.Data != "Hello, World!" {
			t.Errorf("got %q, want %q", res.Data, "Hello, World!")
		}
	})

	t.Run("list", func(t *testing.T) {
		res, err := fs.Execute(ctx, map[string]any{
			"operation": "list",
			"path":      "",
		})
		if err != nil || !res.Success {
			t.Fatalf("list failed: err=%v result=%v", err, res)
		}
		files, ok := res.Data.([]string)
		if !ok {
			t.Fatalf("expected []string, got %T", res.Data)
		}
		found := false
		for _, f := range files {
			if f == "hello.txt" {
				found = true
			}
		}
		if !found {
			t.Errorf("hello.txt not in list: %v", files)
		}
	})

	t.Run("delete", func(t *testing.T) {
		res, err := fs.Execute(ctx, map[string]any{
			"operation": "delete",
			"path":      "hello.txt",
		})
		if err != nil || !res.Success {
			t.Fatalf("delete failed: err=%v result=%v", err, res)
		}
		if _, statErr := os.Stat(filepath.Join(dir, "hello.txt")); !os.IsNotExist(statErr) {
			t.Error("file still exists after delete")
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		res, err := fs.Execute(ctx, map[string]any{
			"operation": "read",
			"path":      "../etc/passwd",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Success {
			t.Error("expected path traversal to fail")
		}
	})

	t.Run("absolute path rejected", func(t *testing.T) {
		res, err := fs.Execute(ctx, map[string]any{
			"operation": "read",
			"path":      "/etc/passwd",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Success {
			t.Error("expected absolute path to fail")
		}
	})
}
