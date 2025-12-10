package configbuilder

import (
	"fmt"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm"
	llmollama "github.com/animus-coder/animus-coder/internal/llm/providers/ollama"
	llmopenai "github.com/animus-coder/animus-coder/internal/llm/providers/openai"
)

// BuildRegistryFromConfig constructs a registry and providers from config.
func BuildRegistryFromConfig(cfg *config.Config) (*llm.Registry, error) {
	reg := llm.NewRegistry()

	for name, pCfg := range cfg.Providers {
		p, err := buildProvider(name, pCfg)
		if err != nil {
			return nil, err
		}
		reg.RegisterProvider(name, p)
	}

	for name, mCfg := range cfg.Models {
		reg.RegisterModel(name, llm.ModelRoute{
			Provider:    mCfg.Provider,
			Model:       mCfg.Model,
			Temperature: mCfg.Temperature,
			MaxTokens:   mCfg.MaxTokens,
		}, mCfg.Default)
		if mCfg.Expensive {
			reg.MarkExpensive(name, true)
		}
	}

	if _, _, err := reg.Resolve(""); err != nil {
		return nil, err
	}

	return reg, nil
}

func buildProvider(name string, cfg config.ProviderConfig) (llm.Provider, error) {
	switch cfg.Type {
	case "openai", "openrouter", "vllm", "lmstudio", "custom":
		return llmopenai.NewProvider(name, cfg.BaseURL, cfg.APIKey, cfg.Timeout), nil
	case "ollama":
		return llmollama.NewProvider(name, cfg.BaseURL, cfg.Timeout), nil
	default:
		return nil, fmt.Errorf("unknown provider type %q for provider %s", cfg.Type, name)
	}
}
