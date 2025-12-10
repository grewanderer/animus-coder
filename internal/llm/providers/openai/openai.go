package openai

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

// Provider implements an OpenAI-compatible chat provider.
type Provider struct {
	name    string
	client  *http.Client
	baseURL string
	apiKey  string
}

// NewProvider constructs a Provider with sane defaults.
func NewProvider(name, baseURL, apiKey string, timeout time.Duration) *Provider {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Provider{
		name:    name,
		client:  &http.Client{Timeout: timeout},
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
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

	body := openAIChatRequest{
		Model:       model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	res, err := p.client.Do(httpReq)
	if err != nil {
		return llm.ChatResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return llm.ChatResponse{}, fmt.Errorf("openai: status %d: %s", res.StatusCode, string(b))
	}

	var resp openAIChatResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return llm.ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return llm.ChatResponse{}, fmt.Errorf("openai: empty choices")
	}

	msg := resp.Choices[0].Message
	return llm.ChatResponse{
		Message: llm.ChatMessage{
			Role:    llm.Role(msg.Role),
			Content: msg.Content,
		},
		FinishReason: resp.Choices[0].FinishReason,
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
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

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
	Name    string `json:"name,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Index        int           `json:"index"`
		FinishReason string        `json:"finish_reason"`
		Message      openAIMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func toOpenAIMessages(msgs []llm.ChatMessage) []openAIMessage {
	out := make([]openAIMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, openAIMessage{
			Role:    string(m.Role),
			Content: m.Content,
			Name:    m.Name,
		})
	}
	return out
}
