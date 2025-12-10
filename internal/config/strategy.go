package config

// StrategyConfig defines per-role model selections and fallbacks.
type StrategyConfig struct {
	DefaultModel string            `mapstructure:"default_model"`
	PlannerModel string            `mapstructure:"planner_model"`
	CoderModel   string            `mapstructure:"coder_model"`
	CriticModel  string            `mapstructure:"critic_model"`
	Overrides    map[string]string `mapstructure:"overrides"` // arbitrary step->model id
	Fallbacks    []string          `mapstructure:"fallbacks"` // ordered fallback model ids
	MaxExpensive int               `mapstructure:"max_expensive"` // limit expensive model uses per run (0=unlimited)
}
