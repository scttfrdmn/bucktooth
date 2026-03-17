package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

const defaultMaxFileSize = 10 * 1024 * 1024 // 10MB

// FilesystemTool provides sandboxed file operations.
type FilesystemTool struct {
	sandboxDir  string
	maxFileSize int64
}

// NewFilesystemTool creates a new sandboxed filesystem tool.
func NewFilesystemTool(sandboxDir string, maxFileSize int64) (*FilesystemTool, error) {
	if sandboxDir == "" {
		sandboxDir = filepath.Join(os.TempDir(), "bucktooth-sandbox")
	}

	// Expand ~ in path
	if strings.HasPrefix(sandboxDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		sandboxDir = filepath.Join(home, sandboxDir[2:])
	}

	// Ensure sandbox directory exists
	if err := os.MkdirAll(sandboxDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sandbox directory: %w", err)
	}

	if maxFileSize <= 0 {
		maxFileSize = defaultMaxFileSize
	}

	return &FilesystemTool{
		sandboxDir:  sandboxDir,
		maxFileSize: maxFileSize,
	}, nil
}

func (t *FilesystemTool) Name() string { return "filesystem" }

func (t *FilesystemTool) Description() string {
	return `Performs sandboxed file operations. Pass parameters as JSON: {"operation":"read|write|list|delete","path":"<relative-path>","content":"<content for write>"}`
}

func (t *FilesystemTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		}
	}

	operation, ok := params["operation"].(string)
	if !ok || operation == "" {
		return agenkit.NewToolError("missing required parameter: operation"), nil
	}

	path, _ := params["path"].(string)

	switch operation {
	case "read":
		return t.readFile(path)
	case "write":
		content, _ := params["content"].(string)
		return t.writeFile(path, content)
	case "list":
		return t.listFiles(path)
	case "delete":
		return t.deleteFile(path)
	default:
		return agenkit.NewToolError(fmt.Sprintf("unknown operation: %s (must be read|write|list|delete)", operation)), nil
	}
}

// safePath resolves a path within the sandbox, rejecting any path traversal attempts.
func (t *FilesystemTool) safePath(relPath string) (string, error) {
	if relPath == "" {
		return t.sandboxDir, nil
	}

	// Reject absolute paths
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	// Reject any ".." components
	clean := filepath.Clean(relPath)
	if strings.Contains(clean, "..") {
		return "", fmt.Errorf("path traversal is not allowed")
	}

	fullPath := filepath.Join(t.sandboxDir, clean)

	// Double-check the resolved path is within the sandbox
	if !strings.HasPrefix(fullPath, t.sandboxDir+string(os.PathSeparator)) && fullPath != t.sandboxDir {
		return "", fmt.Errorf("path is outside sandbox")
	}

	return fullPath, nil
}

func (t *FilesystemTool) readFile(path string) (*agenkit.ToolResult, error) {
	fullPath, err := t.safePath(path)
	if err != nil {
		return agenkit.NewToolError(err.Error()), nil
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("file not found: %s", path)), nil
	}

	if info.Size() > t.maxFileSize {
		return agenkit.NewToolError(fmt.Sprintf("file exceeds maximum size of %d bytes", t.maxFileSize)), nil
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	return agenkit.NewToolResult(string(data)), nil
}

func (t *FilesystemTool) writeFile(path string, content string) (*agenkit.ToolResult, error) {
	if path == "" {
		return agenkit.NewToolError("missing required parameter: path"), nil
	}

	if int64(len(content)) > t.maxFileSize {
		return agenkit.NewToolError(fmt.Sprintf("content exceeds maximum size of %d bytes", t.maxFileSize)), nil
	}

	fullPath, err := t.safePath(path)
	if err != nil {
		return agenkit.NewToolError(err.Error()), nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to write file: %v", err)), nil
	}

	return agenkit.NewToolResult(fmt.Sprintf("written %d bytes to %s", len(content), path)), nil
}

func (t *FilesystemTool) listFiles(path string) (*agenkit.ToolResult, error) {
	fullPath, err := t.safePath(path)
	if err != nil {
		return agenkit.NewToolError(err.Error()), nil
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to list directory: %v", err)), nil
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		files = append(files, name)
	}

	return agenkit.NewToolResult(files), nil
}

func (t *FilesystemTool) deleteFile(path string) (*agenkit.ToolResult, error) {
	if path == "" {
		return agenkit.NewToolError("missing required parameter: path"), nil
	}

	fullPath, err := t.safePath(path)
	if err != nil {
		return agenkit.NewToolError(err.Error()), nil
	}

	if err := os.Remove(fullPath); err != nil {
		return agenkit.NewToolError(fmt.Sprintf("failed to delete file: %v", err)), nil
	}

	return agenkit.NewToolResult(fmt.Sprintf("deleted %s", path)), nil
}
