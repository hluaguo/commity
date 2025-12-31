package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
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
	stateEdit // editing commit message
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
	feedback string // user feedback for regeneration

	// Commit handling (supports split commits)
	commits      []ai.CommitMessage
	currentIndex int
	isSplit      bool
	completed    []bool // track which commits are done

	form        *huh.Form
	confirmForm *ConfirmModel
	editArea    textarea.Model
	spinner     spinner.Model
	err         error
	termWidth   int

	// Theming
	theme  *Theme
	styles *Styles
}

type generateMsg struct {
	result *ai.GenerateResult
	err    error
}

type commitMsg struct {
	err error
}

func New(cfg *config.Config, repo *git.Repository, aiClient *ai.Client, isFirstRun bool) (*Model, error) {
	theme := GetTheme(cfg.UI.Theme)
	styles := NewStyles(theme)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	m := &Model{
		cfg:        cfg,
		repo:       repo,
		aiClient:   aiClient,
		spinner:    s,
		termWidth:  getTermWidth(),
		isFirstRun: isFirstRun,
		theme:      theme,
		styles:     styles,
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
				Title("Select files to commit").
				Options(options...).
				Value(&m.selected),
		),
	).WithTheme(m.theme.GetHuhTheme()).WithShowHelp(false)
}

func (m *Model) getThemeOptions() []huh.Option[string] {
	var options []huh.Option[string]
	for _, name := range GetThemeNames() {
		t := GetTheme(name)
		// Color block using the theme's primary color
		colorBlock := lipgloss.NewStyle().
			Background(t.Primary).
			Render("  ")
		label := fmt.Sprintf("%s %s", colorBlock, name)
		options = append(options, huh.NewOption(label, name))
	}
	return options
}

func (m *Model) initConfirmForm() {
	m.confirmForm = NewConfirmModel(m.theme)
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
			huh.NewSelect[string]().
				Title("Theme").
				Options(m.getThemeOptions()...).
				Value(&m.cfg.UI.Theme),
		),
		huh.NewGroup(
			huh.NewText().
				Title("Custom Instructions").
				Description("Additional instructions for AI").
				Value(&m.cfg.AI.CustomInstructions).
				CharLimit(1000),
		),
	).WithTheme(m.theme.GetHuhTheme()).WithShowHelp(false)
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
			huh.NewSelect[string]().
				Title("Theme").
				Options(m.getThemeOptions()...).
				Value(&m.cfg.UI.Theme),
		),
		huh.NewGroup(
			huh.NewText().
				Title("Custom Instructions (optional)").
				Description("Additional instructions for commit generation").
				Value(&m.cfg.AI.CustomInstructions).
				CharLimit(500),
		),
	).WithTheme(m.theme.GetHuhTheme()).WithShowHelp(false)
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
		return m, m.confirmForm.Init()

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
			return m, m.confirmForm.Init()
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
			// Refresh theme
			m.theme = GetTheme(m.cfg.UI.Theme)
			m.styles = NewStyles(m.theme)
			m.spinner.Style = lipgloss.NewStyle().Foreground(m.theme.Primary)
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
			// Refresh theme
			m.theme = GetTheme(m.cfg.UI.Theme)
			m.styles = NewStyles(m.theme)
			m.spinner.Style = lipgloss.NewStyle().Foreground(m.theme.Primary)
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
		var cmd tea.Cmd
		m.confirmForm, cmd = m.confirmForm.Update(msg)

		if m.confirmForm.Submitted() {
			m.feedback = m.confirmForm.Feedback()
			switch m.confirmForm.Action() {
			case "commit":
				m.state = stateCommitting
				return m, m.doCommit()
			case "cancel":
				return m, tea.Quit
			case "regenerate":
				m.state = stateGenerating
				return m, m.generateCommitMessage()
			case "edit":
				m.state = stateEdit
				// Initialize textarea with current message
				ta := textarea.New()
				ta.SetValue(m.commits[m.currentIndex].String())
				ta.Focus()
				ta.SetWidth(m.termWidth - 4)
				ta.SetHeight(10)
				m.editArea = ta
				return m, textarea.Blink
			}
		}

		return m, cmd

	case stateEdit:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				// Cancel edit, go back to confirm
				m.state = stateConfirm
				m.initConfirmForm()
				return m, m.confirmForm.Init()
			case "ctrl+s":
				// Save edit
				newMsg := m.editArea.Value()
				// Update the commit message (just subject for simplicity)
				m.commits[m.currentIndex] = ai.CommitMessage{
					Subject: newMsg,
					Files:   m.commits[m.currentIndex].Files,
				}
				m.state = stateConfirm
				m.initConfirmForm()
				return m, m.confirmForm.Init()
			}
		}
		var cmd tea.Cmd
		m.editArea, cmd = m.editArea.Update(msg)
		return m, cmd

	case stateGenerating, stateCommitting:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) renderKeyHint(key, desc string) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(m.theme.Secondary)
	return fmt.Sprintf("%s %s", keyStyle.Render(key), descStyle.Render(desc))
}

