package tools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitTool provides minimal git operations with dry-run apply_patch.
type GitTool struct {
	WorkingDir string
	AllowExec  bool
	DryRunOnly bool
	BackupDir  string
	stack      *patchStack
}

// Status returns git status --short.
func (g *GitTool) Status() (string, error) {
	if !g.AllowExec {
		return "", fmt.Errorf("git operations disabled")
	}
	out, err := g.run([]string{"status", "--short"})
	return out, err
}

// ApplyPatch applies a patch; when dryRun=true it uses --check.
func (g *GitTool) ApplyPatch(patch string, dryRun bool) (string, error) {
	if !g.AllowExec {
		return "", fmt.Errorf("git operations disabled")
	}
	if g.DryRunOnly && !dryRun {
		return "", fmt.Errorf("apply_patch is restricted to dry-run mode")
	}
	if !dryRun {
		if err := g.createBackup(patch); err != nil {
			return "", fmt.Errorf("create backup: %w", err)
		}
	}
	args := []string{"apply"}
	if dryRun {
		args = append(args, "--check")
	}
	args = append(args, "-")
	return g.runWithInput(args, patch)
}

// RestoreBackup applies a backup by id/name (latest when empty).
func (g *GitTool) RestoreBackup(name string) (string, error) {
	if !g.AllowExec {
		return "", fmt.Errorf("git operations disabled")
	}
	if g.DryRunOnly {
		return "", fmt.Errorf("restore_backup not allowed in dry-run-only mode")
	}
	dir := g.BackupDir
	if dir == "" {
		dir = ".mycodex/patch-backups"
	}
	data, err := g.readBackup(dir, name)
	if err != nil {
		return "", err
	}
	return g.applyPatchDataReverse(data)
}

func (g *GitTool) createBackup(patch string) error {
	targetDir := g.BackupDir
	if targetDir == "" {
		targetDir = ".mycodex/patch-backups"
	}
	if err := os.MkdirAll(filepath.Join(g.WorkingDir, targetDir), 0o755); err != nil {
		return err
	}
	if g.stack == nil {
		stackPath := filepath.Join(g.WorkingDir, targetDir, "stack.json")
		st, err := loadPatchStack(stackPath)
		if err != nil {
			return err
		}
		g.stack = st
	}
	parent := ""
	if latest := g.stack.latest(); latest != nil {
		parent = latest.ID
	}
	entryID := fmt.Sprintf("backup-%d", time.Now().UnixNano())
	filename := entryID + ".patch"
	path := filepath.Join(g.WorkingDir, targetDir, filename)
	if err := os.WriteFile(path, []byte(patch), 0o644); err != nil {
		return err
	}
	g.stack.Entries = append(g.stack.Entries, PatchEntry{
		ID:        entryID,
		ParentID:  parent,
		FileName:  filename,
		CreatedAt: time.Now().UTC(),
	})
	return g.stack.save(filepath.Join(g.WorkingDir, targetDir, "stack.json"))
}

// ListBackups returns backup filenames sorted as returned by os.ReadDir.
func (g *GitTool) ListBackups() ([]string, error) {
	dir := g.BackupDir
	if dir == "" {
		dir = ".mycodex/patch-backups"
	}
	stack, err := g.loadOrInitStack(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(stack.Entries))
	for _, e := range stack.Entries {
		out = append(out, e.ID)
	}
	return out, nil
}

// PreviewBackup returns contents of a specific backup (or latest if empty).
func (g *GitTool) PreviewBackup(name string) (string, error) {
	data, err := g.readBackup(g.BackupDir, name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (g *GitTool) readBackup(dir string, name string) ([]byte, error) {
	if dir == "" {
		dir = ".mycodex/patch-backups"
	}
	stack, err := g.loadOrInitStack(dir)
	if err != nil {
		return nil, err
	}
	if len(stack.Entries) == 0 {
		return nil, fmt.Errorf("no backups available")
	}
	var targetFile string
	if name != "" {
		for _, e := range stack.Entries {
			if e.ID == name || e.FileName == name {
				targetFile = e.FileName
				break
			}
		}
		if targetFile == "" {
			return nil, fmt.Errorf("backup %s not found", name)
		}
	} else {
		targetFile = stack.Entries[len(stack.Entries)-1].FileName
	}
	data, err := os.ReadFile(filepath.Join(g.WorkingDir, dir, targetFile))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (g *GitTool) applyPatchDataReverse(data []byte) (string, error) {
	args := []string{"apply", "-R"}
	return g.runWithInput(args, string(data))
}

func (g *GitTool) run(args []string) (string, error) {
	return g.runWithInput(args, "")
}

func (g *GitTool) runWithInput(args []string, input string) (string, error) {
	cmd := exec.Command("git", args...)
	if g.WorkingDir != "" {
		cmd.Dir = g.WorkingDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	if err := cmd.Run(); err != nil {
		return stderr.String(), err
	}
	return stdout.String(), nil
}

func (g *GitTool) loadOrInitStack(dir string) (*patchStack, error) {
	if g.stack != nil {
		return g.stack, nil
	}
	path := filepath.Join(g.WorkingDir, dir, "stack.json")
	st, err := loadPatchStack(path)
	if err != nil {
		return nil, err
	}
	g.stack = st
	return st, nil
}
