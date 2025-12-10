package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathGuard ensures operations stay within a base directory.
type PathGuard struct {
	BaseDir string
}

// NewPathGuard constructs a guard rooted at baseDir (defaults to current working directory).
func NewPathGuard(baseDir string) (*PathGuard, error) {
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	return &PathGuard{BaseDir: absBase}, nil
}

// Resolve validates and returns an absolute path inside BaseDir.
func (g *PathGuard) Resolve(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path is required")
	}
	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	abs := filepath.Join(g.BaseDir, clean)
	abs = filepath.Clean(abs)

	if !strings.HasPrefix(abs, g.BaseDir+string(os.PathSeparator)) && abs != g.BaseDir {
		return "", fmt.Errorf("path escapes base directory")
	}
	return abs, nil
}
