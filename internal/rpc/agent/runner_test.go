package agent

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/animus-coder/animus-coder/internal/agent"
	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm"
	llmmock "github.com/animus-coder/animus-coder/internal/llm/mock"
	"github.com/animus-coder/animus-coder/internal/rpc"
	"github.com/animus-coder/animus-coder/internal/semantic"
	"github.com/animus-coder/animus-coder/internal/tools"
)

func newTestAgent() *agent.Agent {
	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{
				Message:      llm.ChatMessage{Role: llm.RoleAssistant, Content: "ok [done]"},
				FinishReason: "stop",
			}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)
	return agent.New(reg, config.AgentConfig{MaxSteps: 2})
}

func TestAgentRunnerStopsOnDone(t *testing.T) {
	ar := &AgentRunner{Agent: newTestAgent()}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)

	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "s1", Prompt: "p"})
	require.NoError(t, err)

	var doneEvents int
	for ev := range ch {
		if ev.Type == "done" {
			doneEvents++
			require.Equal(t, "stop", ev.FinishReason)
		}
	}
	require.Equal(t, 1, doneEvents)
}

func TestAgentRunnerRespectsMaxSteps(t *testing.T) {
	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{
				Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "loop"},
			}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)
	a := agent.New(reg, config.AgentConfig{MaxSteps: 2})
	ar := &AgentRunner{Agent: a}

	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "s2", Prompt: "p"})
	require.NoError(t, err)

	var done rpc.RunTaskEvent
	for ev := range ch {
		if ev.Type == "done" {
			done = ev
		}
	}
	require.Equal(t, "max_steps", done.FinishReason)
	require.Equal(t, 2, done.Step)
}

func TestIsResponseDone(t *testing.T) {
	resp := agent.Response{Message: llm.ChatMessage{Content: "all good [done]"}}
	require.True(t, isResponseDone(resp))
	require.False(t, isResponseDone(agent.Response{Message: llm.ChatMessage{Content: "keep going"}}))
	require.True(t, isResponseDone(agent.Response{FinishReason: "stop"}))
}

func TestAgentRunnerExecutesToolCalls(t *testing.T) {
	tmp := t.TempDir()
	fsTool, err := tools.NewFilesystem(tmp, true)
	require.NoError(t, err)
	reg := tools.NewRegistry(fsTool, nil, nil, nil)

	ar := &AgentRunner{Agent: newTestAgent(), Tools: reg}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)

	path := filepath.Join("file.txt")
	ch, err := ar.Run(req, rpc.RunTaskRequest{
		SessionID: "s-tool",
		Prompt:    "p",
		Tools: []rpc.ToolCall{
			{Name: "fs.write_file", Args: map[string]interface{}{"path": path, "content": "hello"}},
			{Name: "fs.read_file", Args: map[string]interface{}{"path": path}},
		},
	})
	require.NoError(t, err)

	var toolEvents int
	var doneSeen bool
	for ev := range ch {
		if ev.Type == "tool" {
			toolEvents++
			require.NotEmpty(t, ev.ToolOutput)
		}
		if ev.Type == "done" {
			doneSeen = true
		}
	}
	require.Equal(t, 2, toolEvents)
	require.True(t, doneSeen)
}

func TestAgentRunnerEmitsPlanEvent(t *testing.T) {
	reg := llm.NewRegistry()
	callCount := 0
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				require.Contains(t, req.Messages[0].Content, "plan")
				return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "1) inspect\n2) edit"}}, nil
			}
			var hasPlan bool
			for _, m := range req.Messages {
				if strings.Contains(m.Content, "inspect") {
					hasPlan = true
					break
				}
			}
			require.True(t, hasPlan, "agent run should include planned steps")
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "ok [done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)
	a := agent.New(reg, config.AgentConfig{MaxSteps: 1, EnablePlan: true})

	ar := &AgentRunner{Agent: a}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "plan1", Prompt: "do work"})
	require.NoError(t, err)

	var planSeen, doneSeen bool
	for ev := range ch {
		if ev.Type == "plan" {
			planSeen = true
			require.Contains(t, ev.Message, "inspect")
		}
		if ev.Type == "done" {
			doneSeen = true
			require.Equal(t, "stop", ev.FinishReason)
		}
	}

	require.True(t, planSeen)
	require.True(t, doneSeen)
	require.Equal(t, 2, callCount)
}

