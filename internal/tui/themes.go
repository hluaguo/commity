package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color scheme for the TUI.
type Theme struct {
	Name      string
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Error     lipgloss.Color
	Dim       lipgloss.Color
	Border    lipgloss.Color
	HuhTheme  *huh.Theme
}

var themes = map[string]*Theme{
	"tokyonight": {
		Name:      "tokyonight",
		Primary:   lipgloss.Color("#7aa2f7"),
		Secondary: lipgloss.Color("#bb9af7"),
		Success:   lipgloss.Color("#9ece6a"),
		Error:     lipgloss.Color("#f7768e"),
		Dim:       lipgloss.Color("#565f89"),
		Border:    lipgloss.Color("#3b4261"),
	},
	"dracula": {
		Name:      "dracula",
		Primary:   lipgloss.Color("#bd93f9"),
		Secondary: lipgloss.Color("#ff79c6"),
		Success:   lipgloss.Color("#50fa7b"),
		Error:     lipgloss.Color("#ff5555"),
		Dim:       lipgloss.Color("#6272a4"),
		Border:    lipgloss.Color("#44475a"),
	},
	"catppuccin": {
		Name:      "catppuccin",
		Primary:   lipgloss.Color("#cba6f7"),
		Secondary: lipgloss.Color("#f5c2e7"),
		Success:   lipgloss.Color("#a6e3a1"),
		Error:     lipgloss.Color("#f38ba8"),
		Dim:       lipgloss.Color("#6c7086"),
		Border:    lipgloss.Color("#45475a"),
	},
	"nord": {
		Name:      "nord",
		Primary:   lipgloss.Color("#88c0d0"),
		Secondary: lipgloss.Color("#81a1c1"),
		Success:   lipgloss.Color("#a3be8c"),
		Error:     lipgloss.Color("#bf616a"),
		Dim:       lipgloss.Color("#4c566a"),
		Border:    lipgloss.Color("#3b4252"),
	},
}

func GetTheme(name string) *Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["tokyonight"]
}

func GetThemeNames() []string {
	return []string{"tokyonight", "dracula", "catppuccin", "nord"}
}

func (t *Theme) GetHuhTheme() *huh.Theme {
	if t.HuhTheme != nil {
		return t.HuhTheme
	}

	theme := huh.ThemeBase()

	// Customize the huh theme with our colors
	theme.Focused.Title = theme.Focused.Title.Foreground(t.Primary)
	theme.Focused.Description = theme.Focused.Description.Foreground(t.Dim)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(t.Success)
	theme.Focused.UnselectedOption = theme.Focused.UnselectedOption.Foreground(t.Secondary)
	theme.Focused.Base = theme.Focused.Base.BorderForeground(t.Border)

	// Make help text more visible
	theme.Help.ShortKey = theme.Help.ShortKey.Foreground(t.Primary).Bold(true)
	theme.Help.ShortDesc = theme.Help.ShortDesc.Foreground(t.Secondary)
	theme.Help.ShortSeparator = theme.Help.ShortSeparator.Foreground(t.Dim)
	theme.Help.FullKey = theme.Help.FullKey.Foreground(t.Primary).Bold(true)
	theme.Help.FullDesc = theme.Help.FullDesc.Foreground(t.Secondary)
	theme.Help.FullSeparator = theme.Help.FullSeparator.Foreground(t.Dim)

	theme.Blurred.Title = theme.Blurred.Title.Foreground(t.Dim)
	theme.Blurred.Description = theme.Blurred.Description.Foreground(t.Dim)

	t.HuhTheme = theme
	return theme
}
