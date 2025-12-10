package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config describes the top-level application configuration loaded from YAML and ENV.
type Config struct {
	Version   string                    `mapstructure:"version"`
	Providers map[string]ProviderConfig `mapstructure:"providers"`
	Models    map[string]ModelConfig    `mapstructure:"models"`
	Strategy  StrategyConfig            `mapstructure:"strategy"`
	Sandbox   SandboxConfig             `mapstructure:"sandbox"`
	Tools     ToolsConfig               `mapstructure:"tools"`
	Agent     AgentConfig               `mapstructure:"agent"`
	Logging   LoggingConfig             `mapstructure:"logging"`
	Server    ServerConfig              `mapstructure:"server"`
}

// ProviderConfig represents LLM provider configuration such as OpenAI, Ollama, or custom gateways.
type ProviderConfig struct {
	Type      string        `mapstructure:"type"`       // openai, openrouter, ollama, vllm, lmstudio, custom
	Model     string        `mapstructure:"model"`      // default model for the provider
	BaseURL   string        `mapstructure:"base_url"`   // API base URL
	APIKey    string        `mapstructure:"api_key"`    // optional API key
	Timeout   time.Duration `mapstructure:"timeout"`    // request timeout
	MaxTokens int           `mapstructure:"max_tokens"` // optional provider-level token cap
}

