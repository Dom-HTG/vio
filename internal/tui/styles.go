package tui

import(
	gloss "github.com/charmbracelet/lipgloss"
)

headerStyle := gloss.NewStyle().Bold(true).Foreground(gloss.Color("205"))
header := headerStyle.Render("Vio")