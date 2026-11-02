package tools

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"vio/internal/workspace"
)

const (
	defaultShellTimeout = 2 * time.Minute
	maxShellOutput      = 64 << 10
)

type Shell struct {
	ws      workspace.FS
	timeout time.Duration
}

func NewShell(ws workspace.FS) *Shell {
	return &Shell{ws: ws, timeout: timeoutFromEnv()}
}

func (t *Shell) Name() string { return "shell" }

func (t *Shell) Description() string {
	return "Run a shell command from the repository root."
}

func (t *Shell) Schema() ToolSchema {
	return ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Command to run.",
				},
				"timeout_seconds": map[string]any{
					"type":        "number",
					"description": "Optional timeout in seconds.",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (t *Shell) Execute(ctx context.Context, in ToolInput) (ToolOutput, error) {
	command, ok := argString(in, "command")
	if !ok || strings.TrimSpace(command) == "" {
		return failure("shell: missing required argument %q", "command"), nil
	}

	timeout := t.timeout
	if seconds, ok := argNumber(in, "timeout_seconds"); ok && seconds > 0 {
		timeout = time.Duration(seconds * float64(time.Second))
	}

	runCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	name, args := shellCommand(command)
	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Dir = t.ws.Root()

	var output cappedBuffer
	output.max = maxShellOutput
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	text := strings.TrimRight(output.String(), "\r\n")
	if output.truncated {
		text += "\n[output truncated]"
	}

	if err == nil {
		if text == "" {
			text = "(no output)"
		}
		return ToolOutput{Output: text}, nil
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return failure("shell: timed out after %s", timeout), nil
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return failure("shell: canceled"), nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if text != "" {
			text += "\n"
		}
		return failure("%sexit code %d", text, exitErr.ExitCode()), nil
	}
	if text != "" {
		text += "\n"
	}
	return failure("shell: %v\n%s", err, text), nil
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-c", command}
}

func timeoutFromEnv() time.Duration {
	value := os.Getenv("AGENT_TOOL_TIMEOUT")
	if value == "" {
		return defaultShellTimeout
	}
	if d, err := time.ParseDuration(value); err == nil && d > 0 {
		return d
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultShellTimeout
}

type cappedBuffer struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.max - b.buf.Len(); room > 0 {
		if len(p) > room {
			b.buf.Write(p[:room])
			b.truncated = true
		} else {
			b.buf.Write(p)
		}
	} else if len(p) > 0 {
		b.truncated = true
	}
	return len(p), nil
}

func (b *cappedBuffer) String() string { return b.buf.String() }
