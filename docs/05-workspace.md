# 05 — Workspace (`internal/workspace`)

> The controlled filesystem boundary every tool uses (Issue 4, 10).

## Purpose

The agent must not manipulate arbitrary filesystem paths. `internal/workspace`
represents the repository/project being operated on and is the *only* path
through which file tools and repository-context gathering reach the disk.

Issue 10 proposed separate `internal/context`/`internal/git` packages; by
decision, repository context and file concerns live here (with git exposed as
tools in `internal/tools`). The prompt **context builder** itself lives in
`internal/agent` (see 01-agent-loop.md) and consumes this package.

## Package Layout

```text
internal/workspace/
├── workspace.go      Workspace type, root detection, init
├── files.go          path-safe read/write/delete/list/exists
├── file-search.go    recursive text search
└── errors.go         (to be added) typed workspace errors
```

## Workspace Type

```go
type Workspace struct {
    Root string
}
```

- Initialized from the current directory (or a `--cwd` flag).
- Detects the repository root (walks up to find `.git`) so tools rooted at the
  repo root work from subdirectories.
- `Root` is the single trust anchor for path safety and for shell working
  directory.

## Operations (`files.go`)

All file tools call these instead of raw `os` calls:

```go
func (w *Workspace) Read(path string) ([]byte, error)
func (w *Workspace) Write(path string, data []byte) error
func (w *Workspace) Delete(path string) error
func (w *Workspace) List(path string) ([]Entry, error)
func (w *Workspace) Exists(path string) (bool, error)
func (w *Workspace) Resolve(path string) (string, error) // safe absolute path
```

### Path Safety

- Normalize paths (clean `..`, resolve symlinks where practical) before
  touching the filesystem.
- Reject any path that escapes the workspace root after normalization:

```text
../../etc/passwd            → rejected
./main.go                   → ok
internal/foo.go             → ok
../project-file.go          → allowed only when it normalizes *inside* the root
```

- Return useful, typed errors (`workspace.ErrOutsideRoot`, etc.) so tools can
  surface them to the model.

## Repository Context

At startup the agent gathers lightweight repo metadata (Issue 9/10) without
dumping the whole repository into context:

```text
working directory
repository root
project structure (tree, bounded)
git branch
git status
```

Guards:

- Maximum repository tree size (depth/entries) to bound context.
- Ignore `.git`, obvious binary files, and entries ignored by `.gitignore`.
- Do not stream large files into context; read on demand via `read_file`.

## Search (`file-search.go`)

Basic text search (no embeddings / semantic search in v1):

- Recursive over the workspace.
- Ignores `.git` and obvious binary files.
- Respects ignore rules where cheap.
- Returns file paths and matching lines.
- Avoids loading very large files into memory unnecessarily (line streaming /
  bounded reads).

## Output Guards

Apply the shared limits from 01-agent-loop.md at this boundary where feasible:

- Maximum file output (per read).
- Maximum search-result size.
- Maximum listing depth.

## Testing (Issue 10)

- Path traversal rejected: `../`, absolute escapes, symlink escapes.
- Basic read/write/delete/list/exists round-trips against a temp dir tree.
- Root detection from nested dirs in a temp git repo.
- Search ignores `.git`, returns matching lines, handles binary files.
- All filesystem ops return useful errors.
