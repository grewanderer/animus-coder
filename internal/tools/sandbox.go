package tools

import (
	"fmt"
	"time"

	"github.com/animus-coder/animus-coder/internal/config"
)

// Sandbox constructs configured tool instances based on sandbox/tools config.
type Sandbox struct {
	FS       *Filesystem
	Terminal *Terminal
}

var defaultNetworkDenied = []string{
	"curl", "wget", "ping", "nc", "netcat", "telnet", "ssh", "scp", "sftp",
}

// NewSandbox builds filesystem and terminal tools respecting config flags.
func NewSandbox(baseDir string, sandboxCfg config.SandboxConfig, toolsCfg config.ToolsConfig) (*Sandbox, error) {
	fsTool, err := NewFilesystem(baseDir, sandboxCfg.AllowWrite && toolsCfg.AllowFileWrite)
	if err != nil {
		return nil, fmt.Errorf("build filesystem tool: %w", err)
	}
	if !sandboxCfg.AllowNetwork {
		// No-op placeholder; network is controlled via terminal allow/deny in this stub.
	}

	denied := append([]string{}, sandboxCfg.DeniedCommands...)
	if !sandboxCfg.AllowNetwork {
		denied = append(denied, defaultNetworkDenied...)
	}

	term := &Terminal{
		WorkingDir:     baseDir,
		Allowed:        sandboxCfg.AllowedCommands,
		Denied:         dedupeStrings(denied),
		Timeout:        time.Duration(sandboxCfg.TimeoutSeconds) * time.Second,
		AllowExecution: toolsCfg.AllowExec && sandboxCfg.Enabled && allowCommands(sandboxCfg),
	}

	return &Sandbox{
		FS:       fsTool,
		Terminal: term,
	}, nil
}

func allowCommands(s config.SandboxConfig) bool {
	return s.AllowWrite || len(s.AllowedCommands) > 0 || len(s.DeniedCommands) > 0 || s.AllowNetwork
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, v := range values {
		lower := v
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, v)
	}
	return out
}
