package tui

import (
	"image/color"
	"strings"

	gloss "charm.land/lipgloss/v2"
)

// version is shown in the application header.
const version = "v0.1.0-dev"

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

// Layout metrics: the ASCII art width and the minimum metadata width that must
// fit beside it before the header falls back to a stacked layout.
const (
	headerArtWidth = 22
	headerGap      = 3
	minMetaWidth   = 22
	inputHeight    = 3
)

var header = renderHeader()

func renderHeader() string {
	gradient := gloss.Blend2D(headerArtWidth, len(headerLines), 45, headerShades...)
	base := gloss.NewStyle().Bold(true)

	var b strings.Builder
	for y, line := range headerLines {
		for x, r := range []rune(line) {
			b.WriteString(base.Foreground(gradient[y*headerArtWidth+x]).Render(string(r)))
		}
		if y < len(headerLines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// gradientRule renders a horizontal gradient rule of the given width.
func gradientRule(width int) string {
	if width <= 0 {
		return ""
	}
	colors := gloss.Blend1D(width, headerShades...)
	var b strings.Builder
	for i := 0; i < width; i++ {
		b.WriteString(gloss.NewStyle().Foreground(colors[i]).Render("─"))
	}
	return b.String()
}

var (
	accent    = gloss.Color("#C084FC")
	errorClr  = gloss.Color("#F87171")
	toolColor = gloss.Color("#FBBF24")
	dimColor  = gloss.Color("#6B7280")
	textColor = gloss.Color("#E5E7EB")

	userLabelStyle = gloss.NewStyle().Bold(true).Foreground(accent)
	assistantStyle = gloss.NewStyle().Foreground(textColor)
	toolStyle      = gloss.NewStyle().Faint(true).Foreground(toolColor)
	errorStyle     = gloss.NewStyle().Foreground(errorClr)
	infoStyle      = gloss.NewStyle().Faint(true).Foreground(dimColor)

	userBoxStyle = gloss.NewStyle().
			Border(gloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1)

	agentBg = gloss.Color("#2E2757")

	agentReplyStyle = gloss.NewStyle().
			Background(agentBg).
			Padding(0, 1).
			PaddingTop(1)

	agentTextStyle = gloss.NewStyle().
			Foreground(textColor).
			Background(agentBg)

	agentThinkingStyle = gloss.NewStyle().
				Faint(true).
				Foreground(dimColor).
				Background(agentBg)

	violetTagStyle = gloss.NewStyle().
			Bold(true).
			Foreground(accent)

	titleStyle  = gloss.NewStyle().Bold(true).Foreground(accent)
	statusStyle = gloss.NewStyle().Faint(true).Foreground(dimColor)

	metaKeyStyle   = gloss.NewStyle().Faint(true).Foreground(dimColor)
	metaValueStyle = gloss.NewStyle().Foreground(textColor)

	idleDot = gloss.NewStyle().Foreground(dimColor).Render("●")
	busyDot = gloss.NewStyle().Foreground(accent).Render("●")

	inputBoxStyle = gloss.NewStyle().
			Border(gloss.RoundedBorder()).
			BorderForeground(dimColor).
			Padding(0, 1)

	inputFocusedBoxStyle = inputBoxStyle.Copy().BorderForeground(accent)
)

// welcomeView renders the initial screen shown before any messages exist.
func welcomeView(width int) string {
	lines := []string{
		"",
		infoStyle.Render("Build and ship production-grade software from the terminal."),
		"",
		infoStyle.Render("type a task and press enter to begin"),
		infoStyle.Render("ctrl+c to quit  ·  ctrl+l to clear"),
	}
	return gloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}
