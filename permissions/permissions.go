package permissions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PermissionMode string

const (
	ModeAsk   PermissionMode = "ask"
	ModeAllow PermissionMode = "allow"
	ModeDeny  PermissionMode = "deny"
)

type Permission struct {
	Mode PermissionMode `json:"mode"`
}

type Manager struct {
	path        string
	permissions map[string]Permission
}

func NewManager(projectDir string) *Manager {
	path := filepath.Join(projectDir, ".gemini-agent", "permissions.json")
	m := &Manager{
		path:        path,
		permissions: make(map[string]Permission),
	}
	m.load()
	return m
}

func (m *Manager) load() {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &m.permissions)
}

func (m *Manager) save() {
	dir := filepath.Dir(m.path)
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(m.permissions, "", "  ")
	os.WriteFile(m.path, data, 0644)
}

func (m *Manager) Get(tool string) PermissionMode {
	if p, ok := m.permissions[tool]; ok {
		return p.Mode
	}
	return ModeAsk
}

func (m *Manager) Set(tool string, mode PermissionMode) {
	m.permissions[tool] = Permission{Mode: mode}
	m.save()
}

func (m *Manager) IsAllowed(tool string) bool {
	return m.Get(tool) == ModeAllow
}

type PromptFn func(tool string, args map[string]any) (granted, always bool)

func DefaultPrompt(tool string, args map[string]any) (bool, bool) {
	fmt.Printf("\n[Permission required for tool: %s]\n", tool)
	fmt.Printf("Args: %v\n", args)
	fmt.Print("Allow? (y=yes / n=no / a=always allow this tool): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "y":
		return true, false
	case "a", "always":
		return true, true
	case "n":
		return false, false
	default:
		return false, false
	}
}
