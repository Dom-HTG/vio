package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestWorkspace(t *testing.T) *Local {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return w
}

func TestNewDetectsRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "internal", "agent")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	w, err := New(nested)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if w.Root() != want {
		t.Fatalf("Root = %q, want %q", w.Root(), want)
	}
}

func TestResolvePathSafety(t *testing.T) {
	w := newTestWorkspace(t)
	root := w.Root()
	inside := filepath.Join(root, "a", "b.txt")

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr error
	}{
		{name: "relative", path: "a/b.txt", want: inside},
		{name: "dot", path: ".", want: root},
		{name: "parent stays inside", path: "a/../b.txt", want: filepath.Join(root, "b.txt")},
		{name: "absolute inside", path: inside, want: inside},
		{name: "relative escape", path: "../outside.txt", wantErr: ErrOutsideRoot},
		{name: "absolute escape", path: filepath.Join(filepath.Dir(root), "outside.txt"), wantErr: ErrOutsideRoot},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := w.Resolve(tc.path)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Resolve(%q) error = %v, want %v", tc.path, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tc.path, err)
			}
			if got != tc.want {
				t.Fatalf("Resolve(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestResolveRejectsSymlinkEscape(t *testing.T) {
	w := newTestWorkspace(t)
	outside := t.TempDir()
	link := filepath.Join(w.Root(), "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := w.Resolve(filepath.Join("link", "secret.txt")); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("Resolve through symlink error = %v, want ErrOutsideRoot", err)
	}
}

func TestFileOpsRoundTrip(t *testing.T) {
	w := newTestWorkspace(t)

	if ok, err := w.Exists("sub/hello.txt"); err != nil || ok {
		t.Fatalf("Exists before write = %v, %v; want false, nil", ok, err)
	}
	if err := w.Write("sub/hello.txt", []byte("hi")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if ok, err := w.Exists("sub/hello.txt"); err != nil || !ok {
		t.Fatalf("Exists after write = %v, %v; want true, nil", ok, err)
	}

	data, err := w.Read("sub/hello.txt")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(data) != "hi" {
		t.Fatalf("Read = %q, want %q", data, "hi")
	}

	entries, err := w.List("sub")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "hello.txt" || entries[0].Path != "sub/hello.txt" || entries[0].IsDir {
		t.Fatalf("unexpected listing: %+v", entries)
	}

	if err := w.Delete("sub/hello.txt"); err != nil {
		t.Fatalf("Delete file: %v", err)
	}
	if ok, err := w.Exists("sub/hello.txt"); err != nil || ok {
		t.Fatalf("Exists after delete = %v, %v; want false, nil", ok, err)
	}
	if err := w.Delete("sub"); err != nil {
		t.Fatalf("Delete dir: %v", err)
	}
}

func TestReadRejectsDirectoryAndOversize(t *testing.T) {
	w := newTestWorkspace(t)

	if _, err := w.Read("."); !errors.Is(err, ErrIsDir) {
		t.Fatalf("Read(dir) error = %v, want ErrIsDir", err)
	}

	w.MaxReadBytes = 4
	if err := w.Write("big.txt", []byte("0123456789")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := w.Read("big.txt"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Read(oversize) error = %v, want ErrTooLarge", err)
	}
}

func TestListLimit(t *testing.T) {
	w := newTestWorkspace(t)
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := w.Write(name, nil); err != nil {
			t.Fatal(err)
		}
	}
	w.MaxListEntries = 1
	if _, err := w.List("."); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("List over limit error = %v, want ErrTooLarge", err)
	}
}

func TestSearch(t *testing.T) {
	w := newTestWorkspace(t)
	files := map[string]string{
		"a.go":        "Hello world\nfoo\n",
		"sub/b.go":    "hello again\nHELLO\n",
		".git/config": "hello in git metadata\n",
	}
	for name, content := range files {
		if err := w.Write(name, []byte(content)); err != nil {
			t.Fatalf("Write %s: %v", name, err)
		}
	}
	if err := w.Write("blob.bin", []byte{0, 'h', 'e', 'l', 'l', 'o'}); err != nil {
		t.Fatal(err)
	}

	matches, err := w.Search("hello", SearchOptions{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := []Match{
		{Path: "a.go", Line: 1, Text: "Hello world"},
		{Path: "sub/b.go", Line: 1, Text: "hello again"},
		{Path: "sub/b.go", Line: 2, Text: "HELLO"},
	}
	if len(matches) != len(want) {
		t.Fatalf("got %d matches, want %d: %+v", len(matches), len(want), matches)
	}
	for i := range want {
		if matches[i] != want[i] {
			t.Fatalf("match[%d] = %+v, want %+v", i, matches[i], want[i])
		}
	}

	caseSensitive, err := w.Search("Hello", SearchOptions{CaseSensitive: true})
	if err != nil {
		t.Fatalf("Search case-sensitive: %v", err)
	}
	if len(caseSensitive) != 1 || caseSensitive[0].Path != "a.go" {
		t.Fatalf("case-sensitive matches = %+v, want only a.go", caseSensitive)
	}

	limited, err := w.Search("hello", SearchOptions{MaxResults: 1})
	if err != nil {
		t.Fatalf("Search limited: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("limited matches = %d, want 1", len(limited))
	}

	scoped, err := w.Search("hello", SearchOptions{Path: "sub"})
	if err != nil {
		t.Fatalf("Search scoped: %v", err)
	}
	if len(scoped) != 2 {
		t.Fatalf("scoped matches = %d, want 2", len(scoped))
	}
}
