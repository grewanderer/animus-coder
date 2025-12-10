package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/animus-coder/animus-coder/internal/observability"
	"github.com/animus-coder/animus-coder/internal/rpc"
)

// Runner executes a task and yields streamed events.
type Runner interface {
	Run(r *http.Request, req rpc.RunTaskRequest) (<-chan rpc.RunTaskEvent, error)
}

// Handler processes RunTask requests and streams NDJSON events.
type Handler struct {
	runner  Runner
	metrics *observability.Metrics
}

// NewHandler constructs a handler instance.
func NewHandler(runner Runner, metrics *observability.Metrics) *Handler {
	return &Handler{runner: runner, metrics: metrics}
}

// ServeHTTP handles POST /agent/run with an NDJSON stream of RunTaskEvent.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		if h.metrics != nil {
			h.metrics.RecordTransportError("ndjson", "method_not_allowed")
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.metrics != nil {
		h.metrics.IncActiveSessions("ndjson")
		defer h.metrics.DecActiveSessions("ndjson")
	}

	var req rpc.RunTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if h.metrics != nil {
			h.metrics.RecordTransportError("ndjson", "decode")
		}
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	if req.CorrelationID == "" {
		req.CorrelationID = fmt.Sprintf("%s-corr", req.SessionID)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)

	var events <-chan rpc.RunTaskEvent
	if h.runner != nil {
		ev, err := h.runner.Run(r, req)
		if err != nil {
			if h.metrics != nil {
				h.metrics.RecordTransportError("ndjson", "runner_error")
			}
			http.Error(w, fmt.Sprintf("runner error: %v", err), http.StatusInternalServerError)
			return
		}
		events = ev
	} else {
		events = runTaskEcho(req)
	}

	writer := bufio.NewWriter(w)
	for ev := range events {
		if err := json.NewEncoder(writer).Encode(ev); err != nil {
			break
		}
		writer.Flush()
		flusher.Flush()
	}
}

// runTaskEcho simulates an agent run by emitting token events.
func runTaskEcho(req rpc.RunTaskRequest) <-chan rpc.RunTaskEvent {
	out := make(chan rpc.RunTaskEvent, 16)
	go func() {
		defer close(out)
		out <- rpc.RunTaskEvent{
			Type:          "message",
			SessionID:     req.SessionID,
			CorrelationID: req.CorrelationID,
			Message:       "session started",
			Step:          0,
		}

		words := strings.Fields(req.Prompt)
		for i, w := range words {
			out <- rpc.RunTaskEvent{
				Type:          "token",
				SessionID:     req.SessionID,
				CorrelationID: req.CorrelationID,
				Token:         w,
				Step:          i + 1,
			}
		}

		out <- rpc.RunTaskEvent{
			Type:          "done",
			SessionID:     req.SessionID,
			CorrelationID: req.CorrelationID,
			Done:          true,
			FinishReason:  "eos",
		}
	}()
	return out
}
