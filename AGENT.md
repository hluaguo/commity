# AGENT.md

This file provides guidance to AI coding agents when working with code in this repository.

## Build Commands

```bash
make build      # Build binary to ./commity
make install    # Install to $GOPATH/bin
make test       # Run all tests
make lint       # Run golangci-lint
make coverage   # Generate coverage report (coverage.html)
```

Run a single test:
```bash
go test -v -run TestName ./path/to/package
```

## Architecture

Commity is a TUI application that generates git commit messages using AI. It uses the Bubble Tea framework for the terminal UI.

### Package Structure

- `cmd/commity/main.go` - Entry point, orchestrates config loading, git repo init, AI client init, and TUI launch
- `internal/config/` - TOML config loading from `~/.config/commity/config.toml`, supports env vars (`OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`)
- `internal/git/` - Git operations via shell commands (status, diff, add, commit)
- `internal/ai/` - OpenAI-compatible API client with tool-calling for structured commit output
- `internal/tui/` - Bubble Tea model with state machine (file select → generating → confirm → committing → done)

### AI Integration

The AI module uses OpenAI function calling with two tools:
- `submit_commit` - Single commit for related changes
- `split_commits` - Multiple atomic commits when changes are unrelated

The system prompt in `internal/ai/prompt.go` instructs the model to prefer splitting commits. Large diffs are truncated with a show/skip pattern (100 lines shown, 50 skipped) to fit context limits.

### TUI State Machine

States flow: `stateInit` (first run) → `stateFileSelect` → `stateGenerating` → `stateConfirm` → `stateCommitting` → `stateDone`

The confirm view supports regeneration with user feedback and manual message editing.

### Configuration

Config struct in `internal/config/config.go` with sections: `General`, `AI`, `Commit`, `UI`. Environment variables override config file values.

## Testing

Tests are in `test/` directory mirroring `internal/` structure. Tests focus on prompt building and message formatting - no mocked API calls.
