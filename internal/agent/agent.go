package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/animus-coder/animus-coder/internal/config"
	"github.com/animus-coder/animus-coder/internal/llm"
)

// Session stores per-session conversation history.
type Session struct {
	ID      string
	History []llm.ChatMessage
	Plan    string
	// LastReflection holds the latest reflection content to feed into the next turn.
	LastReflection string
}

// Agent orchestrates chat calls with history and context handling.
type Agent struct {
	registry *llm.Registry
	cfg      config.AgentConfig

	mu       sync.Mutex
	sessions map[string]*Session
}

// New creates a new Agent.
func New(registry *llm.Registry, cfg config.AgentConfig) *Agent {
	return &Agent{
		registry: registry,
		cfg:      cfg,
		sessions: make(map[string]*Session),
	}
}

// Run executes a single-turn agent call, maintaining session history.
func (a *Agent) Run(ctx context.Context, req Request) (Response, error) {
	if req.Prompt == "" {
		return Response{}, fmt.Errorf("prompt is required")
	}

	provider, route, err := a.registry.Resolve(req.Model)
	if err != nil {
		return Response{}, err
	}

	if a.cfg.EnablePlan {
		if _, err := a.Plan(ctx, req); err != nil {
			return Response{}, err
		}
	}

	session := a.ensureSession(req.SessionID)

	systemPrompt := buildSystemPrompt(a.cfg)
	userPrompt := buildUserPrompt(req.Prompt, req.Context)

	messages := []llm.ChatMessage{
		{Role: llm.RoleSystem, Content: systemPrompt},
	}
	prevAssistant := a.lastAssistantContent(session)
	if plan := a.sessionPlan(session); plan != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    llm.RoleAssistant,
			Content: "Planned steps:\n" + plan,
		})
	}
	if reflection := a.sessionReflection(session); reflection != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    llm.RoleAssistant,
			Content: "Previous reflection:\n" + reflection,
		})
	}
	messages = append(messages, session.History...)
	messages = append(messages, llm.ChatMessage{Role: llm.RoleUser, Content: userPrompt})

	chatReq := llm.ChatRequest{
		Model:       route.Model,
		Messages:    messages,
		MaxTokens:   pickMaxTokens(a.cfg.MaxTokens, route.MaxTokens),
		Temperature: pickTemperature(a.cfg.Temperature, route.Temperature),
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, chatReq)
	if err != nil {
		return Response{}, err
	}

	a.appendHistory(session, messages[len(messages)-1], resp.Message)

	return Response{
		Message:           resp.Message,
		Route:             route,
		FinishReason:      resp.FinishReason,
		PreviousAssistant: prevAssistant,
	}, nil
}

// Plan builds and caches a short plan for the session when enabled.
func (a *Agent) Plan(ctx context.Context, req Request) (string, error) {
	if !a.cfg.EnablePlan {
		return "", nil
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return "", fmt.Errorf("prompt is required")
	}
	session := a.ensureSession(req.SessionID)
	if plan := a.sessionPlan(session); plan != "" {
		return plan, nil
	}

	provider, route, err := a.registry.Resolve(req.Model)
	if err != nil {
		return "", err
	}

	messages := []llm.ChatMessage{
		{Role: llm.RoleSystem, Content: buildPlanSystemPrompt(a.cfg)},
		{Role: llm.RoleUser, Content: buildPlanUserPrompt(req.Prompt)},
	}

	chatReq := llm.ChatRequest{
		Model:       route.Model,
		Messages:    messages,
		MaxTokens:   pickMaxTokens(a.cfg.MaxTokens, route.MaxTokens),
		Temperature: pickTemperature(a.cfg.Temperature, route.Temperature),
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, chatReq)
	if err != nil {
		return "", err
	}

	plan := strings.TrimSpace(resp.Message.Content)
	a.setPlan(session, plan)
	return plan, nil
}

// Reflect summarizes the last response and records the reflection for future turns.
func (a *Agent) Reflect(ctx context.Context, req Request, last Response, ctxInfo ReflectionContext) (string, error) {
	if !a.cfg.EnableReflect {
		return "", nil
	}
	if strings.TrimSpace(last.Message.Content) == "" {
		return "", fmt.Errorf("last message is required for reflection")
	}

	session := a.ensureSession(req.SessionID)
	provider, route, err := a.registry.Resolve(req.Model)
	if err != nil {
		return "", err
	}

	messages := []llm.ChatMessage{
		{Role: llm.RoleSystem, Content: buildReflectSystemPrompt(a.cfg)},
		{Role: llm.RoleUser, Content: buildReflectUserPrompt(req.Prompt, last.Message.Content, a.sessionPlan(session), ctxInfo)},
	}

	chatReq := llm.ChatRequest{
		Model:       route.Model,
		Messages:    messages,
		MaxTokens:   pickMaxTokens(a.cfg.MaxTokens, route.MaxTokens),
		Temperature: pickTemperature(a.cfg.Temperature, route.Temperature),
		Stream:      false,
	}

	resp, err := provider.Chat(ctx, chatReq)
	if err != nil {
		return "", err
	}

	reflection := strings.TrimSpace(resp.Message.Content)
	a.setReflection(session, reflection)
	a.appendReflection(session, reflection)
	return reflection, nil
}

