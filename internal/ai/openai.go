package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hluaguo/commity/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	model  string
}

// CommitMessage is the structured output from the AI tool call
type CommitMessage struct {
	Type    string   `json:"type"`    // feat, fix, docs, etc.
	Scope   string   `json:"scope"`   // optional scope
	Subject string   `json:"subject"` // commit subject line
	Body    string   `json:"body"`    // optional commit body
	Files   []string `json:"files"`   // files for this commit (used in split)
}

func (c *CommitMessage) String() string {
	msg := ""
	if c.Type != "" {
		msg = c.Type
		if c.Scope != "" {
			msg += "(" + c.Scope + ")"
		}
		msg += ": "
	}
	msg += c.Subject
	if c.Body != "" {
		msg += "\n\n" + c.Body
	}
	return msg
}

// SplitCommits represents multiple commits for split mode
type SplitCommits struct {
	Commits []CommitMessage `json:"commits"`
}

// Tool definition for single commit
var commitTool = openai.Tool{
	Type: openai.ToolTypeFunction,
	Function: &openai.FunctionDefinition{
		Name:        "submit_commit",
		Description: "Submit a single commit for all changes. Use this when all changes are related.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{
					"type":        "string",
					"description": "Commit type (feat, fix, docs, style, refactor, test, chore)",
				},
				"scope": map[string]any{
					"type":        "string",
					"description": "Optional scope of the change",
				},
				"subject": map[string]any{
					"type":        "string",
					"description": "Short commit subject line WITHOUT the type prefix (max 72 chars). Example: 'add user authentication' not 'feat: add user authentication'",
				},
				"body": map[string]any{
					"type":        "string",
					"description": "Optional longer description",
				},
			},
			"required": []string{"type", "subject"},
		},
	},
}

// Tool definition for split commits
var splitCommitsTool = openai.Tool{
	Type: openai.ToolTypeFunction,
	Function: &openai.FunctionDefinition{
		Name:        "split_commits",
		Description: "Split changes into multiple logical commits. Use this when changes are unrelated and should be separate commits.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"commits": map[string]any{
					"type":        "array",
					"description": "Array of commits, each with its own message and files",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"type": map[string]any{
								"type":        "string",
								"description": "Commit type (feat, fix, docs, style, refactor, test, chore)",
							},
							"scope": map[string]any{
								"type":        "string",
								"description": "Optional scope of the change",
							},
							"subject": map[string]any{
								"type":        "string",
								"description": "Short commit subject line WITHOUT the type prefix (max 72 chars). Example: 'add user authentication' not 'feat: add user authentication'",
							},
							"body": map[string]any{
								"type":        "string",
								"description": "Optional longer description",
							},
							"files": map[string]any{
								"type":        "array",
								"items":       map[string]any{"type": "string"},
								"description": "List of file paths for this commit",
							},
						},
						"required": []string{"type", "subject", "files"},
					},
				},
			},
			"required": []string{"commits"},
		},
	},
}

func New(cfg *config.AIConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key not configured. Set OPENAI_API_KEY or configure in ~/.config/commity/config.toml")
	}

	clientCfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientCfg.BaseURL = cfg.BaseURL
	}

	return &Client{
		client: openai.NewClientWithConfig(clientCfg),
		model:  cfg.Model,
	}, nil
}

// GenerateResult represents the AI's response - either single or split commits
type GenerateResult struct {
	Commits []CommitMessage
	IsSplit bool
}

func (c *Client) GenerateCommitMessage(ctx context.Context, files []string, diff string, conventional bool, types []string, customInstructions string, previousMsg string, feedback string) (*GenerateResult, error) {
	prompt := BuildPrompt(files, diff, conventional, types, customInstructions, previousMsg, feedback)

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: SystemPrompt(),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Tools: []openai.Tool{commitTool, splitCommitsTool},
	})

	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	choice := resp.Choices[0]

	// Check for tool call
	if len(choice.Message.ToolCalls) > 0 {
		toolCall := choice.Message.ToolCalls[0]

		switch toolCall.Function.Name {
		case "submit_commit":
			var commit CommitMessage
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &commit); err != nil {
				return nil, fmt.Errorf("failed to parse commit message: %w", err)
			}
			commit.Files = files // single commit uses all files
			return &GenerateResult{
				Commits: []CommitMessage{commit},
				IsSplit: false,
			}, nil

		case "split_commits":
			var split SplitCommits
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &split); err != nil {
				return nil, fmt.Errorf("failed to parse split commits: %w", err)
			}
			return &GenerateResult{
				Commits: split.Commits,
				IsSplit: true,
			}, nil
		}
	}

	// Fallback to content if no tool call - treat as single commit
	if choice.Message.Content != "" {
		return &GenerateResult{
			Commits: []CommitMessage{{
				Subject: choice.Message.Content,
				Files:   files,
			}},
			IsSplit: false,
		}, nil
	}

	return nil, fmt.Errorf("AI did not return a commit message")
}