// ModelConfig binds a logical model name to a provider entry and model parameters.
type ModelConfig struct {
	Provider    string  `mapstructure:"provider"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Default     bool    `mapstructure:"default"`
	Expensive   bool    `mapstructure:"expensive"`
}

// SandboxConfig controls command and filesystem restrictions.
type SandboxConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	AllowNetwork    bool     `mapstructure:"allow_network"`
	AllowWrite      bool     `mapstructure:"allow_write"`
	AllowedCommands []string `mapstructure:"allowed_commands"`
	DeniedCommands  []string `mapstructure:"denied_commands"`
	WorkingDir      string   `mapstructure:"working_dir"`
	TimeoutSeconds  int      `mapstructure:"timeout_seconds"`
}

// ToolsConfig configures tool behaviour.
type ToolsConfig struct {
	AllowExec            bool `mapstructure:"allow_exec"`
	AllowGit             bool `mapstructure:"allow_git"`
	AllowFileWrite       bool `mapstructure:"allow_file_write"`
	ExecTimeoutSeconds   int  `mapstructure:"exec_timeout_seconds"`
	EnableSemantic       bool `mapstructure:"enable_semantic"`
	SemanticMaxFiles     int  `mapstructure:"semantic_max_files"`
	SemanticMaxFileBytes int  `mapstructure:"semantic_max_file_bytes"`
}

// AgentConfig describes Agent Core runtime parameters.
type AgentConfig struct {
	MaxSteps           int     `mapstructure:"max_steps"`
	MaxTokens          int     `mapstructure:"max_tokens"`
	Temperature        float64 `mapstructure:"temperature"`
	EnablePlan         bool    `mapstructure:"enable_plan"`
	EnableReflect      bool    `mapstructure:"enable_reflect"`
	ReflectionPolicy   string  `mapstructure:"reflection_policy"`
	EnableSelfDiff     bool    `mapstructure:"enable_self_diff"`
	EnableTestRun      bool    `mapstructure:"enable_test_run"`
	TestCommand        string  `mapstructure:"test_command"`
	TestRetries        int     `mapstructure:"test_retries"`
	TestTimeoutSeconds int     `mapstructure:"test_timeout_seconds"`
	MaxContextBytes    int     `mapstructure:"max_context_bytes"`
}

// LoggingConfig controls logger behaviour.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // console or json
}

// ServerConfig describes daemon settings.
type ServerConfig struct {
	Addr           string `mapstructure:"addr"`
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	Transport      string `mapstructure:"transport"` // connect or ndjson
}

// Load reads configuration from the provided path or defaults to configs/config.yaml.
// Environment variables override file values (prefix: MYCODEX_, dots replaced with underscores).
func Load(path string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix("MYCODEX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if path == "" {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("configs")
	} else {
		v.SetConfigFile(path)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) && path == "" {
			v.SetConfigName("config.example")
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("read config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults populates sensible defaults for optional fields.
func setDefaults(v *viper.Viper) {
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")

	v.SetDefault("sandbox.enabled", true)
	v.SetDefault("sandbox.allow_network", false)
	v.SetDefault("sandbox.allow_write", false)
	v.SetDefault("sandbox.timeout_seconds", 120)

	v.SetDefault("tools.allow_exec", true)
	v.SetDefault("tools.allow_git", true)
	v.SetDefault("tools.allow_file_write", true)
	v.SetDefault("tools.exec_timeout_seconds", 120)
	v.SetDefault("tools.enable_semantic", false)
	v.SetDefault("tools.semantic_max_files", 200)
	v.SetDefault("tools.semantic_max_file_bytes", 65536)

	v.SetDefault("agent.max_steps", 8)
	v.SetDefault("agent.max_tokens", 1024)
	v.SetDefault("agent.temperature", 0.2)
	v.SetDefault("agent.enable_plan", true)
	v.SetDefault("agent.enable_reflect", true)
	v.SetDefault("agent.reflection_policy", "block_on_critical")
	v.SetDefault("agent.enable_self_diff", false)
	v.SetDefault("agent.enable_test_run", false)
	v.SetDefault("agent.test_command", "")
	v.SetDefault("agent.test_retries", 0)
	v.SetDefault("agent.test_timeout_seconds", 0)
	v.SetDefault("agent.max_context_bytes", 32768)

	v.SetDefault("strategy.default_model", "")
	v.SetDefault("strategy.planner_model", "")
	v.SetDefault("strategy.coder_model", "")
	v.SetDefault("strategy.critic_model", "")
	v.SetDefault("strategy.overrides", map[string]string{})
	v.SetDefault("strategy.fallbacks", []string{})
	v.SetDefault("strategy.max_expensive", 0)

	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.metrics_enabled", true)
	v.SetDefault("server.transport", "connect")
}

// Validate performs basic sanity checks on configuration values.
func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return errors.New("at least one provider must be configured")
	}

	if len(c.Models) == 0 {
		return errors.New("at least one model must be defined")
	}

	var defaultFound bool
	for name, p := range c.Providers {
		if p.Type == "" {
			return fmt.Errorf("provider %q must define type", name)
		}
	}

	for name, m := range c.Models {
		if m.Provider == "" {
			return fmt.Errorf("model %q must reference provider", name)
		}

		if _, ok := c.Providers[m.Provider]; !ok {
			return fmt.Errorf("model %q references unknown provider %q", name, m.Provider)
		}

		if m.Temperature < 0 || m.Temperature > 2 {
			return fmt.Errorf("model %q temperature must be within [0,2]", name)
		}

		if m.MaxTokens < 0 {
			return fmt.Errorf("model %q max_tokens cannot be negative", name)
		}

		if m.Default {
			defaultFound = true
		}
	}

	if !defaultFound {
		return errors.New("at least one model should be marked as default")
	}

	if c.Agent.MaxSteps <= 0 {
		return errors.New("agent.max_steps must be > 0")
	}

	if c.Agent.MaxContextBytes < 0 {
		return errors.New("agent.max_context_bytes must be >= 0")
	}

	if c.Agent.EnableTestRun && strings.TrimSpace(c.Agent.TestCommand) == "" {
		return errors.New("agent.test_command must be set when agent.enable_test_run is true")
	}
	if c.Agent.TestRetries < 0 {
		return errors.New("agent.test_retries must be >= 0")
	}
	if c.Agent.TestTimeoutSeconds < 0 {
		return errors.New("agent.test_timeout_seconds must be >= 0")
	}
	switch strings.ToLower(strings.TrimSpace(c.Agent.ReflectionPolicy)) {
	case "", "block_on_critical", "never_block", "warn_only":
	default:
		return fmt.Errorf("agent.reflection_policy must be one of block_on_critical, never_block, warn_only")
	}

	if c.Sandbox.TimeoutSeconds <= 0 {
		return errors.New("sandbox.timeout_seconds must be > 0")
	}

	if c.Tools.ExecTimeoutSeconds <= 0 {
		return errors.New("tools.exec_timeout_seconds must be > 0")
	}

	for _, modelID := range []string{
		c.Strategy.DefaultModel, c.Strategy.PlannerModel, c.Strategy.CoderModel, c.Strategy.CriticModel,
	} {
		if strings.TrimSpace(modelID) == "" {
			continue
		}
		if _, ok := c.Models[modelID]; !ok {
			return fmt.Errorf("strategy references unknown model %q", modelID)
		}
	}
	for _, modelID := range c.Strategy.Fallbacks {
		if _, ok := c.Models[modelID]; !ok {
			return fmt.Errorf("strategy fallback references unknown model %q", modelID)
		}
	}
	for _, modelID := range c.Strategy.Overrides {
		if _, ok := c.Models[modelID]; !ok {
			return fmt.Errorf("strategy override references unknown model %q", modelID)
		}
	}
	if c.Strategy.MaxExpensive < 0 {
		return fmt.Errorf("strategy.max_expensive must be >= 0")
	}
	if c.Tools.SemanticMaxFiles < 0 {
		return errors.New("tools.semantic_max_files must be >= 0")
	}
	if c.Tools.SemanticMaxFileBytes < 0 {
		return errors.New("tools.semantic_max_file_bytes must be >= 0")
	}

	switch strings.ToLower(strings.TrimSpace(c.Server.Transport)) {
	case "", "connect", "ndjson":
	default:
		return fmt.Errorf("server.transport must be one of connect or ndjson, got %q", c.Server.Transport)
	}

	return nil
}
