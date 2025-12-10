package tools

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Filesystem provides safe file operations rooted at a base directory.
type Filesystem struct {
	guard      *PathGuard
	allowWrite bool
	allowRead  bool
}

// NewFilesystem builds a filesystem tool with write permissions controlled by allowWrite.
func NewFilesystem(baseDir string, allowWrite bool) (*Filesystem, error) {
	guard, err := NewPathGuard(baseDir)
	if err != nil {
		return nil, err
	}
	return &Filesystem{guard: guard, allowWrite: allowWrite, allowRead: true}, nil
}

// ReadFile returns file contents as string.
func (f *Filesystem) ReadFile(path string) (string, error) {
	if !f.allowRead {
		return "", errors.New("read is disabled by configuration")
	}
	resolved, err := f.guard.Resolve(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes content to a file if allowed.
func (f *Filesystem) WriteFile(path string, content string) error {
	if !f.allowWrite {
		return errors.New("write is disabled by configuration")
	}
	resolved, err := f.guard.Resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return err
	}
	return os.WriteFile(resolved, []byte(content), 0o644)
}

// Stat returns file info for a path inside the guard.
func (f *Filesystem) Stat(path string) (fs.FileInfo, error) {
	if !f.allowRead {
		return nil, errors.New("read is disabled by configuration")
	}
	resolved, err := f.guard.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.Stat(resolved)
}

// ListDir lists entries in a directory (names only).
func (f *Filesystem) ListDir(path string) ([]fs.DirEntry, error) {
	resolved, err := f.guard.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(resolved)
}

// Search looks for pattern occurrences in files under root (relative path).
func (f *Filesystem) Search(root string, pattern string, maxResults int) ([]SearchResult, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if maxResults <= 0 {
		maxResults = 20
	}

	resolved, err := f.guard.Resolve(root)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, maxResults)
	err = filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if len(results) >= maxResults {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(f.guard.BaseDir, path)

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 1
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), pattern) {
				results = append(results, SearchResult{
					Path:    rel,
					Line:    lineNum,
					Snippet: scanner.Text(),
				})
				if len(results) >= maxResults {
					return filepath.SkipDir
				}
			}
			lineNum++
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipDir) {
		return results, err
	}
	return results, nil
}

// SearchResult represents a single pattern match.
type SearchResult struct {
	Path    string
	Line    int
	Snippet string
}

// WalkFiles walks files under root and invokes fn with relative path and entry.
func (f *Filesystem) WalkFiles(root string, maxFiles int, fn func(rel string, info fs.DirEntry) error) error {
	if fn == nil {
		return fmt.Errorf("fn is required")
	}
	resolved, err := f.guard.Resolve(root)
	if err != nil {
		return err
	}
	count := 0
	return filepath.WalkDir(resolved, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if maxFiles > 0 && count >= maxFiles {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(f.guard.BaseDir, path)
		count++
		return fn(rel, d)
	})
}

// DescribeStructure returns a tree-like outline for a directory with depth/entry caps.
func (f *Filesystem) DescribeStructure(root string, maxDepth int, maxEntries int) (string, error) {
	if !f.allowRead {
		return "", errors.New("read is disabled by configuration")
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if maxEntries <= 0 {
		maxEntries = 200
	}

	resolved, err := f.guard.Resolve(root)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", root)
	}

	lines := []string{filepath.Clean(root) + "/"}
	added := 0

	var walk func(string, int) error
	walk = func(path string, depth int) error {
		if depth > maxDepth {
			return filepath.SkipDir
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

		for _, e := range entries {
			name := e.Name()
			if skipStructureDir(name) {
				continue
			}

			prefix := strings.Repeat("  ", depth-1)
			line := fmt.Sprintf("%s- %s", prefix, name)
			if e.IsDir() {
				line += "/"
			}
			lines = append(lines, line)
			added++
			if added >= maxEntries {
				lines = append(lines, fmt.Sprintf("%s... truncated after %d entries", prefix, maxEntries))
				return filepath.SkipDir
			}

			if e.IsDir() {
				if err := walk(filepath.Join(path, name), depth+1); err != nil {
					if errors.Is(err, filepath.SkipDir) {
						continue
					}
					return err
				}
			}
		}
		return nil
	}

	if err := walk(resolved, 1); err != nil && !errors.Is(err, filepath.SkipDir) {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

func skipStructureDir(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case ".git", "node_modules", ".idea", ".vscode", "vendor", ".cache", ".github":
		return true
	default:
		return false
	}
}
