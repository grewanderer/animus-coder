package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm"
	llmmock "github.com/animus-coder/animus-coder/internal/llm/mock"
)

func TestAgentPlanCachesAndInjectsIntoRun(t *testing.T) {
	reg := llm.NewRegistry()
	callCount := 0
	mockProvider := &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				require.Contains(t, req.Messages[0].Content, "plan")
				return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "1) inspect\n2) edit\n3) verify"}}, nil
			}
			var hasPlan bool
			for _, m := range req.Messages {
				if strings.Contains(m.Content, "inspect") {
					hasPlan = true
					break
				}
			}
			require.True(t, hasPlan, "run prompt should include cached plan")
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "done"}}, nil
		},
	}
	reg.RegisterProvider("mock", mockProvider)
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	a := New(reg, config.AgentConfig{EnablePlan: true})

	plan, err := a.Plan(context.Background(), Request{SessionID: "p1", Prompt: "build feature"})
	require.NoError(t, err)
	require.Contains(t, plan, "inspect")

	_, err = a.Run(context.Background(), Request{SessionID: "p1", Prompt: "execute plan"})
	require.NoError(t, err)
	require.Equal(t, 2, callCount, "plan should be cached")

	_, err = a.Plan(context.Background(), Request{SessionID: "p1", Prompt: "ignored"})
	require.NoError(t, err)
	require.Equal(t, 2, callCount, "cached plan should avoid extra calls")
}

func TestAgentReflectionFeedsNextRun(t *testing.T) {
	reg := llm.NewRegistry()
	callCount := 0
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			switch callCount {
			case 1:
				// Initial run.
				return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "made change"}}, nil
			case 2:
				require.Contains(t, req.Messages[0].Content, "reflection")
				require.Contains(t, req.Messages[1].Content, "made change")
				return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "reflect: add tests"}}, nil
			default:
				var hasReflection bool
				for _, m := range req.Messages {
					if strings.Contains(m.Content, "reflect: add tests") {
						hasReflection = true
						break
					}
				}
				require.True(t, hasReflection, "reflection should be in history")
				return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "final"}}, nil
			}
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	a := New(reg, config.AgentConfig{EnableReflect: true})

	resp, err := a.Run(context.Background(), Request{SessionID: "r1", Prompt: "task"})
	require.NoError(t, err)
	reflection, err := a.Reflect(context.Background(), Request{SessionID: "r1", Prompt: "task"}, resp, ReflectionContext{})
	require.NoError(t, err)
	require.Contains(t, reflection, "reflect")

	_, err = a.Run(context.Background(), Request{SessionID: "r1", Prompt: "next"})
	require.NoError(t, err)
	require.Equal(t, 3, callCount)
}

func TestAgentUsesRegistryAndHistory(t *testing.T) {
	reg := llm.NewRegistry()
	mockProvider := &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			require.Equal(t, "gpt-test", req.Model)
			require.Len(t, req.Messages, 2) // system + user on first call
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "first"}}, nil
		},
	}
	reg.RegisterProvider("mock", mockProvider)
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "gpt-test"}, true)

	a := New(reg, config.AgentConfig{MaxSteps: 4})

	resp, err := a.Run(context.Background(), Request{
		SessionID: "s1",
		Prompt:    "hello",
	})
	require.NoError(t, err)
	require.Equal(t, "first", resp.Message.Content)

	// Second call should include previous assistant reply in history.
	mockProvider.ChatFn = func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
		require.Len(t, req.Messages, 4) // system + user + assistant + new user
		return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "second"}}, nil
	}

	resp, err = a.Run(context.Background(), Request{
		SessionID: "s1",
		Prompt:    "next",
	})
	require.NoError(t, err)
	require.Equal(t, "second", resp.Message.Content)
}

func TestAgentInjectsContextIntoPrompt(t *testing.T) {
	reg := llm.NewRegistry()
	mockProvider := &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			userMsg := req.Messages[len(req.Messages)-1].Content
			require.Contains(t, userMsg, "main.go")
			require.Contains(t, userMsg, "package main")
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "ok"}}, nil
		},
	}
	reg.RegisterProvider("mock", mockProvider)
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "gpt-test"}, true)

	a := New(reg, config.AgentConfig{})

	_, err := a.Run(context.Background(), Request{
		SessionID: "ctx1",
		Prompt:    "summarize",
		Context: []ContextFile{
			{Path: "main.go", Content: "package main"},
		},
	})
	require.NoError(t, err)
}

func TestPickers(t *testing.T) {
	require.Equal(t, 0.5, pickTemperature(0.5, 0.2))
	require.Equal(t, 0.2, pickTemperature(0, 0.2))
	require.Equal(t, 0.2, pickTemperature(0, 0))

	require.Equal(t, 256, pickMaxTokens(256, 128))
	require.Equal(t, 128, pickMaxTokens(0, 128))
	require.Equal(t, 0, pickMaxTokens(0, 0))
}

func TestSystemPromptIncludesInstruction(t *testing.T) {
	sp := buildSystemPrompt(config.AgentConfig{})
	require.True(t, strings.Contains(sp, "MyCodex"))
}

func TestMaxStepsDefault(t *testing.T) {
	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)
	a := New(reg, config.AgentConfig{})
	require.Equal(t, 1, a.MaxSteps())
	a2 := New(reg, config.AgentConfig{MaxSteps: 5})
	require.Equal(t, 5, a2.MaxSteps())
}
