package workspace

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Entry struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// Resolve returns a safe absolute path inside the workspace. Symlinks are
// resolved, including the nearest existing ancestor of paths that do not exist
// yet, so new files cannot escape through a symlinked parent.
func (w *Local) Resolve(path string) (string, error) {
	if w == nil || w.root == "" {
		return "", fmt.Errorf("%w: workspace has no root", ErrInvalidPath)
	}
	if strings.TrimSpace(path) == "" {
		path = "."
	}

	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Join(w.root, path)
	}

	resolved, err := resolveExisting(abs)
	if err != nil {
		return "", fmt.Errorf("workspace: resolve %q: %w", path, err)
	}

	rel, err := filepath.Rel(w.root, resolved)
	if err != nil || filepath.IsAbs(rel) || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s", ErrOutsideRoot, path)
	}
	return resolved, nil
}

func resolveExisting(path string) (string, error) {
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	dir := path
	var missing []string
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		missing = append(missing, filepath.Base(dir))
		dir = parent

		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
}

func (w *Local) Read(path string) ([]byte, error) {
	abs, err := w.Resolve(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace: read %q: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("workspace: read %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrIsDir, path)
	}

	limit := w.MaxReadBytes
	if limit <= 0 {
		limit = DefaultMaxReadBytes
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, fmt.Errorf("workspace: read %q: %w", path, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrTooLarge, path, limit)
	}
	return data, nil
}

func (w *Local) Write(path string, data []byte) error {
	abs, err := w.Resolve(path)
	if err != nil {
		return err
	}
	if abs == w.root {
		return fmt.Errorf("%w: cannot write to workspace root", ErrIsDir)
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("workspace: create parent for %q: %w", path, err)
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return fmt.Errorf("workspace: write %q: %w", path, err)
	}
	return nil
}

func (w *Local) Delete(path string) error {
	abs, err := w.Resolve(path)
	if err != nil {
		return err
	}
	if abs == w.root {
		return fmt.Errorf("%w: cannot delete workspace root", ErrInvalidPath)
	}
	if err := os.Remove(abs); err != nil {
		return fmt.Errorf("workspace: delete %q: %w", path, err)
	}
	return nil
}

func (w *Local) List(path string) ([]Entry, error) {
	abs, err := w.Resolve(path)
	if err != nil {
		return nil, err
	}

	dirEntries, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace: list %q: %w", path, err)
	}

	limit := w.MaxListEntries
	if limit <= 0 {
		limit = DefaultMaxListEntries
	}
	if len(dirEntries) > limit {
		return nil, fmt.Errorf("%w: %s has %d entries (limit %d)", ErrTooLarge, path, len(dirEntries), limit)
	}

	out := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		full := filepath.Join(abs, de.Name())
		rel, err := filepath.Rel(w.root, full)
		if err != nil {
			rel = de.Name()
		}
		entry := Entry{
			Name:  de.Name(),
			Path:  filepath.ToSlash(rel),
			IsDir: de.IsDir(),
		}
		if info, err := de.Info(); err == nil {
			entry.Size = info.Size()
		}
		out = append(out, entry)
	}
	return out, nil
}

func (w *Local) Exists(path string) (bool, error) {
	abs, err := w.Resolve(path)
	if err != nil {
		return false, err
	}

	_, err = os.Lstat(abs)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("workspace: exists %q: %w", path, err)
	}
}
