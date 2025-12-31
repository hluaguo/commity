package ai

import (
	"fmt"
	"strings"
)

// Truncation limits (exported for testing)
const (
	MaxDiffSize = 12000 // max total diff size in characters
	ShowLines   = 100   // lines to show in each segment
	SkipLines   = 50    // lines to skip between segments
)

const systemPrompt = `You are an expert software engineer who writes clear, professional git commit messages. Your goal is to help developers maintain a clean, atomic git history.

## Your Task
Analyze the provided diff and generate commit message(s). Prefer splitting into multiple atomic commits when changes serve different purposes.

## When to Split Commits
PREFER split_commits when you see:
- Different types of changes (feat + fix, refactor + docs, etc.)
- Changes to unrelated parts of the codebase
- A bug fix alongside a new feature
- Formatting/style changes mixed with logic changes
- Test additions for existing code + new feature code
- Multiple independent improvements

Use submit_commit ONLY when ALL changes serve a single, cohesive purpose.

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
- split_commits: Use this for most cases with multiple distinct changes (PREFERRED)
- submit_commit: Use only when all changes are tightly related to one purpose`

func BuildPrompt(files []string, diff string, conventional bool, types []string, customInstructions string, previousMsg string, feedback string) string {
	var sb strings.Builder

	// Check if this is a regeneration request
	if previousMsg == "" {
		sb.WriteString("Generate a commit message for these changes:\n\n")
	} else {
		sb.WriteString("The user wants you to regenerate the commit message.\n\n")
		sb.WriteString(fmt.Sprintf("Previous message:\n```\n%s\n```\n\n", previousMsg))
		if feedback != "" {
			sb.WriteString(fmt.Sprintf("User feedback: %s\n\n", feedback))
		}
		sb.WriteString("Generate an improved commit message based on the feedback.\n\n")
	}

	sb.WriteString("Files changed:\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("- %s\n", f))
	}

	sb.WriteString("\nDiff:\n```\n")
	sb.WriteString(truncateDiff(diff))
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

// truncateDiff intelligently truncates a diff while preserving context
func truncateDiff(diff string) string {
	var result strings.Builder
	files := splitByFiles(diff)

	for _, file := range files {
		// Always apply hunk truncation for large hunks
		truncatedFile := truncateFile(file)
		result.WriteString(truncatedFile)

		// Stop if we've exceeded the overall limit
		if result.Len() > MaxDiffSize {
			result.WriteString("\n... (remaining files truncated) ...")
			break
		}
	}

	return result.String()
}

// splitByFiles splits a diff into per-file sections
func splitByFiles(diff string) []string {
	var files []string
	lines := strings.Split(diff, "\n")
	var current strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") && current.Len() > 0 {
			files = append(files, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		files = append(files, current.String())
	}

	return files
}

// truncateFile truncates a single file's diff, preserving hunks structure
func truncateFile(fileDiff string) string {
	lines := strings.Split(fileDiff, "\n")
	var result strings.Builder
	var hunkLines []string
	inHunk := false

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Flush previous hunk
			if len(hunkLines) > 0 {
				result.WriteString(truncateHunk(hunkLines))
			}
			hunkLines = []string{line}
			inHunk = true
		} else if inHunk {
			hunkLines = append(hunkLines, line)
		} else {
			// Header lines (diff --git, ---, +++, etc.)
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	// Flush last hunk
	if len(hunkLines) > 0 {
		result.WriteString(truncateHunk(hunkLines))
	}

	return result.String()
}

// truncateHunk truncates a hunk using repeating show/skip pattern
func truncateHunk(lines []string) string {
	// Small hunks don't need truncation
	if len(lines) <= ShowLines {
		return strings.Join(lines, "\n") + "\n"
	}

	var result strings.Builder
	i := 0
	lineNum := 0 // track actual line number for context

	for i < len(lines) {
		// Show segment
		end := i + ShowLines
		if end > len(lines) {
			end = len(lines)
		}
		for j := i; j < end; j++ {
			result.WriteString(lines[j])
			result.WriteString("\n")
			lineNum++
		}
		i = end

		// Skip segment (if there's more content)
		if i < len(lines) {
			skipEnd := i + SkipLines
			if skipEnd > len(lines) {
				skipEnd = len(lines)
			}
			skipped := skipEnd - i
			if skipped > 0 {
				// Provide context: line range and sample of what's skipped
				startLine := lineNum + 1
				endLine := lineNum + skipped
				result.WriteString(fmt.Sprintf("... [lines %d-%d: %d lines skipped - similar changes continue] ...\n", startLine, endLine, skipped))
				lineNum += skipped
			}
			i = skipEnd
		}
	}

	return result.String()
}
