package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hluaguo/commity/internal/git"
)

func TestFileStatusStatusLabel(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"modified", "M", "modified"},
		{"added", "A", "added"},
		{"deleted", "D", "deleted"},
		{"renamed", "R", "renamed"},
		{"untracked", "??", "untracked"},
		{"unknown", "X", "X"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := git.FileStatus{
				Path:   "test.go",
				Status: tt.status,
				Staged: false,
			}

			got := fs.StatusLabel()
			if got != tt.expected {
				t.Errorf("StatusLabel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFileStatusFields(t *testing.T) {
	fs := git.FileStatus{
		Path:   "internal/config/config.go",
		Status: "M",
		Staged: true,
	}

	if fs.Path != "internal/config/config.go" {
		t.Errorf("Path = %q, want %q", fs.Path, "internal/config/config.go")
	}
	if fs.Status != "M" {
		t.Errorf("Status = %q, want %q", fs.Status, "M")
	}
	if !fs.Staged {
		t.Error("Staged should be true")
	}
}

func TestFileStatusUnstaged(t *testing.T) {
	fs := git.FileStatus{
		Path:   "README.md",
		Status: "M",
		Staged: false,
	}

	if fs.Staged {
		t.Error("Staged should be false for unstaged changes")
	}
	if fs.StatusLabel() != "modified" {
		t.Errorf("StatusLabel() = %q, want %q", fs.StatusLabel(), "modified")
	}
}

func TestFileStatusUntracked(t *testing.T) {
	fs := git.FileStatus{
		Path:   "new_file.go",
		Status: "??",
		Staged: false,
	}

	if fs.Staged {
		t.Error("Untracked files should not be staged")
	}
	if fs.StatusLabel() != "untracked" {
		t.Errorf("StatusLabel() = %q, want %q", fs.StatusLabel(), "untracked")
	}
}

// Helper to create a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to config user.email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to config user.name: %v", err)
	}

	// Save current dir and change to temp dir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpDir: %v", err)
	}

	cleanup := func() {
		_ = os.Chdir(originalDir)
	}

	return tmpDir, cleanup
}

func TestDiffAllWithUntrackedFile(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create an untracked file
	untrackedFile := filepath.Join(tmpDir, "untracked.go")
	if err := os.WriteFile(untrackedFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	repo, err := git.New()
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	diff, err := repo.DiffAll([]string{"untracked.go"})
	if err != nil {
		t.Fatalf("DiffAll failed: %v", err)
	}

	// Should contain the untracked file content
	if !strings.Contains(diff, "package main") {
		t.Error("DiffAll should include untracked file content")
	}
	if !strings.Contains(diff, "func main()") {
		t.Error("DiffAll should include untracked file content")
	}
}

func TestDiffAllWithUntrackedDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create an untracked directory with files
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create files in the directory
	file1 := filepath.Join(testDir, "file1.go")
	if err := os.WriteFile(file1, []byte("package testdir\n\nvar File1 = true\n"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}

	file2 := filepath.Join(testDir, "file2.go")
	if err := os.WriteFile(file2, []byte("package testdir\n\nvar File2 = false\n"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	repo, err := git.New()
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	diff, err := repo.DiffAll([]string{"testdir"})
	if err != nil {
		t.Fatalf("DiffAll failed: %v", err)
	}

	// Should contain content from both files in the directory
	if !strings.Contains(diff, "File1") {
		t.Error("DiffAll should include content from file1 in directory")
	}
	if !strings.Contains(diff, "File2") {
		t.Error("DiffAll should include content from file2 in directory")
	}
}

func TestDiffAllWithMixedTrackedAndUntracked(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create and commit a tracked file
	trackedFile := filepath.Join(tmpDir, "tracked.go")
	if err := os.WriteFile(trackedFile, []byte("package main\n\nvar Original = 1\n"), 0644); err != nil {
		t.Fatalf("failed to create tracked file: %v", err)
	}

	cmd := exec.Command("git", "add", "tracked.go")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// Modify the tracked file
	if err := os.WriteFile(trackedFile, []byte("package main\n\nvar Modified = 2\n"), 0644); err != nil {
		t.Fatalf("failed to modify tracked file: %v", err)
	}

	// Create an untracked file
	untrackedFile := filepath.Join(tmpDir, "untracked.go")
	if err := os.WriteFile(untrackedFile, []byte("package main\n\nvar Untracked = 3\n"), 0644); err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	repo, err := git.New()
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	diff, err := repo.DiffAll([]string{"tracked.go", "untracked.go"})
	if err != nil {
		t.Fatalf("DiffAll failed: %v", err)
	}

	// Should contain changes from tracked file
	if !strings.Contains(diff, "Modified") {
		t.Error("DiffAll should include tracked file changes")
	}

	// Should also contain untracked file content
	if !strings.Contains(diff, "Untracked") {
		t.Error("DiffAll should include untracked file content")
	}
}

func TestDiffAllWithNestedUntrackedDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create nested untracked directories
	nestedDir := filepath.Join(tmpDir, "parent", "child")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	// Create file in nested directory
	nestedFile := filepath.Join(nestedDir, "nested.go")
	if err := os.WriteFile(nestedFile, []byte("package child\n\nvar Nested = true\n"), 0644); err != nil {
		t.Fatalf("failed to create nested file: %v", err)
	}

	repo, err := git.New()
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	diff, err := repo.DiffAll([]string{"parent"})
	if err != nil {
		t.Fatalf("DiffAll failed: %v", err)
	}

	// Should contain content from nested file
	if !strings.Contains(diff, "Nested") {
		t.Error("DiffAll should include content from nested directory files")
	}
}
