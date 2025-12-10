package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitStatusAndApplyPatchDryRun(t *testing.T) {
	dir := t.TempDir()
	run := func(cmd string, args ...string) {
		c := exec.Command(cmd, args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("cmd %s %v failed: %v, out=%s", cmd, args, err, string(out))
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("git", "add", "file.txt")

	gitTool := &GitTool{WorkingDir: dir, AllowExec: true}

	status, err := gitTool.Status()
	if err != nil {
		t.Fatalf("status err: %v", err)
	}
	if status == "" {
		t.Fatalf("expected status output")
	}

	patch := "diff --git a/file.txt b/file.txt\nindex e69de29..4b825dc 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1 +1,2 @@\n hello\n+hello world\n"
	if _, err := gitTool.ApplyPatch(patch, true); err != nil {
		t.Fatalf("apply patch dry-run failed: %v", err)
	}
}

func TestGitApplyPatchDryRunOnly(t *testing.T) {
	g := &GitTool{AllowExec: true, DryRunOnly: true}
	if _, err := g.ApplyPatch("patch", false); err == nil {
		t.Fatalf("expected dry-run restriction error")
	}
}

func TestGitApplyPatchCreatesBackup(t *testing.T) {
	dir := t.TempDir()
	run := func(cmd string, args ...string) {
		c := exec.Command(cmd, args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("cmd %s %v failed: %v, out=%s", cmd, args, err, string(out))
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("git", "add", "f.txt")

	g := &GitTool{WorkingDir: dir, AllowExec: true, DryRunOnly: false, BackupDir: ".backup"}
	patch := "diff --git a/f.txt b/f.txt\nindex e69de29..4b825dc 100644\n--- a/f.txt\n+++ b/f.txt\n@@ -1 +1,2 @@\n one\n+two\n"
	if _, err := g.ApplyPatch(patch, false); err != nil {
		t.Fatalf("apply patch failed: %v", err)
	}

	files, err := os.ReadDir(filepath.Join(dir, ".backup"))
	if err != nil {
		t.Fatalf("backup dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected backup file")
	}
}

func TestRestoreBackup(t *testing.T) {
	dir := t.TempDir()
	run := func(cmd string, args ...string) {
		c := exec.Command(cmd, args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("cmd %s %v failed: %v, out=%s", cmd, args, err, string(out))
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("git", "add", "f.txt")

	g := &GitTool{WorkingDir: dir, AllowExec: true, DryRunOnly: false, BackupDir: ".backup"}
	patch := "diff --git a/f.txt b/f.txt\nindex e69de29..4b825dc 100644\n--- a/f.txt\n+++ b/f.txt\n@@ -1 +1,2 @@\n one\n+two\n"
	if _, err := g.ApplyPatch(patch, false); err != nil {
		t.Fatalf("apply patch failed: %v", err)
	}

	if _, err := g.RestoreBackup(""); err != nil {
		t.Fatalf("restore backup failed: %v", err)
	}

	backups, err := g.ListBackups()
	if err != nil {
		t.Fatalf("list backups: %v", err)
	}
	if len(backups) == 0 {
		t.Fatalf("expected backups listed")
	}

	preview, err := g.PreviewBackup("")
	if err != nil || preview == "" {
		t.Fatalf("expected preview content, err=%v", err)
	}
}
