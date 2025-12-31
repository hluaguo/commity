# commity

AI-powered git commit message generator with a beautiful terminal UI.

## Features

- **AI-Generated Commits**: Generates meaningful commit messages using OpenAI-compatible APIs
- **Smart Split Detection**: Automatically suggests splitting unrelated changes into separate commits
- **Conventional Commits**: Follows conventional commit format (feat, fix, docs, etc.)
- **Interactive TUI**: Beautiful terminal interface for file selection and message confirmation
- **Customizable Themes**: Choose from tokyonight, dracula, catppuccin, or nord
- **Custom Instructions**: Add your own instructions to guide AI message generation

## Installation

### Go Install

```bash
go install github.com/hluaguo/commity/cmd/commity@latest
```

### From Source

```bash
git clone https://github.com/hluaguo/commity.git
cd commity
make install
```

### Supported Providers

Any OpenAI-compatible API works: (will support more)

- OpenAI
- OpenRouter
- Ollama (local)
- Azure OpenAI
- Any OpenAI-compatible endpoint

## Usage

```bash
# Run in any git repository with changes
commity
```

### Workflow

1. **Select files**: Choose which files to include in the commit
2. **Generate**: AI analyzes changes and generates commit message
3. **Confirm**: Review the message, edit if needed, or regenerate with feedback
4. **Commit**: Confirm to create the commit

### Key Bindings

| Screen      | Key      | Action                |
| ----------- | -------- | --------------------- |
| File Select | `space`  | Toggle file selection |
| File Select | `enter`  | Submit selection      |
| File Select | `s`      | Open settings         |
| File Select | `q`      | Quit                  |
| Confirm     | `enter`  | Select action         |
| Confirm     | `e`      | Edit message          |
| Edit        | `ctrl+s` | Save changes          |
| Edit        | `esc`    | Cancel edit           |

## Development

```bash
# Build
make build

# Run tests
make test

# Lint
make lint

# Install locally
make install
```

## License

MIT License - see [LICENSE](LICENSE) for details.
