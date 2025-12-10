package llm

import (
	"context"
	"encoding/json"
)

// Role is the message role used in chat exchanges.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ChatMessage represents a single message exchanged with the model.
type ChatMessage struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content,omitempty"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	Metadata   interface{} `json:"metadata,omitempty"`
}

// ToolCall describes a model-initiated tool invocation.
type ToolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolFunctionCall `json:"function,omitempty"`
}

// ToolFunctionCall is the function call payload for a tool request.
type ToolFunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ChatRequest is the input for chat providers.
type ChatRequest struct {
	Model       string
	Messages    []ChatMessage
	MaxTokens   int
	Temperature float64
	Stream      bool
}

// Usage captures token accounting.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ChatResponse is the result of a chat completion.
type ChatResponse struct {
	Message      ChatMessage
	FinishReason string
	Usage        Usage
	RawResponse  interface{}
	ProviderName string
	Model        string
}

// StreamChunk is emitted during streaming responses.
type StreamChunk struct {
	Content      string
	FinishReason string
	Err          error
}

// Provider defines the contract for LLM providers.
type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, <-chan error)
}
