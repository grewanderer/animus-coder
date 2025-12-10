package agent

import "github.com/animus-coder/animus-coder/internal/llm"

// ContextFile represents contextual file content passed to the agent.
type ContextFile struct {
	Path    string
	Content string
}

// Request is a single agent invocation.
type Request struct {
	SessionID string
	Model     string
	Prompt    string
	Context   []ContextFile
}

// Response wraps the model response and route metadata.
type Response struct {
	Message           llm.ChatMessage
	Route             llm.ModelRoute
	FinishReason      string
	PreviousAssistant string
}

// ToolObservation captures a single tool invocation result for reflection.
type ToolObservation struct {
	Name   string
	Output string
	Error  string
}

// TestObservation captures the outcome of an automated test run.
type TestObservation struct {
	Command  string
	Output   string
	ExitCode int
	Error    string
	Summary  string
	Failing  []string
	Attempts int
}

// ReflectionContext carries execution artefacts for the reflection phase.
type ReflectionContext struct {
	Tools    []ToolObservation
	Test     *TestObservation
	SelfDiff string
}
