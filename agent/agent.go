package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"gemini-agent/permissions"
	"gemini-agent/tools"

	"google.golang.org/genai"
)

type Agent struct {
	client    *genai.Client
	model     string
	registry  *tools.Registry
	history   HistoryManager
	perms     *permissions.Manager
	maxTurns  int
	systemMsg string
}

type HistoryManager interface {
	AddUserMessage(text string)
	AddModelMessage(parts []*genai.Part)
	AddFunctionResponse(name, response string)
	GetMessages() []*genai.Content
	Save() error
	Load() error
	Clear()
}

type Option func(*Agent)

func WithModel(model string) Option {
	return func(a *Agent) {
		a.model = model
	}
}

func WithMaxTurns(max int) Option {
	return func(a *Agent) {
		a.maxTurns = max
	}
}

func WithSystemMessage(msg string) Option {
	return func(a *Agent) {
		a.systemMsg = msg
	}
}

func WithHistory(h HistoryManager) Option {
	return func(a *Agent) {
		a.history = h
	}
}

func WithPermissions(perms *permissions.Manager) Option {
	return func(a *Agent) {
		a.perms = perms
	}
}

func New(client *genai.Client, registry *tools.Registry, opts ...Option) *Agent {
	a := &Agent{
		client:   client,
		model:    "gemini-3.6-flash",
		registry: registry,
		maxTurns: 20,
		systemMsg: `You are a helpful coding assistant with access to tools for file operations, shell commands, and code search.

When asked to perform tasks:
1. Break down complex tasks into steps
2. Use tools to read, write, and edit files
3. Run shell commands when needed (build, test, etc.)
4. Search the codebase to understand context
5. Explain your actions clearly

Always verify your work by reading files after editing or checking command outputs.`,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	a.history.AddUserMessage(input)

	turnCount := 0

	for turnCount < a.maxTurns {
		turnCount++

		resp, err := a.generateWithRetry(ctx)
		if err != nil {
			return "", err
		}

		if resp == nil {
			return "", fmt.Errorf("nil response from model")
		}

		funcCalls := resp.FunctionCalls()
		if len(funcCalls) == 0 {
			text := resp.Text()
			if text == "" {
				text = "No response from agent."
			}
			a.history.AddModelMessage([]*genai.Part{{Text: text}})
			_ = a.history.Save()
			return text, nil
		}

		if len(resp.Candidates) == 0 {
			return "", fmt.Errorf("no candidates in response")
		}
		candidate := resp.Candidates[0]
		if candidate.Content != nil {
			a.history.AddModelMessage(candidate.Content.Parts)
		}

		for _, fc := range funcCalls {
			args := tools.ParseFunctionCallArgs(fc)
			fmt.Printf("[Tool: %s]\n", fc.Name)
			fmt.Printf("  Args: %v\n", args)

			allowed := a.checkPermission(fc.Name, args)
			if !allowed {
				a.history.AddFunctionResponse(fc.Name, "Error: Permission denied by user")
				continue
			}

			result, err := a.registry.Execute(ctx, fc.Name, args)
			if err != nil {
				a.history.AddFunctionResponse(fc.Name, fmt.Sprintf("Error: %s", err))
			} else {
				a.history.AddFunctionResponse(fc.Name, result)
			}
		}
	}

	_ = a.history.Save()
	return "Maximum iterations reached.", nil
}

func (a *Agent) checkPermission(tool string, args map[string]any) bool {
	if a.perms == nil {
		return true
	}

	if a.perms.IsAllowed(tool) {
		fmt.Printf("  [Auto-approved (previously allowed)]\n")
		return true
	}

	granted, always := permissions.DefaultPrompt(tool, args)
	if granted && always {
		a.perms.Set(tool, permissions.ModeAllow)
		fmt.Printf("  [Permission saved: %s is now always allowed]\n", tool)
	}
	return granted
}

func (a *Agent) generateWithRetry(ctx context.Context) (*genai.GenerateContentResponse, error) {
	delays := []time.Duration{
		10 * time.Second,
		20 * time.Second,
		30 * time.Second,
		60 * time.Second,
		2 * time.Minute,
	}

	for attempt := 0; attempt < len(delays)+1; attempt++ {
		resp, err := a.generate(ctx)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "429") || strings.Contains(errStr, "RESOURCE_EXHAUSTED") || strings.Contains(errStr, "quota") {
				// Check for daily limit
				if strings.Contains(errStr, "GenerateRequestsPerDayPerProjectPerModel-FreeTier") {
					fmt.Println("\n[!] Free tier daily limit reached for this model.")
					if newModel := a.promptModelSwitch(); newModel != "" {
						a.model = newModel
						fmt.Printf("[Switched to %s, retrying...]\n", newModel)
						continue
					}
					return nil, fmt.Errorf("daily quota exceeded for %s", a.model)
				}

				// Use server-suggested retry delay if available
				if retryDelay := parseRetryDelay(errStr); retryDelay > 0 {
					fmt.Printf("[Rate limited, waiting %v as suggested by server...]\n", retryDelay)
					select {
					case <-time.After(retryDelay):
						continue
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}

				if attempt < len(delays) {
					delay := delays[attempt]
					fmt.Printf("[Rate limited, waiting %v before retry (%d/%d)...]\n", delay, attempt+1, len(delays)+1)
					select {
					case <-time.After(delay):
						continue
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}
			return nil, err
		}
		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func (a *Agent) promptModelSwitch() string {
	fallbackModels := []string{
		"gemini-3.8-flash",
		"gemini-3.5-flash-lite",
		"gemini-3.1-flash-lite",
	}

	fmt.Println("\nSwitch to a different model? (each has separate quota)")
	for i, m := range fallbackModels {
		fmt.Printf("  %d) %s\n", i+1, m)
	}
	fmt.Println("  0) Cancel")
	fmt.Print("Choice: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	fmt.Sscanf(input, "%d", &choice)

	if choice >= 1 && choice <= len(fallbackModels) {
		return fallbackModels[choice-1]
	}
	return ""
}

func parseRetryDelay(errStr string) time.Duration {
	// Look for "retry in X.XXs" or "retryDelay: Xs" pattern
	if idx := strings.Index(errStr, "retry in "); idx != -1 {
		rest := errStr[idx+9:]
		var seconds float64
		if _, err := fmt.Sscanf(rest, "%fs", &seconds); err == nil {
			return time.Duration(seconds*1000) * time.Millisecond
		}
	}
	if idx := strings.Index(errStr, "retryDelay: "); idx != -1 {
		rest := errStr[idx+12:]
		var seconds int
		if _, err := fmt.Sscanf(rest, "%ds", &seconds); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return 0
}

func (a *Agent) generate(ctx context.Context) (*genai.GenerateContentResponse, error) {
	config := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{
			{
				FunctionDeclarations: a.registry.FunctionDeclarations(),
			},
		},
	}

	if a.systemMsg != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: a.systemMsg}},
		}
	}

	return a.client.Models.GenerateContent(ctx, a.model, a.history.GetMessages(), config)
}

func (a *Agent) ClearHistory() {
	a.history.Clear()
	_ = a.history.Save()
}
