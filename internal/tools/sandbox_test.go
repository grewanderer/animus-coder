package tools

import (
	"testing"

	"github.com/animus-coder/animus-coder/internal/config"
)

func TestSandboxRespectsAllowExec(t *testing.T) {
	sb, err := NewSandbox("", config.SandboxConfig{
		Enabled:         true,
		AllowWrite:      true,
		AllowedCommands: []string{"echo"},
		TimeoutSeconds:  5,
	}, config.ToolsConfig{
		AllowExec:      true,
		AllowFileWrite: true,
	})
	if err != nil {
		t.Fatalf("sandbox build: %v", err)
	}
	if sb.Terminal == nil || !sb.Terminal.AllowExecution {
		t.Fatalf("expected terminal exec enabled")
	}
}

func TestSandboxDisablesExecWhenConfigFalse(t *testing.T) {
	sb, err := NewSandbox("", config.SandboxConfig{
		Enabled:        true,
		AllowWrite:     true,
		TimeoutSeconds: 5,
	}, config.ToolsConfig{
		AllowExec:      false,
		AllowFileWrite: true,
	})
	if err != nil {
		t.Fatalf("sandbox build: %v", err)
	}
	if sb.Terminal.AllowExecution {
		t.Fatalf("expected terminal exec disabled")
	}
}

func TestSandboxAddsNetworkDeniesWhenDisabled(t *testing.T) {
	sb, err := NewSandbox("", config.SandboxConfig{
		Enabled:        true,
		AllowWrite:     true,
		AllowNetwork:   false,
		TimeoutSeconds: 5,
	}, config.ToolsConfig{
		AllowExec:      true,
		AllowFileWrite: true,
	})
	if err != nil {
		t.Fatalf("sandbox build: %v", err)
	}
	if sb.Terminal == nil {
		t.Fatalf("terminal missing")
	}
	if len(sb.Terminal.Denied) == 0 {
		t.Fatalf("expected network denies to be populated")
	}
	found := false
	for _, d := range sb.Terminal.Denied {
		if d == "curl" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected curl to be denied when network disabled")
	}
}
