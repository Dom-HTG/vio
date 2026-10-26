package workspace

import "errors"

// Typed errors returned by Workspace operations. Tools surface these to the
// model as structured failures rather than panicking.
var (
	// ErrOutsideRoot means the requested path would escape the workspace root.
	ErrOutsideRoot = errors.New("workspace: path escapes workspace root")
	// ErrInvalidPath means the path itself is unusable (empty root, root write, ...).
	ErrInvalidPath = errors.New("workspace: invalid path")
	// ErrIsDir means a file operation targeted a directory.
	ErrIsDir = errors.New("workspace: path is a directory")
	// ErrNotDir means a directory operation targeted a file.
	ErrNotDir = errors.New("workspace: path is not a directory")
	// ErrTooLarge means an operation exceeded one of the configured guards.
	ErrTooLarge = errors.New("workspace: result exceeds size limit")
)
