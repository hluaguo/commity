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
	stateInit       state = iota // first run setup
	stateFileSelect              // file selection
	stateGenerating
	stateConfirm
	stateCommitting
	stateDone
	stateSettings // settings page
	stateError
)

type Model struct {
	state         state
	previousState state // for returning from settings
	cfg           *config.Config
	repo          *git.Repository
	aiClient      *ai.Client
	isFirstRun    bool

	files    []git.FileStatus
	selected []string
	action   string // "commit", "regenerate", "cancel"

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

func New(cfg *config.Config, repo *git.Repository, aiClient *ai.Client, isFirstRun bool) (*Model, error) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		cfg:        cfg,
		repo:       repo,
		aiClient:   aiClient,
		spinner:    s,
		termWidth:  getTermWidth(),
		isFirstRun: isFirstRun,
	}

	// First run - show setup
	if isFirstRun {
		m.state = stateInit
		m.initFirstRunForm()
		return m, nil
	}

	// Normal run - need files
	files, err := repo.Status()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no changes to commit")
	}

	m.files = files
	m.state = stateFileSelect
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
	m.action = "commit"
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What do you want to do?").
				Options(
					huh.NewOption("Yes - commit", "commit"),
					huh.NewOption("Regenerate message", "regenerate"),
					huh.NewOption("Cancel", "cancel"),
				).
				Value(&m.action),
		),
	).WithTheme(huh.ThemeDracula())
}

func (m *Model) initSettingsForm() {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("API Base URL").
				Value(&m.cfg.AI.BaseURL),
			huh.NewInput().
				Title("API Key").
				Value(&m.cfg.AI.APIKey).
				EchoMode(huh.EchoModePassword),
			huh.NewInput().
				Title("Model").
				Value(&m.cfg.AI.Model),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use Conventional Commits?").
				Value(&m.cfg.Commit.Conventional),
			huh.NewText().
				Title("Custom Instructions").
				Description("Additional instructions for AI").
				Value(&m.cfg.AI.CustomInstructions).
				CharLimit(500),
		),
	).WithTheme(huh.ThemeDracula())
}

func (m *Model) initFirstRunForm() {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Welcome to Commity!").
				Description("Let's set up your configuration."),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("API Base URL").
				Description("OpenAI-compatible API endpoint").
				Value(&m.cfg.AI.BaseURL),
			huh.NewInput().
				Title("API Key").
				Value(&m.cfg.AI.APIKey).
				EchoMode(huh.EchoModePassword),
			huh.NewInput().
				Title("Model").
				Description("e.g., gpt-4o-mini, claude-3-sonnet").
				Value(&m.cfg.AI.Model),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use Conventional Commits?").
				Affirmative("Yes").
				Negative("No").
				Value(&m.cfg.Commit.Conventional),
			huh.NewText().
				Title("Custom Instructions (optional)").
				Description("Additional instructions for commit generation").
				Value(&m.cfg.AI.CustomInstructions).
				CharLimit(500),
		),
	).WithTheme(huh.ThemeDracula())
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.form.Init(), m.spinner.Tick)
}

type initCompleteMsg struct{}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state != stateInit && m.state != stateSettings {
				return m, tea.Quit
			}
		case "s", "S":
			// Open settings from file select
			if m.state == stateFileSelect {
				m.previousState = m.state
				m.state = stateSettings
				m.initSettingsForm()
				return m, m.form.Init()
			}
		}

	case initCompleteMsg:
		// After first run setup, reload and continue
		files, err := m.repo.Status()
		if err != nil {
			m.state = stateError
			m.err = err
			return m, nil
		}
		if len(files) == 0 {
			m.state = stateError
			m.err = fmt.Errorf("no changes to commit")
			return m, nil
		}
		m.files = files
		m.state = stateFileSelect
		m.initFileSelectForm()
		return m, m.form.Init()

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
	case stateInit:
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			// Save config and continue
			if err := m.cfg.Save(); err != nil {
				m.state = stateError
				m.err = fmt.Errorf("failed to save config: %w", err)
				return m, nil
			}
			// Reinitialize AI client with new config
			newClient, err := ai.New(&m.cfg.AI)
			if err != nil {
				m.state = stateError
				m.err = err
				return m, nil
			}
			m.aiClient = newClient
			return m, func() tea.Msg { return initCompleteMsg{} }
		}

		return m, cmd

	case stateSettings:
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			// Save config
			if err := m.cfg.Save(); err != nil {
				m.state = stateError
				m.err = fmt.Errorf("failed to save config: %w", err)
				return m, nil
			}
			// Reinitialize AI client with new config
			newClient, err := ai.New(&m.cfg.AI)
			if err != nil {
				m.state = stateError
				m.err = err
				return m, nil
			}
			m.aiClient = newClient
			// Return to previous state
			m.state = m.previousState
			m.initFileSelectForm()
			return m, m.form.Init()
		}

		return m, cmd

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
			switch m.action {
			case "commit":
				m.state = stateCommitting
				return m, m.doCommit()
			case "regenerate":
				m.state = stateGenerating
				return m, m.generateCommitMessage()
			case "cancel":
				return m, tea.Quit
			}
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
	case stateInit:
		s.WriteString(m.form.View())

	case stateSettings:
		s.WriteString(dimStyle.Render("Settings (saves on complete)"))
		s.WriteString("\n\n")
		s.WriteString(m.form.View())

	case stateFileSelect:
		s.WriteString(dimStyle.Render("Press 's' for settings"))
		s.WriteString("\n\n")
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
			m.cfg.AI.CustomInstructions,
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
