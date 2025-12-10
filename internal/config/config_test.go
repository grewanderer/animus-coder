package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	configYAML := `
version: "0.1.0"
providers:
  openai:
    type: openai
    base_url: https://api.openai.com
    api_key: dummy
    timeout: 30s
models:
  main:
    provider: openai
    model: gpt-4o
    temperature: 0.2
    max_tokens: 2048
    default: true
sandbox:
  enabled: true
tools:
  allow_exec: true
agent:
  max_steps: 6
`

	require.NoError(t, os.WriteFile(cfgPath, []byte(configYAML), 0o644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	require.Equal(t, "openai", cfg.Models["main"].Provider)
	require.Equal(t, 6, cfg.Agent.MaxSteps)
	require.Equal(t, true, cfg.Sandbox.Enabled)
}

func TestEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	configYAML := `
providers:
  openrouter:
    type: openrouter
    base_url: https://openrouter.ai
    api_key: dummy
models:
  coder:
    provider: openrouter
    model: qwen2.5
    default: true
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(configYAML), 0o644))

	t.Setenv("MYCODEX_AGENT_MAX_STEPS", "12")
	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	require.Equal(t, 12, cfg.Agent.MaxSteps)
}

func TestValidateFailsOnUnknownProvider(t *testing.T) {
	cfg := Config{
		Providers: map[string]ProviderConfig{
			"openai": {Type: "openai"},
		},
		Models: map[string]ModelConfig{
			"broken": {Provider: "missing", Default: true},
		},
		Agent: AgentConfig{
			MaxSteps: 1,
		},
		Sandbox: SandboxConfig{
			TimeoutSeconds: 10,
		},
		Tools: ToolsConfig{
			ExecTimeoutSeconds: 10,
		},
	}

	err := cfg.Validate()
	require.Error(t, err)
}