func TestAgentRunnerRunsTestsOnDone(t *testing.T) {
	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)
	a := agent.New(reg, config.AgentConfig{MaxSteps: 1, EnableTestRun: true, TestCommand: "echo ok"})

	term := &tools.Terminal{
		AllowExecution: true,
		Allowed:        []string{"echo"},
	}
	toolReg := tools.NewRegistry(nil, term, nil, nil)

	ar := &AgentRunner{Agent: a, Tools: toolReg}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "test-run", Prompt: "run tests"})
	require.NoError(t, err)

	var testSeen, doneSeen bool
	for ev := range ch {
		if ev.Type == "test" {
			testSeen = true
			require.Equal(t, 0, ev.ExitCode)
			require.Contains(t, ev.Message, "ok")
			require.Equal(t, "", ev.TestSummary)
			require.Equal(t, 1, ev.TestAttempts)
		}
		if ev.Type == "done" {
			doneSeen = true
			require.Equal(t, "stop", ev.FinishReason)
		}
	}
	require.True(t, testSeen)
	require.True(t, doneSeen)
}

func TestAgentRunnerParsesFailingTests(t *testing.T) {
	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)
	a := agent.New(reg, config.AgentConfig{MaxSteps: 1, EnableTestRun: true, TestCommand: "echo \"--- FAIL: TestFoo\""})

	term := &tools.Terminal{
		AllowExecution: true,
		Allowed:        []string{"echo"},
	}
	toolReg := tools.NewRegistry(nil, term, nil, nil)

	ar := &AgentRunner{Agent: a, Tools: toolReg}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "test-run-fail", Prompt: "run tests"})
	require.NoError(t, err)

	var testEvt rpc.RunTaskEvent
	for ev := range ch {
		if ev.Type == "test" {
			testEvt = ev
		}
	}
	require.Contains(t, testEvt.FailingTests, "TestFoo")
	require.Contains(t, testEvt.TestSummary, "Failing tests")
}

func TestAgentRunnerLoadsContextFiles(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "ctx.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello context"), 0o644))

	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			userMsg := req.Messages[len(req.Messages)-1].Content
			require.Contains(t, userMsg, "hello context")
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	fsTool, err := tools.NewFilesystem(tmp, true)
	require.NoError(t, err)
	ar := &AgentRunner{
		Agent: regAgentWithConfig(reg, config.AgentConfig{MaxContextBytes: 20}),
		Tools: tools.NewRegistry(fsTool, nil, nil, nil),
	}

	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{
		SessionID:    "ctx-run",
		Prompt:       "use context",
		ContextPaths: []string{"ctx.txt"},
	})
	require.NoError(t, err)

	var doneSeen bool
	for ev := range ch {
		if ev.Type == "done" {
			doneSeen = true
		}
	}
	require.True(t, doneSeen)
}

func TestAgentRunnerAutoLoadsContextFromPrompt(t *testing.T) {
	tmp := t.TempDir()
	mainPath := filepath.Join(tmp, "main.go")
	require.NoError(t, os.WriteFile(mainPath, []byte("package main\n// auto context"), 0o644))

	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			userMsg := req.Messages[len(req.Messages)-1].Content
			require.Contains(t, userMsg, "package main")
			require.Contains(t, userMsg, "main.go")
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	fsTool, err := tools.NewFilesystem(tmp, true)
	require.NoError(t, err)

	ar := &AgentRunner{
		Agent: regAgentWithConfig(reg, config.AgentConfig{MaxSteps: 1, MaxContextBytes: 1024}),
		Tools: tools.NewRegistry(fsTool, nil, nil, nil),
	}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{
		SessionID: "auto-ctx",
		Prompt:    "review main.go and explain its purpose",
	})
	require.NoError(t, err)

	var doneSeen bool
	for ev := range ch {
		if ev.Type == "done" {
			doneSeen = true
		}
	}
	require.True(t, doneSeen)
}

