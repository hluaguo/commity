package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/hluaguo/commity/internal/ai"
	"github.com/hluaguo/commity/internal/config"
	"github.com/hluaguo/commity/internal/git"
)

type state int

const (
	stateFileSelect state = iota
	stateGenerating
	stateConfirm
	stateCommitting
	stateDone
	stateError
)

type Model struct {
	state    state
	cfg      *config.Config
	repo     *git.Repository
	aiClient *ai.Client

	files     []git.FileStatus
	selected  []string
	confirmed bool

	// Commit handling (supports split commits)
	commits      []ai.CommitMessage
	currentIndex int
	isSplit      bool
	completed    []bool // track which commits are done

	form      *huh.Form
	spinner   spinner.Model
	err       error
	termWidth int
}

type generateMsg struct {
	result *ai.GenerateResult
	err    error
}

type commitMsg struct {
	err error
}

func New(cfg *config.Config, repo *git.Repository, aiClient *ai.Client) (*Model, error) {
	files, err := repo.Status()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no changes to commit")
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		state:     stateFileSelect,
		cfg:       cfg,
		repo:      repo,
		aiClient:  aiClient,
		files:     files,
		spinner:   s,
		termWidth: getTermWidth(),
	}

	m.initFileSelectForm()
	return m, nil
}

func (m *Model) initFileSelectForm() {
	options := make([]huh.Option[string], len(m.files))

	// Pre-populate selected with already staged files
	m.selected = nil
	for i, f := range m.files {
		label := fmt.Sprintf("[%s] %s", f.Status, f.Path)
		options[i] = huh.NewOption(label, f.Path).Selected(f.Staged)
		if f.Staged {
			m.selected = append(m.selected, f.Path)
		}
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select files to commit (space to toggle, enter to confirm)").
				Options(options...).
				Value(&m.selected),
		),
	).WithTheme(huh.ThemeDracula())
}

func (m *Model) initConfirmForm() {
	m.confirmed = true // default to yes
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Commit with this message?").
				Affirmative("Yes").
				Negative("No").
				Value(&m.confirmed),
		),
	).WithTheme(huh.ThemeDracula())
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.form.Init(), m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case generateMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.commits = msg.result.Commits
		m.isSplit = msg.result.IsSplit
		m.currentIndex = 0
		m.completed = make([]bool, len(m.commits))
		m.state = stateConfirm
		m.initConfirmForm()
		return m, m.form.Init()

	case commitMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.completed[m.currentIndex] = true
		m.currentIndex++

		// Check if more commits to process
		if m.currentIndex < len(m.commits) {
			m.state = stateConfirm
			m.initConfirmForm()
			return m, m.form.Init()
		}

		m.state = stateDone
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.state {
	case stateFileSelect:
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			if len(m.selected) == 0 {
				m.state = stateError
				m.err = fmt.Errorf("no files selected")
				return m, nil
			}
			m.state = stateGenerating
			return m, m.generateCommitMessage()
		}

		return m, cmd

	case stateConfirm:
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			if m.confirmed {
				m.state = stateCommitting
				return m, m.doCommit()
			}
			// User said no - regenerate
			m.state = stateGenerating
			return m, m.generateCommitMessage()
		}

		return m, cmd

	case stateGenerating, stateCommitting:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("commity"))
	s.WriteString("\n\n")

	switch m.state {
	case stateFileSelect:
		s.WriteString(m.form.View())

	case stateGenerating:
		s.WriteString(m.spinner.View())
		s.WriteString(" Generating commit message...")

	case stateConfirm:
		if m.isSplit {
			s.WriteString(fmt.Sprintf("Commit %d of %d:\n\n", m.currentIndex+1, len(m.commits)))
		} else {
			s.WriteString("Commit message:\n\n")
		}
		commit := m.commits[m.currentIndex]
		// Wrap message box to terminal width (minus border padding)
		msgWidth := m.termWidth - 8
		if msgWidth < 40 {
			msgWidth = 40
		}
		s.WriteString(messageStyle.Width(msgWidth).Render(commit.String()))
		if m.isSplit && len(commit.Files) > 0 {
			s.WriteString("\n\n")
			filesStr := fmt.Sprintf("Files: %s", strings.Join(commit.Files, ", "))
			s.WriteString(wrapText(dimStyle.Render(filesStr), m.termWidth-2))
		}
		s.WriteString("\n\n")
		s.WriteString(m.form.View())

	case stateCommitting:
		s.WriteString(m.spinner.View())
		s.WriteString(" Committing...")

	case stateDone:
		if m.isSplit {
			s.WriteString(successStyle.Render(fmt.Sprintf("Created %d commits successfully!", len(m.commits))))
		} else {
			s.WriteString(successStyle.Render("Committed successfully!"))
		}
		s.WriteString("\n\n")
		for i, c := range m.commits {
			if m.completed[i] {
				s.WriteString(wrapText(dimStyle.Render(fmt.Sprintf("  %s", c.String())), m.termWidth-2))
				s.WriteString("\n")
			}
		}

	case stateError:
		s.WriteString(wrapText(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)), m.termWidth-2))
	}

	s.WriteString("\n")
	return s.String()
}

func (m *Model) generateCommitMessage() tea.Cmd {
	return func() tea.Msg {
		diff, err := m.repo.DiffAll(m.selected)
		if err != nil {
			return generateMsg{err: err}
		}

		result, err := m.aiClient.GenerateCommitMessage(
			context.Background(),
			m.selected,
			diff,
			m.cfg.Commit.Conventional,
			m.cfg.Commit.Types,
		)

		return generateMsg{result: result, err: err}
	}
}

func (m *Model) doCommit() tea.Cmd {
	return func() tea.Msg {
		commit := m.commits[m.currentIndex]
		files := commit.Files
		if len(files) == 0 {
			files = m.selected // fallback for single commit
		}

		if err := m.repo.Add(files); err != nil {
			return commitMsg{err: err}
		}

		if err := m.repo.Commit(commit.String()); err != nil {
			return commitMsg{err: err}
		}

		return commitMsg{}
	}
}
