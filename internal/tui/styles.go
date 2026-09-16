package tui

import (
	"image/color"
	"strings"

	gloss "charm.land/lipgloss/v2"
)

var headerLines = []string{
	`██╗   ██╗██╗ ██████╗ `,
	`██║   ██║██║██╔═══██╗`,
	`██║   ██║██║██║   ██║`,
	`╚██╗ ██╔╝██║██║   ██║`,
	` ╚████╔╝ ██║╚██████╔╝`,
	`  ╚═══╝  ╚═╝ ╚═════╝ `,
}

var headerShades = []color.Color{
	gloss.Color("#F3E8FF"),
	gloss.Color("#C084FC"),
	gloss.Color("#9333EA"),
	gloss.Color("#581C87"),
}

var header = renderHeader()

func renderHeader() string {
	width := 0
	for _, line := range headerLines {
		if n := len([]rune(line)); n > width {
			width = n
		}
	}
	height := len(headerLines)

	gradient := gloss.Blend2D(width, height, 45, headerShades...)
	base := gloss.NewStyle().Bold(true)

	var b strings.Builder
	for y, line := range headerLines {
		for x, r := range []rune(line) {
			b.WriteString(base.Foreground(gradient[y*width+x]).Render(string(r)))
		}
		if y < height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
