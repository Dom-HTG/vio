// Package workspace is the controlled filesystem boundary used by file tools.
// Every path is resolved against a single root and may never escape it.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultMaxReadBytes       int64 = 2 << 20
	DefaultMaxListEntries           = 2000
	DefaultMaxSearchResults         = 200
	DefaultMaxSearchFileBytes int64 = 2 << 20
)

// Workspace identifies the root paths are resolved against.
type Workspace interface {
	Root() string
	Resolve(path string) (string, error)
}

// Filesystem is the path-safe file operation set exposed to tools.
type Filesystem interface {
	Read(path string) ([]byte, error)
	Write(path string, data []byte) error
	Delete(path string) error
	List(path string) ([]Entry, error)
	Exists(path string) (bool, error)
	Search(query string, opts SearchOptions) ([]Match, error)
}

// FS is the complete boundary used by filesystem-related tools.
type FS interface {
	Workspace
	Filesystem
}

type Local struct {
	root string

	MaxReadBytes       int64
	MaxListEntries     int
	MaxSearchResults   int
	MaxSearchFileBytes int64
}

var _ FS = (*Local)(nil)

func New(root string) (*Local, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("workspace: resolve root %q: %w", root, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace: stat root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: root is not a directory: %s", ErrInvalidPath, abs)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace: resolve root %q: %w", root, err)
	}

	return &Local{
		root:               DetectRoot(resolved),
		MaxReadBytes:       DefaultMaxReadBytes,
		MaxListEntries:     DefaultMaxListEntries,
		MaxSearchResults:   DefaultMaxSearchResults,
		MaxSearchFileBytes: DefaultMaxSearchFileBytes,
	}, nil
}

func NewFromCWD() (*Local, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("workspace: getwd: %w", err)
	}
	return New(dir)
}

func (l *Local) Root() string { return l.root }

// DetectRoot returns the nearest ancestor of start containing a .git entry, or
// start itself when none is found.
func DetectRoot(start string) string {
	abs, err := filepath.Abs(start)
	if err != nil {
		abs = filepath.Clean(start)
	}

	dir := abs
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return abs
		}
		dir = parent
	}
}
