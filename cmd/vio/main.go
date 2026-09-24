package main

import (
	"log"

	tea "charm.land/bubbletea/v2"

	"vio/internal/tui"
)

func main() {
    newModel := tui.NewModel()
    program := tea.NewProgram(newModel)
    if _, err := program.Run(); err != nil {
        log.Fatal(err)
    }
}