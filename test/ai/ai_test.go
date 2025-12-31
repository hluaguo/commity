package ai_test

import (
	"strings"
	"testing"

	"github.com/hluaguo/commity/internal/ai"
)

func TestCommitMessageString(t *testing.T) {
	tests := []struct {
		name     string
		msg      ai.CommitMessage
		expected string
	}{
		{
			name: "type and subject only",
			msg: ai.CommitMessage{
				Type:    "feat",
				Subject: "add user authentication",
			},
			expected: "feat: add user authentication",
		},
		{
			name: "with scope",
			msg: ai.CommitMessage{
				Type:    "fix",
				Scope:   "auth",
				Subject: "prevent token expiration race condition",
			},
			expected: "fix(auth): prevent token expiration race condition",
		},
		{
			name: "with body",
			msg: ai.CommitMessage{
				Type:    "docs",
				Subject: "update README",
				Body:    "Added installation instructions for Homebrew users.",
			},
			expected: "docs: update README\n\nAdded installation instructions for Homebrew users.",
		},
		{
			name: "full message with scope and body",
			msg: ai.CommitMessage{
				Type:    "refactor",
				Scope:   "config",
				Subject: "extract validation logic",
				Body:    "Moved validation into separate functions for better testability.",
			},
			expected: "refactor(config): extract validation logic\n\nMoved validation into separate functions for better testability.",
		},
		{
			name: "subject only (no type)",
			msg: ai.CommitMessage{
				Subject: "Update dependencies",
			},
			expected: "Update dependencies",
		},
		{
			name: "empty scope ignored",
			msg: ai.CommitMessage{
				Type:    "chore",
				Scope:   "",
				Subject: "update dependencies",
			},
			expected: "chore: update dependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.String()
			if got != tt.expected {
				t.Errorf("String() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}

func TestCommitMessageFiles(t *testing.T) {
	msg := ai.CommitMessage{
		Type:    "feat",
		Subject: "add new feature",
		Files:   []string{"file1.go", "file2.go", "file3.go"},
	}

	if len(msg.Files) != 3 {
		t.Errorf("expected 3 files, got %d", len(msg.Files))
	}
	if msg.Files[0] != "file1.go" {
		t.Errorf("expected first file 'file1.go', got %q", msg.Files[0])
	}
}

func TestBuildPromptBasic(t *testing.T) {
	files := []string{"main.go", "config.go"}
	diff := "diff --git a/main.go b/main.go\n+// new comment"
	types := []string{"feat", "fix", "docs"}

	prompt := ai.BuildPrompt(files, diff, true, types, "", "", "")

	// Check that files are included
	if !strings.Contains(prompt, "main.go") {
		t.Error("prompt should contain file name 'main.go'")
	}
	if !strings.Contains(prompt, "config.go") {
		t.Error("prompt should contain file name 'config.go'")
	}

	// Check that diff is included
	if !strings.Contains(prompt, "+// new comment") {
		t.Error("prompt should contain diff content")
	}

	// Check conventional commit types
	if !strings.Contains(prompt, "feat, fix, docs") {
		t.Error("prompt should contain commit types")
	}
}

func TestBuildPromptWithCustomInstructions(t *testing.T) {
	files := []string{"api.go"}
	diff := "some diff"
	types := []string{"feat"}
	customInstructions := "Always mention the ticket number"

	prompt := ai.BuildPrompt(files, diff, true, types, customInstructions, "", "")

	if !strings.Contains(prompt, "Always mention the ticket number") {
		t.Error("prompt should contain custom instructions")
	}
	if !strings.Contains(prompt, "Additional instructions:") {
		t.Error("prompt should have 'Additional instructions' prefix")
	}
}

func TestBuildPromptRegeneration(t *testing.T) {
	files := []string{"handler.go"}
	diff := "some diff"
	types := []string{"fix"}
	previousMsg := "fix: update handler"
	feedback := "Make it more descriptive"

	prompt := ai.BuildPrompt(files, diff, true, types, "", previousMsg, feedback)

	if !strings.Contains(prompt, "regenerate") {
		t.Error("prompt should mention regeneration")
	}
	if !strings.Contains(prompt, previousMsg) {
		t.Error("prompt should contain previous message")
	}
	if !strings.Contains(prompt, feedback) {
		t.Error("prompt should contain user feedback")
	}
}

func TestBuildPromptRegenerationWithoutFeedback(t *testing.T) {
	files := []string{"service.go"}
	diff := "some diff"
	types := []string{"refactor"}
	previousMsg := "refactor: clean up code"

	prompt := ai.BuildPrompt(files, diff, true, types, "", previousMsg, "")

	if !strings.Contains(prompt, "regenerate") {
		t.Error("prompt should mention regeneration")
	}
	if !strings.Contains(prompt, previousMsg) {
		t.Error("prompt should contain previous message")
	}
	// Should not contain "User feedback:" when no feedback provided
	if strings.Contains(prompt, "User feedback:") {
		t.Error("prompt should not contain 'User feedback:' when feedback is empty")
	}
}

func TestBuildPromptNonConventional(t *testing.T) {
	files := []string{"readme.md"}
	diff := "some diff"
	types := []string{"feat", "fix"}

	prompt := ai.BuildPrompt(files, diff, false, types, "", "", "")

	// When conventional is false, should not mention commit types
	if strings.Contains(prompt, "conventional commit format") {
		t.Error("prompt should not mention conventional format when disabled")
	}
}

func TestBuildPromptDiffTruncation(t *testing.T) {
	files := []string{"large.go"}
	// Create a diff larger than 8000 characters
	largeDiff := strings.Repeat("a", 10000)
	types := []string{"feat"}

	prompt := ai.BuildPrompt(files, largeDiff, true, types, "", "", "")

	// Check that diff was truncated
	if !strings.Contains(prompt, "... (truncated)") {
		t.Error("large diff should be truncated")
	}

	// Original 10000 char diff should not be fully present
	if strings.Contains(prompt, strings.Repeat("a", 9000)) {
		t.Error("truncated diff should not contain full original content")
	}
}

func TestSystemPrompt(t *testing.T) {
	sp := ai.SystemPrompt()

	if sp == "" {
		t.Error("SystemPrompt should not be empty")
	}

	// Check for key content
	if !strings.Contains(sp, "commit") {
		t.Error("SystemPrompt should mention commits")
	}
	if !strings.Contains(sp, "submit_commit") {
		t.Error("SystemPrompt should mention submit_commit tool")
	}
	if !strings.Contains(sp, "split_commits") {
		t.Error("SystemPrompt should mention split_commits tool")
	}
}

func TestSplitCommitsStructure(t *testing.T) {
	// Test the SplitCommits type
	split := ai.SplitCommits{
		Commits: []ai.CommitMessage{
			{
				Type:    "feat",
				Subject: "add feature A",
				Files:   []string{"feature_a.go"},
			},
			{
				Type:    "fix",
				Subject: "fix bug B",
				Files:   []string{"bug_b.go"},
			},
		},
	}

	if len(split.Commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(split.Commits))
	}
	if split.Commits[0].Type != "feat" {
		t.Errorf("expected first commit type 'feat', got %q", split.Commits[0].Type)
	}
	if split.Commits[1].Type != "fix" {
		t.Errorf("expected second commit type 'fix', got %q", split.Commits[1].Type)
	}
}

func TestGenerateResultStructure(t *testing.T) {
	// Test single commit result
	singleResult := ai.GenerateResult{
		Commits: []ai.CommitMessage{
			{Type: "feat", Subject: "add feature"},
		},
		IsSplit: false,
	}

	if singleResult.IsSplit {
		t.Error("single commit should not be marked as split")
	}
	if len(singleResult.Commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(singleResult.Commits))
	}

	// Test split commits result
	splitResult := ai.GenerateResult{
		Commits: []ai.CommitMessage{
			{Type: "feat", Subject: "add feature"},
			{Type: "fix", Subject: "fix bug"},
		},
		IsSplit: true,
	}

	if !splitResult.IsSplit {
		t.Error("split commits should be marked as split")
	}
	if len(splitResult.Commits) != 2 {
		t.Errorf("expected 2 commits, got %d", len(splitResult.Commits))
	}
}
