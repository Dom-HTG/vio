# vio Roadmap — Path to a Polished Coding Agent

This is the working checklist for finishing vio. Items are grouped by priority:
**P0** blocks a useful coding agent, **P1** makes it feel polished, **P2** hardens it
for release.

GitHub issue references from `Dom-HTG/vio` are included where relevant.

## Current state

Working today:

- CLI + Bubble Tea shell with streaming transcript, tool rows, status states, and Ctrl+C cancellation.
- Provider layer: OpenAI-compatible HTTP provider with non-streaming and SSE streaming, tool-call accumulation, typed errors, and a DeepSeek preset.
- Workspace: path-safe filesystem abstraction with root detection, read/write/delete/list/exists/search, and size guards.
- Tools: `read_file`, `write_file`, `edit_file`, `search`, `shell`, plus the tool registry.
- Agent loop: multi-step tool execution, event emission, iteration limit, and cancellation.
- TUI ↔ agent wiring: `AgentRunner` events drive the transcript; tool calls/results and errors render; Ctrl+C cancels a turn.
- Stack wiring in `cmd/vio`: workspace → provider → registry → session → agent, with `.env` loading and fail-fast API-key validation.
- Tests for workspace, tools, registry, model provider, agent loop, and TUI event handling.

Known gaps are listed below.

---

## P0 — Core agent quality

- [ ] **System prompt + context builder** (`docs/01-agent-loop.md`, Issues #6, #10). Add instructions, working directory, repo root, and tool-usage guidance. Without this, behavior is generic.
- [ ] **Repository metadata in context**: branch, `git status`, and a bounded project tree. Keep it small and deterministic.
- [ ] **Context budgeting**: trim conversation history, cap tool-result size at the source (not just in the UI), cap file reads, and avoid dumping huge outputs into the model.
- [ ] **Git awareness**: implement `git_status` and `git_diff` (optionally `git_log`), replacing the empty stubs.
- [ ] **Complete the tool set**: implement `list_directory`; decide whether `delete`/`move` belong in v1.
- [ ] **Permission/approval flow**: classify read-only vs. mutating vs. `shell`, prompt before writes/shell, and wire the TUI `Waiting` state.
- [ ] **Loop hardening** (Issue #6): `MaxToolCalls`, per-tool timeouts, typed errors for model/tool/invalid-call/iteration/cancel, and detection of repeated identical tool calls.
- [ ] **Provider resilience**: retry/backoff for 429/5xx and network errors; handle malformed tool calls and partial streams gracefully.
- [ ] **Session persistence** (Issue #10): implement `state.Store` on disk, plus `AddMessage`, `Clear`, and session resume.
- [ ] **Fix keyless local endpoints**: the unconditional `MODEL_API_KEY` check blocks Ollama/llama.cpp; make the key optional for explicit local base URLs.
- [ ] **End-to-end tests + CI**: fake-provider agent test through the TUI event bridge, workspace traversal tests, git tool tests, and `go test -race ./...` in CI.

## P1 — UX and workflow polish

- [ ] **Tool view** (`internal/tui/tool_view.go`): dedicated tool blocks with name, args, streaming output, duration, and truncation indicators.
- [ ] **Streaming tool output**: emit `ToolOutput` incrementally for `shell` instead of only the final result.
- [ ] **Permission prompt UI**: approve/deny for mutating tools, with a `--yolo` escape hatch.
- [ ] **Diff previews**: show a diff before/after `edit_file`/`write_file`.
- [ ] **Markdown/code rendering**: syntax highlighting and wrapping for assistant messages and tool output.
- [ ] **Multiline input** and input history (`Up`/`Down`).
- [ ] **Session commands**: `/clear`, `/model`, `/help`, `/quit`, and a session picker.
- [ ] **Status detail**: elapsed time, current tool, token/cost usage.
- [ ] **CLI surface**: wire the existing cobra command, add `--cwd`, `--debug`, `--yolo`, `--version`, and a non-interactive `-p/--prompt` mode.
- [ ] **Debug logging**: structured logs to a file, never interleaved with the TUI.
- [ ] **Friendly error UX**: clear auth/network/timeout messages with retry guidance instead of raw errors.

## P2 — Release and long-term quality

- [ ] **README + docs**: installation, configuration, supported providers, tools, security stance, architecture, limitations.
- [ ] **Packaging**: `go build -o vio ./cmd/vio`, release binaries, version injection, install instructions.
- [ ] **Secret safety**: never read `.env`/credential files into model context; optionally redact secrets in tool output.
- [ ] **Search quality**: respect `.gitignore`, ignore binary/large files consistently, optionally add regex search.
- [ ] **Provider presets**: DeepSeek, OpenAI, OpenRouter, Ollama, LM Studio with sensible defaults.
- [ ] **Theming/keybindings**: configurable styles and keys; accessibility-friendly defaults.
- [ ] **Performance**: faster large-repo search/listing, context caching, avoid re-reading unchanged files.

---

## Suggested order

1. System prompt + context builder
2. Git tools + repository metadata
3. Permissions for writes and shell
4. Session persistence
5. Loop hardening + retries
6. TUI tool view, diffs, and status polish
7. CLI, configuration, docs, and release

The first three are what make it feel like a coding agent rather than a chat loop
with tools.
