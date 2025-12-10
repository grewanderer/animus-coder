package tools

import (
	"path/filepath"
	"testing"
)

func TestFilesystemReadWriteList(t *testing.T) {
	dir := t.TempDir()
	fsTool, err := NewFilesystem(dir, true)
	if err != nil {
		t.Fatalf("new filesystem: %v", err)
	}

	requireNoError(t, fsTool.WriteFile("sub/file.txt", "hello"))

	content, err := fsTool.ReadFile("sub/file.txt")
	requireNoError(t, err)
	if content != "hello" {
		t.Fatalf("expected hello, got %s", content)
	}

	entries, err := fsTool.ListDir("sub")
	requireNoError(t, err)
	if len(entries) != 1 || entries[0].Name() != "file.txt" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestFilesystemPreventsTraversal(t *testing.T) {
	dir := t.TempDir()
	fsTool, err := NewFilesystem(dir, false)
	if err != nil {
		t.Fatalf("new filesystem: %v", err)
	}

	if _, err := fsTool.ReadFile("../etc/passwd"); err == nil {
		t.Fatalf("expected traversal error")
	}
}

func TestFilesystemSearch(t *testing.T) {
	dir := t.TempDir()
	fsTool, err := NewFilesystem(dir, true)
	requireNoError(t, err)

	requireNoError(t, fsTool.WriteFile("a.txt", "hello world\nsecond line"))
	requireNoError(t, fsTool.WriteFile(filepath.Join("nested", "b.txt"), "hello again"))

	results, err := fsTool.Search(".", "hello", 10)
	requireNoError(t, err)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