func (m *Model) View() string {
	var s strings.Builder

	s.WriteString(m.styles.Title.Render("commity"))
	s.WriteString("\n\n")

	switch m.state {
	case stateInit:
		s.WriteString(m.form.View())
		s.WriteString("\n")
		s.WriteString(m.renderKeyHint("[↑↓]", "navigate") + "  " +
			m.renderKeyHint("[enter]", "next"))

	case stateSettings:
		s.WriteString(m.styles.Dim.Render("Settings (saves on complete)"))
		s.WriteString("\n\n")
		s.WriteString(m.form.View())
		s.WriteString("\n")
		s.WriteString(m.renderKeyHint("[↑↓]", "navigate") + "  " +
			m.renderKeyHint("[enter]", "next"))

	case stateFileSelect:
		s.WriteString(m.form.View())
		s.WriteString("\n")
		s.WriteString(m.renderKeyHint("[space]", "toggle") + "  " +
			m.renderKeyHint("[↑↓]", "navigate") + "  " +
			m.renderKeyHint("[enter]", "submit") + "  " +
			m.renderKeyHint("[s]", "settings") + "  " +
			m.renderKeyHint("[q]", "quit"))

	case stateGenerating:
		s.WriteString(m.spinner.View())
		s.WriteString(" Generating commit message...")

	case stateConfirm:
		// Show branch
		branch := m.repo.Branch()
		branchStyle := lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true)
		s.WriteString(fmt.Sprintf("Branch: %s\n\n", branchStyle.Render(branch)))

		// Get files for this commit
		commit := m.commits[m.currentIndex]
		commitFiles := commit.Files
		if len(commitFiles) == 0 {
			commitFiles = m.selected
		}

		// Show files with status
		s.WriteString(m.styles.Dim.Render("Files:"))
		s.WriteString("\n")
		statusStyle := lipgloss.NewStyle().Foreground(m.theme.Success)
		for _, path := range commitFiles {
			// Find status for this file
			status := "M"
			for _, f := range m.files {
				if f.Path == path {
					status = f.Status
					break
				}
			}
			s.WriteString(fmt.Sprintf("  %s %s\n", statusStyle.Render(status), path))
		}

		// Show diff stats
		added, removed := m.repo.DiffStats(commitFiles)
		statsStyle := lipgloss.NewStyle().Foreground(m.theme.Dim)
		addStyle := lipgloss.NewStyle().Foreground(m.theme.Success)
		removeStyle := lipgloss.NewStyle().Foreground(m.theme.Error)
		s.WriteString(statsStyle.Render(fmt.Sprintf("\n%d files, ", len(commitFiles))))
		s.WriteString(addStyle.Render(fmt.Sprintf("+%d", added)))
		s.WriteString(statsStyle.Render(" "))
		s.WriteString(removeStyle.Render(fmt.Sprintf("-%d", removed)))
		s.WriteString("\n\n")

		if m.isSplit {
			s.WriteString(fmt.Sprintf("Commit %d of %d:\n\n", m.currentIndex+1, len(m.commits)))
		} else {
			s.WriteString("Commit message:\n\n")
		}
		// Wrap message box to terminal width (minus border padding)
		msgWidth := m.termWidth - 8
		if msgWidth < 40 {
			msgWidth = 40
		}
		s.WriteString(m.styles.Message.Width(msgWidth).Render(commit.String()))
		s.WriteString("\n\n")
		s.WriteString(m.confirmForm.View())
		s.WriteString("\n\n")
		s.WriteString(m.renderKeyHint("[↑↓]", "navigate") + "  " +
			m.renderKeyHint("[enter]", "select") + "  " +
			m.renderKeyHint("[e]", "edit"))

	case stateEdit:
		s.WriteString(m.styles.Dim.Render("Edit commit message:"))
		s.WriteString("\n\n")
		s.WriteString(m.editArea.View())
		s.WriteString("\n\n")
		s.WriteString(m.renderKeyHint("[ctrl+s]", "save") + "  " + m.renderKeyHint("[esc]", "cancel"))

	case stateCommitting:
		s.WriteString(m.spinner.View())
		s.WriteString(" Committing...")

	case stateDone:
		if m.isSplit {
			s.WriteString(m.styles.Success.Render(fmt.Sprintf("Created %d commits successfully!", len(m.commits))))
		} else {
			s.WriteString(m.styles.Success.Render("Committed successfully!"))
		}
		s.WriteString("\n\n")
		for i, c := range m.commits {
			if m.completed[i] {
				s.WriteString(wrapText(m.styles.Dim.Render(fmt.Sprintf("  %s", c.String())), m.termWidth-2))
				s.WriteString("\n")
			}
		}

	case stateError:
		s.WriteString(wrapText(m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)), m.termWidth-2))
	}

	s.WriteString("\n")
	return s.String()
}

func (m *Model) generateCommitMessage() tea.Cmd {
	// Capture previous message for regeneration context
	var previousMsg string
	if len(m.commits) > 0 && m.currentIndex < len(m.commits) {
		previousMsg = m.commits[m.currentIndex].String()
	}
	feedback := m.feedback

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
			previousMsg,
			feedback,
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
