package tui

import (
	"strings"
)

// renderMessage renders a single transcript entry.
//
// User prompts render as purple rounded containers; agent replies render as a
// purple-tinted panel with a "vio" label. Tool calls, errors, and info lines
// stay inline.
func (m *model) renderMessage(msg Message) string {
	width := m.viewport.Width()
	if width < 4 {
		width = 4
	}

	switch msg.Role {
	case RoleUser:
		body := userLabelStyle.Render("❯ ") + assistantStyle.Render(msg.Content)
		return userBoxStyle.Width(width).Render(body)

	case RoleAssistant:
		content := msg.Content
		if strings.TrimSpace(content) == "" {
			content = agentThinkingStyle.Render("thinking…")
		} else {
			content = agentTextStyle.Render(content)
		}
		body := agentReplyStyle.Width(width).Render(content)
		return violetTagStyle.Render(" vio ") + "\n" + body

	case RoleTool:
		return toolStyle.Render("  ● " + msg.Content)

	case RoleError:
		return errorStyle.Render("  ✗ " + msg.Content)

	default:
		return infoStyle.Render("  " + msg.Content)
	}
}

// transcript builds the full scrollable conversation content.
func (m *model) transcript() string {
	if len(m.messages) == 0 {
		return welcomeView(m.viewport.Width())
	}
	parts := make([]string, 0, len(m.messages))
	for _, msg := range m.messages {
		parts = append(parts, m.renderMessage(msg))
	}
	return strings.Join(parts, "\n\n")
}

// refreshViewport re-renders the transcript into the viewport and scrolls to
// the latest message.
func (m *model) refreshViewport() {
	m.viewport.SetContent(m.transcript())
	m.viewport.GotoBottom()
}
