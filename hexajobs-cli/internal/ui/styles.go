package ui

import (
	"github.com/charmbracelet/lipgloss"
	"hexajobs.dev/hexajobs-cli/internal/ui/views"
)

// NewStyles keeps all visual decisions in the UI layer. No backend package
// imports these styles or depends on a terminal's color capabilities.
func NewStyles() views.Styles {
	return views.Styles{
		Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00FFFF")),
		Text:     lipgloss.NewStyle().Foreground(lipgloss.Color("#ECECEC")),
		Muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("#999999")),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#080A0C")).Background(lipgloss.Color("#00FFFF")),
		Green:    lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")),
		Yellow:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB800")),
		Red:      lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")),
		Border:   lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#444A50")),
	}
}