// MaxSteps returns configured maximum steps (>0).
func (a *Agent) MaxSteps() int {
	if a.cfg.MaxSteps > 0 {
		return a.cfg.MaxSteps
	}
	return 1
}

// PlanningEnabled reports whether planning step is enabled.
func (a *Agent) PlanningEnabled() bool {
	return a.cfg.EnablePlan
}

// ReflectionEnabled reports whether reflection step is enabled.
func (a *Agent) ReflectionEnabled() bool {
	return a.cfg.EnableReflect
}

// ReflectionPolicy returns how to handle critique outcomes.
func (a *Agent) ReflectionPolicy() string {
	if strings.TrimSpace(a.cfg.ReflectionPolicy) == "" {
		return "block_on_critical"
	}
	return strings.ToLower(strings.TrimSpace(a.cfg.ReflectionPolicy))
}

// TestRunEnabled reports whether automated test execution is enabled.
func (a *Agent) TestRunEnabled() bool {
	return a.cfg.EnableTestRun && strings.TrimSpace(a.cfg.TestCommand) != ""
}

// TestCommand returns configured test command.
func (a *Agent) TestCommand() string {
	return a.cfg.TestCommand
}

// TestRetries returns configured retry attempts for tests.
func (a *Agent) TestRetries() int {
	if a.cfg.TestRetries < 0 {
		return 0
	}
	return a.cfg.TestRetries
}

// TestTimeoutSeconds returns per-test timeout in seconds (0 = default).
func (a *Agent) TestTimeoutSeconds() int {
	if a.cfg.TestTimeoutSeconds < 0 {
		return 0
	}
	return a.cfg.TestTimeoutSeconds
}

// MaxContextBytes returns the limit for aggregated context bytes (0 = unlimited).
func (a *Agent) MaxContextBytes() int {
	if a.cfg.MaxContextBytes < 0 {
		return 0
	}
	return a.cfg.MaxContextBytes
}

// EnableSelfDiff reports whether reflection should receive self-diff context.
func (a *Agent) EnableSelfDiff() bool {
	return a.cfg.EnableSelfDiff
}

func (a *Agent) ensureSession(id string) *Session {
	a.mu.Lock()
	defer a.mu.Unlock()

	if id == "" {
		id = fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}

	if s, ok := a.sessions[id]; ok {
		return s
	}
	s := &Session{ID: id, History: make([]llm.ChatMessage, 0, 8)}
	a.sessions[id] = s
	return s
}

func (a *Agent) appendHistory(s *Session, userMsg llm.ChatMessage, assistantMsg llm.ChatMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()

	s.History = append(s.History, userMsg, assistantMsg)
}

func (a *Agent) lastAssistantContent(s *Session) string {
	if s == nil {
		return ""
	}
	for i := len(s.History) - 1; i >= 0; i-- {
		if s.History[i].Role == llm.RoleAssistant {
			return s.History[i].Content
		}
	}
	return ""
}

func (a *Agent) setPlan(s *Session, plan string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	s.Plan = plan
}

func (a *Agent) sessionPlan(s *Session) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return s.Plan
}

func (a *Agent) setReflection(s *Session, reflection string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	s.LastReflection = reflection
}

func (a *Agent) sessionReflection(s *Session) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return s.LastReflection
}

func (a *Agent) appendReflection(s *Session, reflection string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	s.History = append(s.History, llm.ChatMessage{
		Role:    llm.RoleAssistant,
		Content: "Reflection: " + reflection,
	})
}

func pickTemperature(agentTemp float64, routeTemp float64) float64 {
	if agentTemp > 0 {
		return agentTemp
	}
	if routeTemp > 0 {
		return routeTemp
	}
	return 0.2
}

func pickMaxTokens(agentMax int, routeMax int) int {
	if agentMax > 0 {
		return agentMax
	}
	if routeMax > 0 {
		return routeMax
	}
	return 0
}
