package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"vio/internal/app"
)

const (
	maxToolArgRunes    = 80
	maxToolOutputLines = 40
	maxToolOutputRunes = 8000
)

// formatToolCall renders a tool name and its arguments as one compact line.
func formatToolCall(tool app.ToolStarted) string {
	if len(tool.Args) == 0 {
		return tool.Name
	}

	keys := make([]string, 0, len(tool.Args))
	for key := range tool.Args {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+shortValue(tool.Args[key]))
	}
	return tool.Name + " " + strings.Join(parts, " ")
}

func shortValue(value any) string {
	var text string
	if s, ok := value.(string); ok {
		text = strconv.Quote(s)
	} else {
		text = fmt.Sprintf("%v", value)
	}

	runes := []rune(text)
	if len(runes) > maxToolArgRunes {
		return string(runes[:maxToolArgRunes-3]) + "..."
	}
	return text
}

// truncateOutput bounds tool output for display; the full result remains in the
// agent's conversation.
func truncateOutput(text string) string {
	truncated := false

	runes := []rune(text)
	if len(runes) > maxToolOutputRunes {
		runes = runes[:maxToolOutputRunes]
		truncated = true
	}

	lines := strings.Split(string(runes), "\n")
	if len(lines) > maxToolOutputLines {
		lines = lines[:maxToolOutputLines]
		truncated = true
	}

	out := strings.Join(lines, "\n")
	if truncated {
		out += "\n… output truncated"
	}
	return out
}

func indentBlock(text, prefix string) string {
	if text == "" {
		return prefix + "(no output)"
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
