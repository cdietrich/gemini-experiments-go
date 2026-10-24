package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GlobTool struct {
	WorkDir string
}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return "Find files matching a glob pattern. Returns a list of matching file paths."
}

func (t *GlobTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The glob pattern to match (e.g., '**/*.go', 'src/*.ts')",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern argument must be a string")
	}

	var matches []string
	err := filepath.WalkDir(t.WorkDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(t.WorkDir, path)
		if err != nil {
			return nil
		}

		matched, err := filepath.Match(pattern, filepath.Base(relPath))
		if err != nil {
			return nil
		}

		if strings.Contains(pattern, "**") {
			globPattern := strings.ReplaceAll(pattern, "**", "*")
			matched, _ = filepath.Match(globPattern, relPath)
			if !matched {
				matched, _ = filepath.Match(filepath.Join("**", globPattern), relPath)
			}
		}

		if matched {
			matches = append(matches, relPath)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "No files found matching pattern", nil
	}

	return strings.Join(matches, "\n"), nil
}

type GrepTool struct {
	WorkDir string
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "Search for a regex pattern in files. Returns matching file paths and line numbers."
}

func (t *GrepTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The regex pattern to search for",
			},
			"include": map[string]any{
				"type":        "string",
				"description": "File pattern to include (e.g., '*.go')",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern argument must be a string")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	include := "*.go"
	if inc, ok := args["include"].(string); ok && inc != "" {
		include = inc
	}

	var results []string

	err = filepath.WalkDir(t.WorkDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		matched, _ := filepath.Match(include, filepath.Base(path))
		if !matched {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		relPath, _ := filepath.Rel(t.WorkDir, path)

		for i, line := range lines {
			if re.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d: %s", relPath, i+1, strings.TrimSpace(line)))
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "No matches found", nil
	}

	if len(results) > 100 {
		results = results[:100]
		results = append(results, "... (truncated, more than 100 matches)")
	}

	return strings.Join(results, "\n"), nil
}

type ListDirTool struct {
	WorkDir string
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List the contents of a directory. Returns file and directory names."
}

func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The directory path to list (default: current directory)",
			},
		},
		"required": []string{},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	fullPath := filepath.Join(t.WorkDir, path)
	fullPath = filepath.Clean(fullPath)

	cleanWorkDir := filepath.Clean(t.WorkDir)
	if !strings.HasPrefix(fullPath, cleanWorkDir+string(os.PathSeparator)) && fullPath != cleanWorkDir {
		return "", fmt.Errorf("path is outside working directory")
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return "", err
	}

	var results []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		results = append(results, name)
	}

	if len(results) == 0 {
		return "Empty directory", nil
	}

	return strings.Join(results, "\n"), nil
}
