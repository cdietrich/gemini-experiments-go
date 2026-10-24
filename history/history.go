package history

import (
	"encoding/json"
	"os"
	"path/filepath"

	"google.golang.org/genai"
)

type History struct {
	path     string
	Messages []*genai.Content
}

func New(path string) *History {
	return &History{
		path:     path,
		Messages: make([]*genai.Content, 0),
	}
}

func (h *History) AddUserMessage(text string) {
	h.Messages = append(h.Messages, &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: text},
		},
	})
}

func (h *History) AddModelMessage(parts []*genai.Part) {
	h.Messages = append(h.Messages, &genai.Content{
		Role:  "model",
		Parts: parts,
	})
}

func (h *History) AddFunctionResponse(name, response string) {
	part := genai.NewPartFromFunctionResponse(name, map[string]any{
		"result": response,
	})
	h.Messages = append(h.Messages, &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{part},
	})
}

func (h *History) Save() error {
	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(h.Messages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(h.path, data, 0644)
}

func (h *History) Load() error {
	data, err := os.ReadFile(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &h.Messages)
}

func (h *History) Clear() {
	h.Messages = make([]*genai.Content, 0)
}

func (h *History) GetMessages() []*genai.Content {
	return h.Messages
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "gemini-agent", "history.json")
	}
	return filepath.Join(home, ".config", "gemini-agent", "history.json")
}
