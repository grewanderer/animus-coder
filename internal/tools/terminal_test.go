package tools

import (
	"context"
	"testing"
	"time"
)

func TestTerminalExecAllowsWhitelisted(t *testing.T) {
	term := &Terminal{
		WorkingDir:     "",
		Allowed:        []string{"echo"},
		Denied:         []string{"rm"},
		Timeout:        time.Second * 2,
		AllowExecution: true,
	}

	res, err := term.Exec(context.Background(), "echo", "hi")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d", res.ExitCode)
	}
	if res.Stdout == "" {
		t.Fatalf("expected stdout")
	}
}

func TestTerminalExecDenied(t *testing.T) {
	term := &Terminal{
		Denied:         []string{"rm"},
		AllowExecution: true,
	}
	if _, err := term.Exec(context.Background(), "rm", "-rf", "/"); err == nil {
		t.Fatalf("expected deny error")
	}
}

func TestTerminalExecNetworkDeniedByDefaultList(t *testing.T) {
	term := &Terminal{
		Denied:         []string{"curl"},
		AllowExecution: true,
	}
	if _, err := term.Exec(context.Background(), "curl", "http://example.com"); err == nil {
		t.Fatalf("expected network deny error")
	}
}

func TestTerminalExecDisabled(t *testing.T) {
	term := &Terminal{AllowExecution: false}
	if _, err := term.Exec(context.Background(), "echo", "hi"); err == nil {
		t.Fatalf("expected disabled error")
	}
}
