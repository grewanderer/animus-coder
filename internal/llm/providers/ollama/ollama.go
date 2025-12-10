package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/animus-coder/animus-coder/internal/llm"
)

// Provider implements a minimal Ollama chat client.
type Provider struct {
	name    string
	client  *http.Client
	baseURL string
}

// NewProvider constructs an Ollama provider.
func NewProvider(name, baseURL string, timeout time.Duration) *Provider {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	if timeout == 0 {
		timeout = 20 * time.Second
	}

	return &Provider{
		name:    name,
		client:  &http.Client{Timeout: timeout},
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Name returns provider identifier.
func (p *Provider) Name() string {
	return p.name
}

// Chat executes a non-streaming chat completion.
func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		return llm.ChatResponse{}, fmt.Errorf("model is required")
	}

	body := ollamaChatRequest{
		Model:    model,
		Messages: toOllamaMessages(req.Messages),
		Stream:   false,
		Options: map[string]interface{}{
			"temperature": req.Temperature,
			"num_predict": req.MaxTokens,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := p.client.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return llm.ChatResponse{}, fmt.Errorf("ollama: status %d: %s", res.StatusCode, string(b))
	}

	var resp ollamaChatResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return llm.ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}

	return llm.ChatResponse{
		Message: llm.ChatMessage{
			Role:    llm.Role(resp.Message.Role),
			Content: resp.Message.Content,
		},
		FinishReason: "stop",
		RawResponse:  resp,
		ProviderName: p.name,
		Model:        model,
	}, nil
}

// Stream performs chat and emits a single chunk (simulated streaming).
func (p *Provider) Stream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, <-chan error) {
	ch := make(chan llm.StreamChunk, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(errCh)

		resp, err := p.Chat(ctx, req)
		if err != nil {
			errCh <- err
			return
		}
		ch <- llm.StreamChunk{
			Content:      resp.Message.Content,
			FinishReason: resp.FinishReason,
		}
	}()

	return ch, errCh
}

type ollamaChatRequest struct {
	Model    string                 `json:"model"`
	Messages []ollamaMessage        `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

func toOllamaMessages(msgs []llm.ChatMessage) []ollamaMessage {
	out := make([]ollamaMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, ollamaMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}
	return out
}