func TestAgentRunnerLoadsDirectoryStructure(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "dir")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0o644))

	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			userMsg := req.Messages[len(req.Messages)-1].Content
			require.Contains(t, userMsg, "dir (structure)")
			require.Contains(t, userMsg, "file.txt")
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	fsTool, err := tools.NewFilesystem(tmp, true)
	require.NoError(t, err)
	ar := &AgentRunner{
		Agent: regAgentWithConfig(reg, config.AgentConfig{MaxSteps: 1, MaxContextBytes: 2048}),
		Tools: tools.NewRegistry(fsTool, nil, nil, nil),
	}

	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{
		SessionID:    "dir-ctx",
		Prompt:       "look into dir for details",
		ContextPaths: []string{"dir"},
	})
	require.NoError(t, err)

	var doneSeen bool
	for ev := range ch {
		if ev.Type == "done" {
			doneSeen = true
		}
	}
	require.True(t, doneSeen)
}

func TestAgentRunnerReflectionUsesToolAndTestContext(t *testing.T) {
	tmp := t.TempDir()
	fsTool, err := tools.NewFilesystem(tmp, true)
	require.NoError(t, err)
	term := &tools.Terminal{AllowExecution: true, Allowed: []string{"echo"}}
	toolReg := tools.NewRegistry(fsTool, term, nil, nil)

	reg := llm.NewRegistry()
	callCount := 0
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			callCount++
			if strings.Contains(req.Messages[0].Content, "reflection") {
				userMsg := req.Messages[len(req.Messages)-1].Content
				require.Contains(t, userMsg, "fs.read_file")
				require.Contains(t, userMsg, "ref-data")
				require.Contains(t, userMsg, "tests failing")
				return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "reflection ok"}}, nil
			}
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	ar := &AgentRunner{
		Agent: regAgentWithConfig(reg, config.AgentConfig{
			MaxSteps:      1,
			EnableReflect: true,
			EnableTestRun: true,
			TestCommand:   "echo tests failing",
		}),
		Tools: toolReg,
	}

	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{
		SessionID: "reflect-step",
		Prompt:    "run with tools",
		Tools: []rpc.ToolCall{
			{Name: "fs.write_file", Args: map[string]interface{}{"path": "data.txt", "content": "ref-data"}},
			{Name: "fs.read_file", Args: map[string]interface{}{"path": "data.txt"}},
		},
	})
	require.NoError(t, err)

	var reflectSeen, doneSeen bool
	for ev := range ch {
		if ev.Type == "reflect" {
			reflectSeen = true
		}
		if ev.Type == "done" {
			doneSeen = true
		}
	}

	require.True(t, reflectSeen)
	require.True(t, doneSeen)
	require.Equal(t, 2, callCount)
}

func TestAgentRunnerSemanticContext(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "hidden.txt"), []byte("special needle content"), 0o644))

	fsTool, err := tools.NewFilesystem(tmp, true)
	require.NoError(t, err)
	sem := semantic.NewEngine(fsTool, 10, 1024)

	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			userMsg := req.Messages[len(req.Messages)-1].Content
			require.Contains(t, userMsg, "special needle content")
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	ar := &AgentRunner{
		Agent: regAgentWithConfig(reg, config.AgentConfig{MaxSteps: 1}),
		Tools: tools.NewRegistry(fsTool, nil, nil, sem),
	}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "sem-ctx", Prompt: "find the hidden needle"})
	require.NoError(t, err)

	var doneSeen bool
	for ev := range ch {
		if ev.Type == "done" {
			doneSeen = true
		}
	}
	require.True(t, doneSeen)
}

func TestAgentRunnerBlocksOnCritique(t *testing.T) {
	reg := llm.NewRegistry()
	call := 0
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			call++
			if strings.Contains(req.Messages[0].Content, "reflection") {
				return llm.ChatResponse{
					Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: `{"quality":"poor","issues":["bad"],"recommendations":["fix"],"block_apply":true}`},
				}, nil
			}
			return llm.ChatResponse{
				Message:      llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"},
				FinishReason: "stop",
			}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	ar := &AgentRunner{
		Agent: regAgentWithConfig(reg, config.AgentConfig{
			MaxSteps:      2,
			EnableReflect: true,
		}),
	}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "block-reflect", Prompt: "do something risky"})
	require.NoError(t, err)

	var doneEvt rpc.RunTaskEvent
	var reflectEvt rpc.RunTaskEvent
	var haltedMsg bool
	for ev := range ch {
		if ev.Type == "reflect" {
			reflectEvt = ev
		}
		if ev.Type == "message" && strings.Contains(ev.Message, "halted by reflection") {
			haltedMsg = true
		}
		if ev.Type == "done" {
			doneEvt = ev
		}
	}

	require.Equal(t, "blocked_by_reflect", doneEvt.FinishReason)
	require.True(t, haltedMsg)
	require.NotNil(t, reflectEvt.Critique)
	require.True(t, reflectEvt.Critique["block_apply"].(bool))
	require.Equal(t, 2, call)
}

