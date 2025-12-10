package mock

import (
	"context"

	"github.com/animus-coder/animus-coder/internal/llm"
)

// Provider is a test double implementing llm.Provider.
type Provider struct {
	NameValue    string
	ChatFn       func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error)
	StreamChunks []llm.StreamChunk
	StreamErr    error
}

func (p *Provider) Name() string {
	if p.NameValue != "" {
		return p.NameValue
	}
	return "mock"
}

func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	if p.ChatFn != nil {
		return p.ChatFn(ctx, req)
	}
	return llm.ChatResponse{
		Message: llm.ChatMessage{
			Role:    llm.RoleAssistant,
			Content: "mock",
		},
	}, nil
}

func (p *Provider) Stream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, <-chan error) {
	ch := make(chan llm.StreamChunk, len(p.StreamChunks))
	errCh := make(chan error, 1)
	go func() {
		defer close(ch)
		defer close(errCh)
		for _, c := range p.StreamChunks {
			ch <- c
		}
		if p.StreamErr != nil {
			errCh <- p.StreamErr
		}
	}()
	return ch, errCh
}
