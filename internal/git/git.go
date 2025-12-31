package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type FileStatus struct {
	Path   string
	Status string // M, A, D, ??, R, etc.
	Staged bool
}

func (f FileStatus) StatusLabel() string {
	switch f.Status {
	case "M":
		return "modified"
	case "A":
		return "added"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "??":
		return "untracked"
	default:
		return f.Status
	}
}

type Repository struct {
	path string
}

func New() (*Repository, error) {
	// Check if we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}
	return &Repository{path: strings.TrimSpace(string(out))}, nil
}

func (r *Repository) Status() ([]FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	var files []FileStatus
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 4 {
			continue
		}

		// Format: XY PATH
		// X = staged status, Y = unstaged status
		x := line[0]
		y := line[1]
		path := strings.TrimSpace(line[3:])

		// Handle renamed files (R  old -> new)
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[len(parts)-1]
		}

		// Determine status
		var status string
		var staged bool

		if x == '?' && y == '?' {
			status = "??"
			staged = false
		} else if x != ' ' && x != '?' {
			status = string(x)
			staged = true
		} else if y != ' ' {
			status = string(y)
			staged = false
		}

		if status != "" {
			files = append(files, FileStatus{
				Path:   path,
				Status: status,
				Staged: staged,
			})
		}
	}

	return files, scanner.Err()
}

func (r *Repository) Diff(files []string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--")
	args = append(args, files...)

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(out), nil
}

func (r *Repository) DiffAll(files []string) (string, error) {
	// Get both staged and unstaged diff
	staged, _ := r.Diff(files, true)
	unstaged, _ := r.Diff(files, false)

	if staged == "" && unstaged == "" {
		// For untracked files, show content using git diff --no-index
		// Note: --no-index returns exit code 1 when there are differences, so we ignore the error
		var buf bytes.Buffer
		for _, f := range files {
			cmd := exec.Command("git", "diff", "--no-index", "--", "/dev/null", f)
			out, _ := cmd.CombinedOutput()
			buf.Write(out)
		}
		// If still empty, just read file contents
		if buf.Len() == 0 {
			for _, f := range files {
				content, err := os.ReadFile(f)
				if err == nil {
					buf.WriteString(fmt.Sprintf("+++ %s\n", f))
					buf.Write(content)
					buf.WriteString("\n")
				}
			}
		}
		return buf.String(), nil
	}

	return staged + unstaged, nil
}

func (r *Repository) Add(files []string) error {
	args := []string{"add", "--"}
	args = append(args, files...)
	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	return nil
}

func (r *Repository) Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	return nil
}
