package semantic

import (
	"io/fs"
	"testing"
)

func TestEngineSearchRanksMatches(t *testing.T) {
	fw := &fakeWalker{
		files: map[string]string{
			"a.txt": "alpha beta gamma",
			"b.txt": "beta delta epsilon",
		},
	}

	engine := NewEngine(fw, 10, 1024)
	res, err := engine.Search("beta gamma", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Path != "a.txt" {
		t.Fatalf("expected a.txt first, got %s", res[0].Path)
	}
	if !(res[0].Score > res[1].Score) {
		t.Fatalf("expected higher score for a.txt")
	}
}

type fakeWalker struct {
	files map[string]string
}

func (f *fakeWalker) WalkFiles(root string, maxFiles int, fn func(rel string, info fs.DirEntry) error) error {
	count := 0
	for path := range f.files {
		count++
		if maxFiles > 0 && count > maxFiles {
			break
		}
		_ = fn(path, fakeEntry{name: path})
	}
	return nil
}

func (f *fakeWalker) ReadFile(path string) (string, error) {
	return f.files[path], nil
}

type fakeEntry struct {
	name string
}

func (f fakeEntry) Name() string               { return f.name }
func (f fakeEntry) IsDir() bool                { return false }
func (f fakeEntry) Type() fs.FileMode          { return 0 }
func (f fakeEntry) Info() (fs.FileInfo, error) { return nil, nil }
