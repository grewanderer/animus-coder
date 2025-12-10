package rpc

// StartSession initializes a new session handshake.
type StartSession struct {
	SessionID string `json:"session_id"`
	Model     string `json:"model,omitempty"`
}

// UserInput represents a chunk of user-supplied text.
type UserInput struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
}

// RunTaskRequest is the top-level request for starting an agent task.
type RunTaskRequest struct {
	SessionID     string     `json:"session_id"`
	CorrelationID string     `json:"correlation_id,omitempty"`
	Model         string     `json:"model,omitempty"`
	PlannerModel  string     `json:"planner_model,omitempty"`
	CriticModel   string     `json:"critic_model,omitempty"`
	Prompt        string     `json:"prompt"`
	Tools         []ToolCall `json:"tools,omitempty"`
	ContextPaths  []string   `json:"context_paths,omitempty"`
}

// RunTaskEvent streams back progress from the daemon.
type RunTaskEvent struct {
	Type          string `json:"type"` // token|message|error|done|tool|plan|reflect|test
	SessionID     string `json:"session_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Token         string `json:"token,omitempty"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
	Done          bool   `json:"done,omitempty"`
	Step          int    `json:"step,omitempty"`
	FinishReason  string `json:"finish_reason,omitempty"`
	ToolName      string `json:"tool_name,omitempty"`
	ToolOutput    string `json:"tool_output,omitempty"`
	ExitCode      int    `json:"exit_code,omitempty"`
	Critique      map[string]interface{} `json:"critique,omitempty"`
	TestSummary   string   `json:"test_summary,omitempty"`
	FailingTests  []string `json:"failing_tests,omitempty"`
	TestAttempts  int      `json:"test_attempts,omitempty"`
}

// ToolCall describes an invocation request.
type ToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// RunTaskStreamRequest is the bidirectional stream payload for Connect RPC.
// The first message must contain the Run task; subsequent messages can carry control signals.
type RunTaskStreamRequest struct {
	Run           *RunTaskRequest `json:"run,omitempty"`
	Cancel        bool            `json:"cancel,omitempty"`
	SessionID     string          `json:"session_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
}
