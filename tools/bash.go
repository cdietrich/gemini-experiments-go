package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type BashTool struct {
	WorkDir     string
	Timeout     time.Duration
	BlockedCmds map[string]bool
	RequirePrompt bool
}

func NewBashTool(workDir string) *BashTool {
	return &BashTool{
		WorkDir: workDir,
		Timeout: 120 * time.Second,
		BlockedCmds: map[string]bool{
			"rm -rf /":      true,
			"mkfs":          true,
			"dd if=/dev/":   true,
			":(){ :|:& };:": true,
		},
		RequirePrompt: false,
	}
}

func (t *BashTool) Name() string {
	return "run_bash"
}

func (t *BashTool) Description() string {
	return "Execute a bash command in the terminal. Use this for running shell commands, git operations, npm, go, etc."
}

func (t *BashTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in milliseconds (default 120000)",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("command argument must be a string")
	}

	for blocked := range t.BlockedCmds {
		if strings.Contains(command, blocked) {
			return "", fmt.Errorf("command contains blocked pattern: %s", blocked)
		}
	}

	timeout := t.Timeout
	if ts, ok := args["timeout"].(float64); ok && ts > 0 {
		timeout = time.Duration(ts) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = t.WorkDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := ""
	if stdout.Len() > 0 {
		result += stdout.String()
	}
	if stderr.Len() > 0 {
		result += "\nStderr:\n" + stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return result + "\nCommand timed out", nil
	}

	if err != nil {
		return result + "\nExit code: " + err.Error(), nil
	}

	if result == "" {
		return "Command completed successfully (no output)", nil
	}

	return strings.TrimSpace(result), nil
}
