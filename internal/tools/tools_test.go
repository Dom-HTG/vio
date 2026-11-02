package tools

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"vio/internal/workspace"
)

func testFS(t *testing.T) workspace.FS {
	t.Helper()
	ws, err := workspace.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func run(t *testing.T, tool Tool, args map[string]any) ToolOutput {
	t.Helper()
	out, err := tool.Execute(context.Background(), ToolInput{Args: args})
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", tool.Name(), err)
	}
	return out
}

func TestRegistry(t *testing.T) {
	ws := testFS(t)
	r := NewRegistry()

	if err := r.Register(NewReadFile(ws)); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(NewReadFile(ws)); err == nil {
		t.Fatal("duplicate registration should fail")
	}
	if err := r.Register(nil); err == nil {
		t.Fatal("nil tool should fail")
	}

	if _, ok := r.Get("read_file"); !ok {
		t.Fatal("read_file not found")
	}
	if _, ok := r.Get("missing"); ok {
		t.Fatal("missing tool found")
	}
	if got := r.Names(); !reflect.DeepEqual(got, []string{"read_file"}) {
		t.Fatalf("Names = %v", got)
	}
}

func TestRegistryDefinitionsAreSorted(t *testing.T) {
	ws := testFS(t)
	r := NewRegistry()
	for _, tool := range []Tool{
		NewShell(ws),
		NewSearch(ws),
		NewEditFile(ws),
		NewWriteFile(ws),
		NewReadFile(ws),
	} {
		if err := r.Register(tool); err != nil {
			t.Fatal(err)
		}
	}

	want := []string{"edit_file", "read_file", "search", "shell", "write_file"}
	if got := r.Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names = %v, want %v", got, want)
	}

	defs := r.Definitions()
	if len(defs) != len(want) {
		t.Fatalf("got %d definitions, want %d", len(defs), len(want))
	}
	for i, name := range want {
		if defs[i].Name != name {
			t.Fatalf("definition[%d] = %q, want %q", i, defs[i].Name, name)
		}
		if defs[i].Params == nil {
			t.Fatalf("definition %q has nil params", name)
		}
	}
}

func TestReadWriteEdit(t *testing.T) {
	ws := testFS(t)

	if out := run(t, NewWriteFile(ws), map[string]any{"path": "dir/a.txt", "content": "hello\n"}); out.IsError {
		t.Fatalf("write failed: %s", out.Output)
	}
	if out := run(t, NewReadFile(ws), map[string]any{"path": "dir/a.txt"}); out.IsError || out.Output != "hello\n" {
		t.Fatalf("read = %+v", out)
	}

	if out := run(t, NewEditFile(ws), map[string]any{"path": "dir/a.txt", "old_text": "hello", "new_text": "world"}); out.IsError {
		t.Fatalf("edit failed: %s", out.Output)
	}
	if out := run(t, NewReadFile(ws), map[string]any{"path": "dir/a.txt"}); out.Output != "world\n" {
		t.Fatalf("read after edit = %q", out.Output)
	}
}

func TestEditRejectsMissingAndAmbiguousText(t *testing.T) {
	ws := testFS(t)
	run(t, NewWriteFile(ws), map[string]any{"path": "a.txt", "content": "x\nx\n"})

	if out := run(t, NewEditFile(ws), map[string]any{"path": "a.txt", "old_text": "missing", "new_text": "y"}); !out.IsError {
		t.Fatal("missing old_text should fail")
	}
	if out := run(t, NewEditFile(ws), map[string]any{"path": "a.txt", "old_text": "x", "new_text": "y"}); !out.IsError {
		t.Fatal("ambiguous old_text should fail")
	}
}

func TestReadRejectsEscapeAndMissingFile(t *testing.T) {
	ws := testFS(t)
	if out := run(t, NewReadFile(ws), map[string]any{"path": "../outside.txt"}); !out.IsError {
		t.Fatal("path escape should fail")
	}
	if out := run(t, NewReadFile(ws), map[string]any{"path": "missing.txt"}); !out.IsError {
		t.Fatal("missing file should fail")
	}
}

func TestSearchTool(t *testing.T) {
	ws := testFS(t)
	run(t, NewWriteFile(ws), map[string]any{"path": "a.txt", "content": "Alpha\nbeta\n"})
	run(t, NewWriteFile(ws), map[string]any{"path": "b.txt", "content": "alpha two\n"})

	out := run(t, NewSearch(ws), map[string]any{"query": "alpha"})
	if out.IsError {
		t.Fatalf("search failed: %s", out.Output)
	}
	if !strings.Contains(out.Output, "a.txt:1: Alpha") || !strings.Contains(out.Output, "b.txt:1: alpha two") {
		t.Fatalf("unexpected search output:\n%s", out.Output)
	}

	out = run(t, NewSearch(ws), map[string]any{"query": "Alpha", "case_sensitive": true})
	if strings.Contains(out.Output, "b.txt") {
		t.Fatalf("case-sensitive search matched b.txt:\n%s", out.Output)
	}
}

func TestShellTool(t *testing.T) {
	ws := testFS(t)
	shell := NewShell(ws)

	out := run(t, shell, map[string]any{"command": "echo hello"})
	if out.IsError || !strings.Contains(out.Output, "hello") {
		t.Fatalf("echo = %+v", out)
	}

	out = run(t, shell, map[string]any{"command": "exit 1"})
	if !out.IsError || !strings.Contains(out.Output, "exit code 1") {
		t.Fatalf("exit 1 = %+v", out)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if out, err := shell.Execute(ctx, ToolInput{Args: map[string]any{"command": "echo hello"}}); err != nil ||
		!out.IsError || !strings.Contains(out.Output, "canceled") {
		t.Fatalf("canceled = %+v, %v", out, err)
	}
}
