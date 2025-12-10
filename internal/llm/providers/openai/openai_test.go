package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/animus-coder/animus-coder/internal/llm"
)

func TestChatSendsRequestAndParsesResponse(t *testing.T) {
	t.Parallel()

	p := NewProvider("openai", "http://mock", "key", 5*time.Second)
	p.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, "/v1/chat/completions", r.URL.Path)
			require.Equal(t, "Bearer key", r.Header.Get("Authorization"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var reqBody map[string]interface{}
			require.NoError(t, json.Unmarshal(body, &reqBody))
			require.Equal(t, "gpt-4o-mini", reqBody["model"])

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"choices": [{
						"index": 0,
						"finish_reason": "stop",
						"message": {"role": "assistant", "content": "hello"}
					}],
					"usage": {"prompt_tokens": 1, "completion_tokens": 2, "total_tokens": 3}
				}`)),
			}, nil
		}),
	}

	resp, err := p.Chat(context.Background(), llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.ChatMessage{
			{Role: llm.RoleUser, Content: "hi"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "hello", resp.Message.Content)
	require.Equal(t, "stop", resp.FinishReason)
	require.Equal(t, 3, resp.Usage.TotalTokens)
}

func TestStreamWrapsChat(t *testing.T) {
	t.Parallel()

	p := NewProvider("openai", "http://mock", "", 0)
	p.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"choices": [{
						"index": 0,
						"finish_reason": "stop",
						"message": {"role": "assistant", "content": "streamed"}
					}],
					"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
				}`)),
			}, nil
		}),
	}

	ch, errCh := p.Stream(context.Background(), llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.ChatMessage{
			{Role: llm.RoleUser, Content: "hi"},
		},
	})

	chunk := <-ch
	require.Equal(t, "streamed", chunk.Content)
	require.Equal(t, "stop", chunk.FinishReason)
	require.Empty(t, <-errCh)
}

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
