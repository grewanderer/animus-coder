package agent

import (
	"strings"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm"
)

// StrategyEngine chooses models for different phases.
type StrategyEngine struct {
	registry *llm.Registry
	cfg      config.StrategyConfig
}

// NewStrategyEngine builds a strategy selector.
func NewStrategyEngine(reg *llm.Registry, cfg config.StrategyConfig) *StrategyEngine {
	return &StrategyEngine{registry: reg, cfg: cfg}
}

// ResolveModel picks a model id based on role/hint; falls back to default registry resolution.
func (s *StrategyEngine) ResolveModel(role string, override string) (llm.Provider, llm.ModelRoute, error) {
	if s == nil || s.registry == nil {
		return nil, llm.ModelRoute{}, nil
	}
	role = strings.ToLower(strings.TrimSpace(role))
	modelID := firstNonEmpty(
		override,
		s.cfg.Overrides[role],
		roleModel(role, s.cfg),
		s.cfg.DefaultModel,
	)
	if modelID != "" {
		if p, route, err := s.registry.Resolve(modelID); err == nil {
			return p, route, nil
		}
	}
	for _, fb := range s.cfg.Fallbacks {
		if p, route, err := s.registry.Resolve(fb); err == nil {
			return p, route, nil
		}
	}
	return s.registry.Resolve("")
}

// PickWithBudget chooses a model honoring max_expensive; expensiveUsed is the count so far.
func (s *StrategyEngine) PickWithBudget(role, override string, expensiveUsed int) (llm.Provider, llm.ModelRoute, string, bool, error) {
	prov, route, err := s.ResolveModel(role, override)
	if err != nil {
		return nil, llm.ModelRoute{}, "", false, err
	}
	if prov == nil {
		return nil, llm.ModelRoute{}, "", false, nil
	}
	chosen := route.Name
	isExp := s.registry.IsExpensive(chosen)
	if s.cfg.MaxExpensive > 0 && isExp && expensiveUsed >= s.cfg.MaxExpensive {
		for _, fb := range s.cfg.Fallbacks {
			p, r, err := s.registry.Resolve(fb)
			if err != nil {
				continue
			}
			chosen = r.Name
			prov = p
			route = r
			isExp = s.registry.IsExpensive(chosen)
			break
		}
	}
	// If we still have an expensive model and budget exceeded with no fallback, drop to default if available.
	if s.cfg.MaxExpensive > 0 && isExp && expensiveUsed >= s.cfg.MaxExpensive && s.cfg.DefaultModel != "" {
		if p, r, err := s.registry.Resolve(s.cfg.DefaultModel); err == nil {
			chosen = r.Name
			prov = p
			route = r
			isExp = s.registry.IsExpensive(chosen)
		}
	}
	return prov, route, chosen, isExp, nil
}

// NextFallback returns the next fallback model id different from current.
func (s *StrategyEngine) NextFallback(current string) string {
	for _, fb := range s.cfg.Fallbacks {
		if strings.TrimSpace(fb) == "" || fb == current {
			continue
		}
		return fb
	}
	return ""
}

func roleModel(role string, cfg config.StrategyConfig) string {
	switch role {
	case "planner":
		return cfg.PlannerModel
	case "critic", "reflect":
		return cfg.CriticModel
	default:
		return cfg.CoderModel
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
