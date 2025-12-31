package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const minStatusLineLength = 4 // "XY " + at least 1 char path

// FileStatus represents the git status of a file in the working tree.
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

// Repository provides git operations for a local repository.
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
		if len(line) < minStatusLineLength {
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
			// Check if path is a directory and expand it
			info, err := os.Stat(path)
			if err == nil && info.IsDir() {
				// Expand directory into individual files
				expandedFiles := expandDirectory(path, status, staged)
				files = append(files, expandedFiles...)
			} else {
				files = append(files, FileStatus{
					Path:   path,
					Status: status,
					Staged: staged,
				})
			}
		}
	}

	return files, scanner.Err()
}

// expandDirectory recursively expands a directory into individual FileStatus entries
func expandDirectory(dir string, status string, staged bool) []FileStatus {
	var files []FileStatus

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			// Recursively expand subdirectories
			files = append(files, expandDirectory(fullPath, status, staged)...)
		} else {
			files = append(files, FileStatus{
				Path:   fullPath,
				Status: status,
				Staged: staged,
			})
		}
	}

	return files
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
	var buf bytes.Buffer

	// Get both staged and unstaged diff for tracked files
	staged, _ := r.Diff(files, true)
	unstaged, _ := r.Diff(files, false)
	buf.WriteString(staged)
	buf.WriteString(unstaged)

	// Also handle untracked files - check each file individually
	for _, f := range files {
		cmd := exec.Command("git", "ls-files", "--error-unmatch", f)
		if err := cmd.Run(); err != nil {
			// File/directory is untracked
			r.appendUntrackedContent(&buf, f)
		}
	}

	return buf.String(), nil
}

// appendUntrackedContent adds content of untracked file or directory to buffer
func (r *Repository) appendUntrackedContent(buf *bytes.Buffer, path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if info.IsDir() {
		// For directories, read all files recursively
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())
			r.appendUntrackedContent(buf, fullPath)
		}
		return
	}

	// For files, try git diff --no-index first
	diffCmd := exec.Command("git", "diff", "--no-index", "--", "/dev/null", path)
	out, _ := diffCmd.CombinedOutput()
	if len(out) > 0 {
		buf.Write(out)
	} else {
		// Fallback to reading file content directly
		content, err := os.ReadFile(path)
		if err == nil {
			buf.WriteString(fmt.Sprintf("+++ %s\n", path))
			buf.Write(content)
			buf.WriteString("\n")
		}
	}
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

func (r *Repository) Branch() string {
	cmd := exec.Command("git", "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// DiffStats returns lines added and removed for the given files
func (r *Repository) DiffStats(files []string) (added, removed int) {
	// Get stats for staged + unstaged
	for _, staged := range []bool{true, false} {
		args := []string{"diff", "--numstat"}
		if staged {
			args = append(args, "--cached")
		}
		args = append(args, "--")
		args = append(args, files...)

		cmd := exec.Command("git", args...)
		out, err := cmd.Output()
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			line := scanner.Text()
			var a, r int
			_, _ = fmt.Sscanf(line, "%d\t%d", &a, &r)
			added += a
			removed += r
		}
	}

	// For untracked files, count lines
	for _, f := range files {
		cmd := exec.Command("git", "ls-files", "--error-unmatch", f)
		if err := cmd.Run(); err != nil {
			// File is untracked, count its lines
			content, err := os.ReadFile(f)
			if err == nil {
				lines := bytes.Count(content, []byte("\n"))
				if len(content) > 0 && content[len(content)-1] != '\n' {
					lines++
				}
				added += lines
			}
		}
	}

	return added, removed
}
