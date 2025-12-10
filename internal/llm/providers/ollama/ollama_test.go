package ollama

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/animus-coder/animus-coder/internal/llm"
)

func TestChat(t *testing.T) {
	t.Parallel()

	p := NewProvider("ollama", "http://mock", 0)
	p.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, "/api/chat", r.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"message":{"role":"assistant","content":"pong"}}`)),
			}, nil
		}),
	}

	resp, err := p.Chat(context.Background(), llm.ChatRequest{
		Model: "llama3",
		Messages: []llm.ChatMessage{
			{Role: llm.RoleUser, Content: "ping"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "pong", resp.Message.Content)
}

func TestStream(t *testing.T) {
	t.Parallel()

	p := NewProvider("ollama", "http://mock", 0)
	p.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"message":{"role":"assistant","content":"chunk"}}`)),
			}, nil
		}),
	}

	ch, errCh := p.Stream(context.Background(), llm.ChatRequest{
		Model: "llama3",
		Messages: []llm.ChatMessage{
			{Role: llm.RoleUser, Content: "hi"}},
	})

	chunk := <-ch
	require.Equal(t, "chunk", chunk.Content)
	require.Empty(t, <-errCh)
}

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
