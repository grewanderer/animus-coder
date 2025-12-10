package agent

import (
	"testing"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm"
	llmmock "github.com/animus-coder/animus-coder/internal/llm/mock"
)

func TestStrategyResolvesRoles(t *testing.T) {
	reg := llm.NewRegistry()
	reg.RegisterProvider("p", &llmmock.Provider{})
	reg.RegisterModel("plan-model", llm.ModelRoute{Provider: "p", Model: "m1"}, true)
	reg.RegisterModel("code-model", llm.ModelRoute{Provider: "p", Model: "m2"}, false)
	reg.RegisterModel("crit-model", llm.ModelRoute{Provider: "p", Model: "m3"}, false)

	engine := NewStrategyEngine(reg, config.StrategyConfig{
		PlannerModel: "plan-model",
		CoderModel:   "code-model",
		CriticModel:  "crit-model",
	})

	_, route, err := engine.ResolveModel("planner", "")
	if err != nil || route.Name != "plan-model" {
		t.Fatalf("expected planner model, got %s err=%v", route.Name, err)
	}
	_, route, err = engine.ResolveModel("coder", "")
	if err != nil || route.Name != "code-model" {
		t.Fatalf("expected coder model, got %s err=%v", route.Name, err)
	}
	_, route, err = engine.ResolveModel("critic", "")
	if err != nil || route.Name != "crit-model" {
		t.Fatalf("expected critic model, got %s err=%v", route.Name, err)
	}
}

func TestStrategyFallsBackWhenBudgetExceeded(t *testing.T) {
	reg := llm.NewRegistry()
	reg.RegisterProvider("p", &llmmock.Provider{})
	reg.RegisterModel("expensive-model", llm.ModelRoute{Provider: "p", Model: "m1"}, true)
	reg.RegisterModel("cheap-model", llm.ModelRoute{Provider: "p", Model: "m2"}, false)
	reg.MarkExpensive("expensive-model", true)

	engine := NewStrategyEngine(reg, config.StrategyConfig{
		CoderModel:   "expensive-model",
		Fallbacks:    []string{"cheap-model"},
		MaxExpensive: 1,
		DefaultModel: "cheap-model",
	})

	_, _, chosen, isExp, err := engine.PickWithBudget("coder", "", 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if chosen != "cheap-model" {
		t.Fatalf("expected fallback cheap-model, got %s", chosen)
	}
	if isExp {
		t.Fatalf("expected fallback not expensive")
	}
}
