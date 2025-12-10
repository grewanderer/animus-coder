package semantic

import (
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// FileWalker abstracts file traversal and reading.
type FileWalker interface {
	WalkFiles(root string, maxFiles int, fn func(rel string, info fs.DirEntry) error) error
	ReadFile(path string) (string, error)
}

// Engine performs simple relevance search across project files.
type Engine struct {
	fs           FileWalker
	maxFiles     int
	maxFileBytes int
}

// Result captures a semantic search hit.
type Result struct {
	Path    string
	Score   float64
	Snippet string
}

// NewEngine constructs a semantic engine over the provided walker.
func NewEngine(fw FileWalker, maxFiles int, maxFileBytes int) *Engine {
	if maxFiles <= 0 {
		maxFiles = 200
	}
	if maxFileBytes <= 0 {
		maxFileBytes = 64 * 1024
	}
	return &Engine{fs: fw, maxFiles: maxFiles, maxFileBytes: maxFileBytes}
}

// Search returns top-k files ranked by token overlap with the query.
func (e *Engine) Search(query string, limit int) ([]Result, error) {
	if e == nil || e.fs == nil {
		return nil, fmt.Errorf("semantic engine unavailable")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if limit <= 0 {
		limit = 5
	}

	qTokens := tokenize(query)
	if len(qTokens) == 0 {
		return nil, fmt.Errorf("query too short")
	}

	results := make([]Result, 0, limit*2)
	err := e.fs.WalkFiles(".", e.maxFiles, func(rel string, info fs.DirEntry) error {
		if info.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		content, err := e.fs.ReadFile(rel)
		if err != nil {
			return nil
		}
		if len(content) > e.maxFileBytes {
			content = content[:e.maxFileBytes]
		}
		score := overlapScore(qTokens, tokenize(content))
		if score <= 0 {
			return nil
		}
		snippet := summarize(content)
		results = append(results, Result{Path: rel, Score: score, Snippet: snippet})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Path < results[j].Path
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func overlapScore(query, doc []string) float64 {
	if len(query) == 0 || len(doc) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(doc))
	for _, t := range doc {
		seen[t] = struct{}{}
	}
	var overlap int
	for _, q := range query {
		if _, ok := seen[q]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(query))
}

var tokenRe = regexp.MustCompile(`[A-Za-z0-9_]+`)

func tokenize(s string) []string {
	matches := tokenRe.FindAllString(strings.ToLower(s), -1)
	if len(matches) == 0 {
		return nil
	}
	return matches
}

func summarize(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if len(trim) > 200 {
			return trim[:200] + "..."
		}
		return trim
	}
	if len(content) > 200 {
		return content[:200] + "..."
	}
	return content
}
