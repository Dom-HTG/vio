package workspace

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxSearchLineBytes = 1 << 20
	binarySniffBytes   = 8000
)

type Match struct {
	Path string
	Line int
	Text string
}

type SearchOptions struct {
	Path          string
	CaseSensitive bool
	MaxResults    int
}

// Search walks the workspace recursively and returns lines containing query.
// It skips .git, non-regular files, oversized files, and binary files.
func (w *Local) Search(query string, opts SearchOptions) ([]Match, error) {
	if query == "" {
		return nil, nil
	}

	start := opts.Path
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	abs, err := w.Resolve(start)
	if err != nil {
		return nil, err
	}

	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = w.MaxSearchResults
	}
	if maxResults <= 0 {
		maxResults = DefaultMaxSearchResults
	}

	maxFile := w.MaxSearchFileBytes
	if maxFile <= 0 {
		maxFile = DefaultMaxSearchFileBytes
	}

	needle := query
	if !opts.CaseSensitive {
		needle = strings.ToLower(needle)
	}

	var matches []Match
	errStop := errors.New("search: result limit reached")
	walkErr := filepath.WalkDir(abs, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFile {
			return nil
		}

		found, err := searchFile(p, w.root, needle, opts.CaseSensitive)
		if err != nil {
			return nil
		}
		for _, m := range found {
			matches = append(matches, m)
			if len(matches) >= maxResults {
				return errStop
			}
		}
		return nil
	})

	if walkErr != nil && !errors.Is(walkErr, errStop) {
		return nil, fmt.Errorf("workspace: search %q: %w", start, walkErr)
	}
	return matches, nil
}

func searchFile(path, root, needle string, caseSensitive bool) ([]Match, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	head := make([]byte, binarySniffBytes)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	if bytes.IndexByte(head[:n], 0) >= 0 {
		return nil, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSearchLineBytes)

	var out []Match
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		haystack := line
		if !caseSensitive {
			haystack = strings.ToLower(line)
		}
		if strings.Contains(haystack, needle) {
			out = append(out, Match{Path: rel, Line: lineNo, Text: line})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
