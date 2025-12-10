package llm_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm"
	"github.com/animus-coder/animus-coder/internal/llm/configbuilder"
	llmmock "github.com/animus-coder/animus-coder/internal/llm/mock"
)

func TestRegistryResolve(t *testing.T) {
	reg := llm.NewRegistry()
	mockProvider := &llmmock.Provider{NameValue: "mock"}
	reg.RegisterProvider("mock", mockProvider)
	reg.RegisterModel("default", llm.ModelRoute{
		Provider:    "mock",
		Model:       "dummy",
		Temperature: 0.2,
	}, true)

	p, route, err := reg.Resolve("")
	require.NoError(t, err)
	require.Equal(t, mockProvider, p)
	require.Equal(t, "dummy", route.Model)
}

func TestBuildRegistryFromConfig(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"openai": {Type: "openai", BaseURL: "http://example.com"},
		},
		Models: map[string]config.ModelConfig{
			"main": {Provider: "openai", Model: "gpt-4o", Default: true},
		},
	}

	reg, err := configbuilder.BuildRegistryFromConfig(cfg)
	require.NoError(t, err)

	p, _, err := reg.Resolve("main")
	require.NoError(t, err)
	require.Equal(t, "openai", p.Name())
}
