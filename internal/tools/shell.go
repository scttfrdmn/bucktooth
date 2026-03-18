package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

const defaultShellTimeout = 30 * time.Second

// ShellTool executes shell commands with an optional human-approval gate.
//
// When require_approval is true (default), the tool implements a two-step flow:
// 1. First call: returns an approval prompt describing the command.
// 2. Second call with approved:true: executes the command and returns output.
//
// This lets the LLM relay the approval request to the user before executing,
// ensuring human oversight for potentially destructive operations.
type ShellTool struct {
	requireApproval bool
	allowedCmds     []string // optional allowlist of command prefixes; empty = all allowed
}

// NewShellTool creates a ShellTool.
// requireApproval gates execution behind a two-step confirmation (default true).
// allowedCmds, if non-empty, restricts execution to commands whose base name
// (first whitespace-delimited token) matches one of the listed prefixes.
func NewShellTool(requireApproval bool, allowedCmds []string) *ShellTool {
	return &ShellTool{
		requireApproval: requireApproval,
		allowedCmds:     allowedCmds,
	}
}

func (t *ShellTool) Name() string { return "shell" }

func (t *ShellTool) Description() string {
	return `Execute a shell command and return its output. ` +
		`Parameters: {"command":"<shell command>","working_dir":"<optional path>",` +
		`"timeout_seconds":<optional int>,"approved":<optional bool>}. ` +
		`When human approval is required, first call returns an approval prompt; ` +
		`re-call with approved:true to execute.`
}

// Execute runs the shell command or returns an approval prompt.
func (t *ShellTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
	// Unwrap ReActAgent {"input": "<json string>"} wrapper.
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		} else {
			params = map[string]any{"command": raw}
		}
	}

	command, _ := params["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return agenkit.NewToolError("missing required parameter: command"), nil
	}

	// Allowlist check.
	if err := t.checkAllowlist(command); err != nil {
		return agenkit.NewToolError(err.Error()), nil
	}

	// Approval gate.
	approved := false
	if v, ok := params["approved"].(bool); ok {
		approved = v
	}
	if t.requireApproval && !approved {
		prompt := fmt.Sprintf(
			"APPROVAL REQUIRED\n\nThe following command is pending human approval:\n\n  %s\n\n"+
				"To execute, call this tool again with approved:true. To cancel, do not proceed.",
			command,
		)
		return agenkit.NewToolResult(prompt), nil
	}

	// Build execution context with timeout.
	timeout := defaultShellTimeout
	if v, ok := params["timeout_seconds"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	workDir, _ := params["working_dir"].(string)

	cmd := exec.CommandContext(execCtx, "sh", "-c", command) //nolint:gosec
	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var sb strings.Builder
	if stdout.Len() > 0 {
		sb.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[stderr]\n")
		sb.WriteString(stderr.String())
	}
	if err != nil {
		exitMsg := fmt.Sprintf("\n[exit error: %v]", err)
		sb.WriteString(exitMsg)
	}

	output := strings.TrimRight(sb.String(), "\n")
	if output == "" {
		output = "(no output)"
	}

	return agenkit.NewToolResult(output), nil
}

// checkAllowlist validates the command against the configured allowlist.
// If the allowlist is empty, all commands are permitted.
func (t *ShellTool) checkAllowlist(command string) error {
	if len(t.allowedCmds) == 0 {
		return nil
	}
	// Extract the base command name (first token, strip any path prefix).
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}
	base := parts[0]
	// Strip path prefix (e.g. /usr/bin/ls → ls).
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	for _, allowed := range t.allowedCmds {
		if base == allowed || strings.HasPrefix(base, allowed) {
			return nil
		}
	}
	return fmt.Errorf("command %q is not in the allowed commands list", base)
}
