# Gemini Coding Agent

A terminal-based coding agent powered by Google's Gemini AI. It can read, write, and edit files, execute shell commands, and search your codebase.

## Features

- **File Operations**: Read, write, and edit files
- **Shell Commands**: Execute bash commands with safety checks
- **Code Search**: Grep and glob for finding files and content
- **Conversation History**: Persistent chat history across sessions
- **Permission System**: Ask-before-execute with "always allow" option
- **Rate Limit Handling**: Automatic retry with exponential backoff
- **Model Fallback**: Switch models when quota exceeded

## Installation

Requires Go 1.26.10 or newer.

```bash
go build -o gemini-agent .
```

## Usage

```bash
# Set your API key
export GEMINI_API_KEY=your-api-key
# or create a .env file

# Run
./gemini-agent

# Options
./gemini-agent -model gemini-2.0-flash -workdir ./src
./gemini-agent -clear  # Clear conversation history
```

## Configuration

### API Key

Set via environment variable or `.env` file:
```
GEMINI_API_KEY=your-api-key
```

Get your API key at [Google AI Studio](https://aistudio.google.com/).

### Permissions

Permissions are stored per-project in `.gemini-agent/permissions.json`.

When a tool is called, you'll be prompted:
- `y` - Allow once
- `n` - Deny
- `a` - Always allow this tool (saves to permissions file)

### Models

Default model is `gemini-3-flash-preview`. Available models:

| Model | Input/1M | Output/1M | Notes |
|-------|----------|-----------|-------|
| gemini-3-flash-preview | $0.50 | $3.00 | Latest, most capable |
| gemini-2.5-flash | $0.30 | $2.50 | Good balance |
| gemini-2.0-flash | $0.10 | $0.40 | Most cost-effective |

## Commands

- `/help` - Show available commands
- `/clear` - Clear conversation history
- `/quit` - Exit the agent

## Available Tools

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Create or overwrite a file |
| `edit_file` | Replace text in a file |
| `run_bash` | Execute shell commands |
| `glob` | Find files by pattern |
| `grep` | Search file contents with regex |
| `list_dir` | List directory contents |

## Rate Limits (Free Tier)

| Model | RPM | RPD |
|-------|-----|-----|
| gemini-3-flash-preview | 15 | 20 |
| gemini-2.0-flash | 15 | 1500 |

Each model has separate quotas. The agent will prompt to switch models when quota is exceeded.

## Project Structure

```
gemini-agent/
├── main.go              # CLI entry point
├── agent/
│   └── agent.go         # Agent loop, Gemini integration
├── tools/
│   ├── tools.go         # Tool interface and registry
│   ├── files.go         # File operations
│   ├── bash.go          # Shell commands
│   └── search.go        # Grep, glob, list_dir
├── history/
│   └── history.go       # Conversation persistence
├── permissions/
│   └── permissions.go   # Permission management
└── go.mod
```

## License

MIT
