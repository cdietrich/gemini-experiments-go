package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileTool struct {
	WorkDir string
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file. Returns the file content as a string."
}

func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to read",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "The line number to start reading from (1-indexed)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "The maximum number of lines to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument must be a string")
	}

	fullPath := filepath.Join(t.WorkDir, path)
	fullPath = filepath.Clean(fullPath)

	cleanWorkDir := filepath.Clean(t.WorkDir)
	if !strings.HasPrefix(fullPath, cleanWorkDir+string(os.PathSeparator)) && fullPath != cleanWorkDir {
		return "", fmt.Errorf("path is outside working directory")
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	offset := 1
	limit := len(lines)

	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	if offset < 1 {
		offset = 1
	}
	if offset > len(lines) {
		return "", nil
	}

	end := offset + limit
	if end > len(lines) {
		end = len(lines)
	}

	result := strings.Join(lines[offset-1:end], "\n")
	return result, nil
}

type WriteFileTool struct {
	WorkDir string
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file. Creates the file if it doesn't exist, overwrites if it does."
}

func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument must be a string")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument must be a string")
	}

	fullPath := filepath.Join(t.WorkDir, path)
	fullPath = filepath.Clean(fullPath)

	cleanWorkDir := filepath.Clean(t.WorkDir)
	if !strings.HasPrefix(fullPath, cleanWorkDir+string(os.PathSeparator)) && fullPath != cleanWorkDir {
		return "", fmt.Errorf("path is outside working directory")
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

type EditFileTool struct {
	WorkDir string
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing a specific string with a new string. The old string must exist in the file."
}

func (t *EditFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to edit",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The exact string to replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The string to replace with",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument must be a string")
	}
	oldString, ok := args["old_string"].(string)
	if !ok {
		return "", fmt.Errorf("old_string argument must be a string")
	}
	newString, ok := args["new_string"].(string)
	if !ok {
		return "", fmt.Errorf("new_string argument must be a string")
	}

	fullPath := filepath.Join(t.WorkDir, path)
	fullPath = filepath.Clean(fullPath)

	cleanWorkDir := filepath.Clean(t.WorkDir)
	if !strings.HasPrefix(fullPath, cleanWorkDir+string(os.PathSeparator)) && fullPath != cleanWorkDir {
		return "", fmt.Errorf("path is outside working directory")
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, oldString) {
		return "", fmt.Errorf("old_string not found in file")
	}

	count := strings.Count(contentStr, oldString)
	if count > 1 {
		return "", fmt.Errorf("old_string found %d times in file, must be unique", count)
	}

	newContent := strings.Replace(contentStr, oldString, newString, 1)
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully edited %s", path), nil
}
