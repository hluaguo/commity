package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func getTermWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80 // default
	}
	return width
}

// Styles holds all the styled components using a theme.
type Styles struct {
	Title   lipgloss.Style
	Success lipgloss.Style
	Error   lipgloss.Style
	Dim     lipgloss.Style
	Message lipgloss.Style
}

func NewStyles(theme *Theme) *Styles {
	return &Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary),
		Success: lipgloss.NewStyle().
			Foreground(theme.Success),
		Error: lipgloss.NewStyle().
			Foreground(theme.Error),
		Dim: lipgloss.NewStyle().
			Foreground(theme.Dim),
		Message: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(1, 2),
	}
}

func wrapText(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}
