package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rickykimani/cubiceos/internal/tui"
)

func main() {
	if _, err := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
