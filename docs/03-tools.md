# 03 — Tools (`internal/tools`)

> The model's hands: a small, deterministic set of tools for inspecting and
> modifying a repository (Issues 2, 5, 10).

## Purpose

Tools are the only way the model interacts with the world beyond talking. All
filesystem access goes through `internal/workspace`; all process execution goes
through the `shell` tool. The agent loop never executes a tool by name string
itself — it asks the `Registry`.

## Package Layout

The scaffold already carries these files (issue #5 proposed `read.go`/`write.go`
etc.; the scaffold's prefixed names are canonical):

```text
internal/tools/
├── tool.go              Tool contract
├── tool-registry.go     Registry
├── read_file.go         read_file
├── write_file.go        write_file
├── edit_file.go         edit_file
├── list_directory.go    list_directory
├── search.go            search
├── shell.go             shell
├── git_status.go        git_status
└── git_diff.go          git_diff
```

## Tool Contract

```go
type Tool interface {
    Name() string
    Description() string
    Schema() ToolSchema        // JSON Schema for the model
    Execute(ctx context.Context, input ToolInput) ToolResult
}
```

Concrete shapes:

```go
type ToolSchema struct {
    Name        string
    Description string
    Parameters  map[string]any // JSON Schema object
}

type ToolInput struct {
    Arguments map[string]any // validated against Schema
}

type ToolResult struct {
    Output  string // human/model-readable output
    IsError bool   // structured failure, NOT a panic
    Data    map[string]any // optional structured payload for tool_view.go
}
```

A tool failure is represented as a structured `ToolResult{IsError: true}` so the
agent loop can hand it back to the model for recovery (Issue 6). Tools never
panic on expected failures.

Every tool independently unit-testable: no hidden global state, dependencies
(workspace, cwd, environment) injected via an execution context.

## Registry (`tool-registry.go`)

```go
type Registry struct {
    tools map[string]Tool
}

func (r *Registry) Register(tool Tool) error      // error on duplicate name
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) Definitions() []ToolDefinition // for the provider
func (r *Registry) Names() []string
```

The registry is the single source of tool definitions handed to the provider and
the allow/deny gate for tool execution.

## Built-in Tools

### `read_file`

```text
path
```

Returns file contents, rooted in the workspace, with the size-capped output
limits applied.

### `write_file`

```text
path
content
```

Creates or replaces a file. `path` must stay within the workspace.

### `edit_file`

```text
path
old_text
new_text
```

Deterministic single-replacement model. **Fails** if `old_text` cannot be found
— prefer failing over silently modifying the wrong location (Issue 5). No
regex/fuzzy semantics in v1.

### `list_directory`

```text
path
```

Lists entries (files/dirs) under a workspace path. Used by the agent to get a
project overview without dumping whole trees into context.

### `search`

```text
query
path?
```

Text search over the workspace: recursive, ignores `.git` and binary files,
returns file paths + matching lines, bounded to avoid loading huge files.

### `shell`

```text
command
```

Executes a shell command from the workspace root. Requirements (Issue 5):

- Runs from workspace root.
- Captures stdout and stderr.
- Returns the exit code.
- Respects context cancellation (`exec.CommandContext`).
- Configurable timeout (`AGENT_TOOL_TIMEOUT`).
- Returns complete command failures (including exit code) to the agent.

Security stance for v1: no full sandbox, but shell execution is clearly
isolated, commands run in the workspace working directory, and the result
includes the exit code so the model can react.

### `git_status`

```text
(no required args)
```

Returns `git status`-style summary (branch, staged/unstaged changes) for the
workspace. Provides the "am I clean / what changed" signal.

### `git_diff`

```text
path?          // optional: restrict diff to a path
staged?        // optional: diff staged instead of unstaged
```

Returns the diff. Gives the agent the "what did I change" and "what will I
change" signal.

Git tools run through a thin wrapper (or the shell tool) — the working directory
is the workspace root. Issue 10's separate `internal/git` package was folded
here by decision; keep git logic in `git_status.go`/`git_diff.go`.

## Permissions / Approval (Issue 8, v1 scope)

Tools are classified read-only (`view` group: `read_file`, `list_directory`,
`search`, `git_status`, `git_diff`) vs. mutating (`edit` group: `write_file`,
`edit_file`) vs. arbitrary (`shell`).

- By default, mutating tools and `shell` ask the user before running
  (emitting `EventPermissionNeeded`, see 01-agent-loop.md).
- The TUI surfaces an approve/deny prompt (see 06-tui.md).
- A `--yolo`-style escape hatch (approve everything) is a candidate for v1 but
  defaults to prompting.

Allow/deny lists can restrict which tools the model can see/use.

## Output Limits

Large tool output is truncated for the UI while the complete result is preserved
internally for the model loop (Issue 8). Enforce the max tool-result size in the
tool boundary (see `internal/workspace` guards and agent context limits).

## Testing (Issue 10)

- Registry register/get/definitions + duplicate rejection.
- Each tool against a temp workspace dir (happy path + failure path).
- `edit_file` fails when `old_text` absent.
- `shell` returns stdout/stderr/exit code and is cancelled by context.
- Path traversal attempts are rejected (see 05-workspace.md).
