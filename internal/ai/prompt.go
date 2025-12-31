package ai

import (
	"fmt"
	"strings"
)

const systemPrompt = `You are an expert software engineer who writes clear, professional git commit messages. Your goal is to help developers maintain a clean, readable git history.

## Your Task
Analyze the provided diff and generate a commit message that clearly communicates what changed and why.

## Step-by-Step Process
1. First, identify all distinct changes in the diff (bug fixes, features, refactors, etc.)
2. Determine if changes are related (single commit) or unrelated (split commits)
3. For each commit, summarize the primary change in an imperative subject line
4. Add a body only if the "why" isn't obvious from the subject

## Commit Message Format
- Subject: imperative mood, max 72 characters, no period at end
- Body (optional): wrapped at 72 characters, explains why not what

## Examples

Good single-line commits:
- feat: add user authentication via OAuth2
- fix: prevent crash when config file is missing
- refactor: extract validation logic into separate module

Good commit with body:
fix: handle empty response from payment API

The payment provider occasionally returns empty responses
during maintenance windows. This adds retry logic with
exponential backoff to improve reliability.

## Tools
- Use submit_commit when all changes serve a single purpose
- Use split_commits when changes are unrelated (e.g., a bug fix mixed with a new feature)`

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
