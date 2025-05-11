// tui/list_view.go
package tui

import (
	"fmt"
	// Ensure this import path correctly points to your types package
	// based on your go.mod module name.
	"github.com/yackko/satcom-code/types" 

	tea "github.com/charmbracelet/bubbletea"
)

// ListModel is a minimal TUI model for testing.
type ListModel struct {
	Satellites []types.Satellite // To ensure types package is resolving
	Message    string
}

// NewListModel creates a new minimal model.
func NewListModel(sats []types.Satellite) ListModel {
	return ListModel{
		Satellites: sats,
		Message:    "Minimal TUI Model Initialized. Press 'q' to quit.",
	}
}

// Init is a required method for tea.Model.
func (m ListModel) Init() tea.Cmd {
	// This is just for a quick check if Init runs, can be removed later.
	// fmt.Println("Minimal ListModel Init()") 
	return nil
}

// Update is a required method for tea.Model.
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View is a required method for tea.Model.
func (m ListModel) View() string {
	var s string
	if len(m.Satellites) > 0 {
		s = fmt.Sprintf("Minimal TUI: %d satellites loaded. First: %s\n", len(m.Satellites), m.Satellites[0].Name)
	} else {
		s = "Minimal TUI: No satellites loaded.\n"
	}
	return s + m.Message + "\n"
}
