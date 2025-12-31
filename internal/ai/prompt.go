package ai

import (
	"fmt"
	"strings"
)

const systemPrompt = `You are a git commit message generator. Generate concise, meaningful commit messages following best practices.

Rules:
1. Use imperative mood ("Add feature" not "Added feature")
2. Keep subject line under 72 characters
3. Be specific about what changed and why
4. For conventional commits, use appropriate type: feat, fix, docs, style, refactor, test, chore

Tools:
- Use submit_commit for a single commit when all changes are related
- Use split_commits when changes are unrelated and should be separate commits (e.g., a bug fix and a new feature)`

func BuildPrompt(files []string, diff string, conventional bool, types []string, customInstructions string, previousMsg string, feedback string) string {
	var sb strings.Builder

	// Check if this is a regeneration request
	if previousMsg != "" {
		sb.WriteString("The user wants you to regenerate the commit message.\n\n")
		sb.WriteString(fmt.Sprintf("Previous message:\n```\n%s\n```\n\n", previousMsg))
		if feedback != "" {
			sb.WriteString(fmt.Sprintf("User feedback: %s\n\n", feedback))
		}
		sb.WriteString("Generate an improved commit message based on the feedback.\n\n")
	} else {
		sb.WriteString("Generate a commit message for these changes:\n\n")
	}

	sb.WriteString("Files changed:\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("- %s\n", f))
	}

	sb.WriteString("\nDiff:\n```\n")
	// Truncate diff if too long
	if len(diff) > 8000 {
		diff = diff[:8000] + "\n... (truncated)"
	}
	sb.WriteString(diff)
	sb.WriteString("\n```\n")

	if conventional {
		sb.WriteString(fmt.Sprintf("\nUse conventional commit format with one of these types: %s\n", strings.Join(types, ", ")))
	}

	if customInstructions != "" {
		sb.WriteString(fmt.Sprintf("\nAdditional instructions: %s\n", customInstructions))
	}

	sb.WriteString("\nAnalyze the changes and decide: use `submit_commit` for related changes, or `split_commits` if changes should be separate commits.")

	return sb.String()
}

func SystemPrompt() string {
	return systemPrompt
}
