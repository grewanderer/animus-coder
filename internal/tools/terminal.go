package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Terminal executes commands with allow/deny checks.
type Terminal struct {
	WorkingDir     string
	Allowed        []string
	Denied         []string
	Timeout        time.Duration
	AllowExecution bool
}

// ExecResult carries output and status code.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Exec runs a command if allowed by configuration.
func (t *Terminal) Exec(ctx context.Context, command string, args ...string) (ExecResult, error) {
	if !t.AllowExecution {
		return ExecResult{}, errors.New("execution disabled by configuration")
	}
	if command == "" {
		return ExecResult{}, fmt.Errorf("command is required")
	}
	if err := t.validateCommand(command); err != nil {
		return ExecResult{}, err
	}

	timeout := t.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	if t.WorkingDir != "" {
		cmd.Dir = t.WorkingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	res := ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		ExitCode: func() int {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			if err != nil {
				return -1
			}
			return 0
		}(),
	}

	if err != nil {
		return res, err
	}
	return res, nil
}

func (t *Terminal) validateCommand(cmd string) error {
	lower := strings.ToLower(cmd)
	for _, deny := range t.Denied {
		if lower == strings.ToLower(deny) {
			return fmt.Errorf("command %q is denied", cmd)
		}
	}
	if len(t.Allowed) > 0 {
		for _, allow := range t.Allowed {
			if lower == strings.ToLower(allow) {
				return nil
			}
		}
		return fmt.Errorf("command %q is not in allowlist", cmd)
	}
	return nil
}
