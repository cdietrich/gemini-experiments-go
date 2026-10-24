package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"gemini-agent/agent"
	"gemini-agent/history"
	"gemini-agent/permissions"
	"gemini-agent/tools"

	"github.com/chzyer/readline"
	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

func main() {
	clearFlag := flag.Bool("clear", false, "Clear conversation history")
	modelFlag := flag.String("model", "gemini-3.8-flash", "Model to use")
	workDirFlag := flag.String("workdir", ".", "Working directory")
	flag.Parse()

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: error loading .env: %v\n", err)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: GEMINI_API_KEY or GOOGLE_API_KEY environment variable is required")
		fmt.Fprintln(os.Stderr, "Set it in your environment or create a .env file")
		os.Exit(1)
	}

	workDir, err := filepath.Abs(*workDirFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving working directory: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{WorkDir: workDir})
	registry.Register(&tools.WriteFileTool{WorkDir: workDir})
	registry.Register(&tools.EditFileTool{WorkDir: workDir})
	registry.Register(tools.NewBashTool(workDir))
	registry.Register(&tools.GlobTool{WorkDir: workDir})
	registry.Register(&tools.GrepTool{WorkDir: workDir})
	registry.Register(&tools.ListDirTool{WorkDir: workDir})

	h := history.New(history.DefaultPath())
	if err := h.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error loading history: %v\n", err)
	}

	perms := permissions.NewManager(workDir)

	ag := agent.New(client, registry,
		agent.WithModel(*modelFlag),
		agent.WithHistory(h),
		agent.WithPermissions(perms),
		agent.WithSystemMessage(systemPrompt(workDir)),
	)

	if *clearFlag {
		ag.ClearHistory()
		fmt.Println("History cleared.")
		return
	}

	fmt.Println("Gemini Coding Agent")
	fmt.Printf("Working directory: %s\n", workDir)
	fmt.Println("Type your message and press Enter. Ctrl+C to exit.")
	fmt.Println("Use /clear to clear history, /quit to exit, /help for more.")
	fmt.Println("Permissions will be asked for each tool (stored in .gemini-agent/).")
	fmt.Println()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     filepath.Join(os.TempDir(), ".gemini_agent_history"),
		AutoComplete:    nil,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line, err := rl.Readline()
		if err != nil {
			return
		}

		line = trimWhitespace(line)
		if line == "" {
			continue
		}

		switch line {
		case "/quit", "/exit":
			return
		case "/clear":
			ag.ClearHistory()
			fmt.Println("History cleared.")
			continue
		case "/help":
			printHelp()
			continue
		}

		fmt.Println()
		response, err := ag.Run(ctx, line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Println(response)
		fmt.Println()
	}
}

func trimWhitespace(s string) string {
	return strings.TrimSpace(s)
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /clear - Clear conversation history")
	fmt.Println("  /quit  - Exit the agent")
	fmt.Println("  /help  - Show this help message")
	fmt.Println()
	fmt.Println("Permissions are stored in .gemini-agent/permissions.json")
}

func systemPrompt(workDir string) string {
	return `You are a helpful coding assistant with access to tools for file operations, shell commands, and code search.

Working directory: ` + workDir + `

Available tools:
- read_file: Read file contents
- write_file: Create or overwrite a file
- edit_file: Replace text in a file
- run_bash: Execute shell commands
- glob: Find files by pattern
- grep: Search file contents with regex
- list_dir: List directory contents

Guidelines:
1. Read files before editing to understand context
2. Use edit_file for small changes, write_file for new files
3. Run tests/commands to verify your changes
4. Explain your actions concisely
5. Ask for clarification if the request is ambiguous`
}