func TestAgentRunnerPolicyNeverBlocks(t *testing.T) {
	reg := llm.NewRegistry()
	reg.RegisterProvider("mock", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			if strings.Contains(req.Messages[0].Content, "reflection") {
				return llm.ChatResponse{
					Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: `{"block_apply":true}`},
				}, nil
			}
			return llm.ChatResponse{
				Message:      llm.ChatMessage{Role: llm.RoleAssistant, Content: "[done]"},
				FinishReason: "stop",
			}, nil
		},
	})
	reg.RegisterModel("default", llm.ModelRoute{Provider: "mock", Model: "m"}, true)

	ar := &AgentRunner{
		Agent: regAgentWithConfig(reg, config.AgentConfig{
			MaxSteps:         1,
			EnableReflect:    true,
			ReflectionPolicy: "never_block",
		}),
	}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "never-block", Prompt: "do work"})
	require.NoError(t, err)

	var doneEvt rpc.RunTaskEvent
	var reflectSeen bool
	for ev := range ch {
		if ev.Type == "done" {
			doneEvt = ev
		}
		if ev.Type == "reflect" {
			reflectSeen = true
			require.NotNil(t, ev.Critique)
		}
	}
	require.Equal(t, "stop", doneEvt.FinishReason)
	require.True(t, reflectSeen)
}

func TestAgentRunnerFallsBackOnCoderError(t *testing.T) {
	reg := llm.NewRegistry()
	primaryCalls := 0
	fallbackCalls := 0
	reg.RegisterProvider("fail", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			primaryCalls++
			return llm.ChatResponse{}, fmt.Errorf("boom")
		},
	})
	reg.RegisterProvider("ok", &llmmock.Provider{
		ChatFn: func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			fallbackCalls++
			return llm.ChatResponse{Message: llm.ChatMessage{Role: llm.RoleAssistant, Content: "fallback [done]"}, FinishReason: "stop"}, nil
		},
	})
	reg.RegisterModel("primary", llm.ModelRoute{Provider: "fail", Model: "m1"}, true)
	reg.RegisterModel("backup", llm.ModelRoute{Provider: "ok", Model: "m2"}, false)

	a := agent.New(reg, config.AgentConfig{MaxSteps: 1})
	metrics := &fakeMetrics{}
	ar := &AgentRunner{
		Agent:    a,
		Metrics:  metrics,
		Strategy: agent.NewStrategyEngine(reg, config.StrategyConfig{CoderModel: "primary", Fallbacks: []string{"backup"}}),
	}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	ch, err := ar.Run(req, rpc.RunTaskRequest{SessionID: "fallback", Prompt: "do work"})
	require.NoError(t, err)

	var doneSeen, messageSeen bool
	for ev := range ch {
		if ev.Type == "message" {
			messageSeen = true
			require.Contains(t, ev.Message, "fallback")
		}
		if ev.Type == "done" {
			doneSeen = true
			require.Equal(t, "stop", ev.FinishReason)
		}
	}

	require.True(t, doneSeen)
	require.True(t, messageSeen)
	require.Equal(t, 1, primaryCalls)
	require.Equal(t, 1, fallbackCalls)
	require.Contains(t, metrics.failures, "coder:primary")
	require.Contains(t, metrics.usage, "coder:backup")
}

func regAgentWithConfig(reg *llm.Registry, cfg config.AgentConfig) *agent.Agent {
	return agent.New(reg, cfg)
}

type fakeMetrics struct {
	usage    []string
	failures []string
}

func (f *fakeMetrics) RecordAgentRun(finishReason string, duration time.Duration, tokenCount int) {}
func (f *fakeMetrics) RecordModelUsage(role, model string) {
	f.usage = append(f.usage, role+":"+model)
}
func (f *fakeMetrics) RecordModelFailure(role, model string) {
	f.failures = append(f.failures, role+":"+model)
}
