package main

import (
	"log"

	tea "charm.land/bubbletea/v2"

	"vio/internal/tui"
)

func main() {
	s, err := newStack()
	if err != nil {
		log.Fatal(err)
	}

	program := tea.NewProgram(tui.NewModel(s.Agent))
	if _, err := program.Run(); err != nil {
		log.Fatal(err)
	}
}
