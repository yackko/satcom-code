// tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss"

// You can centralize more lipgloss styles here if the TUI grows.
// For example:
var (
	FocusedStyle = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("69"))
	BlurredStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5733")) // Example error color
)

// Ensure styles defined in list_view.go are either moved here and exported,
// or kept local if only used by list_view.go.
// If moved here, they need to be Exported (e.g., DocStyle instead of docStyle).
