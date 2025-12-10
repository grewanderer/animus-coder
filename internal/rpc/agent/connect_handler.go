package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"

	"github.com/animus-coder/animus-coder/internal/observability"
	"github.com/animus-coder/animus-coder/internal/rpc"
	"github.com/animus-coder/animus-coder/internal/rpc/connectjson"
)

const ConnectRunTaskProcedure = "/connect.agent.v1.AgentService/RunTask"

// NewConnectHandler builds a Connect bidi stream handler for RunTask.
func NewConnectHandler(runner Runner, metrics *observability.Metrics) (string, http.Handler) {
	h := &connectRunHandler{runner: runner, metrics: metrics}
	return ConnectRunTaskProcedure, connect.NewBidiStreamHandler(ConnectRunTaskProcedure, h.handle, connect.WithCodec(connectjson.Codec{}))
}

type connectRunHandler struct {
	runner  Runner
	metrics *observability.Metrics
}

func (h *connectRunHandler) handle(ctx context.Context, stream *connect.BidiStream[rpc.RunTaskStreamRequest, rpc.RunTaskEvent]) error {
	if h.metrics != nil {
		h.metrics.IncActiveSessions("connect")
		defer h.metrics.DecActiveSessions("connect")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	first, err := stream.Receive()
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordTransportError("connect", "receive_first")
		}
		return err
	}
	if first == nil || first.Run == nil {
		if h.metrics != nil {
			h.metrics.RecordTransportError("connect", "missing_run")
		}
		return connect.NewError(connect.CodeInvalidArgument, errors.New("first message must include run payload"))
	}

	req := *first.Run
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	if req.CorrelationID == "" {
		req.CorrelationID = req.SessionID + "-corr"
	}

	// Listen for cancellation messages from the client.
	go func() {
		for {
			msg, recvErr := stream.Receive()
			if recvErr != nil {
				if h.metrics != nil && !errors.Is(recvErr, context.Canceled) {
					h.metrics.RecordTransportError("connect", "receive_stream")
				}
				cancel()
				return
			}
			if msg != nil && msg.Cancel {
				cancel()
				return
			}
		}
	}()

	httpReq := &http.Request{}
	httpReq = httpReq.WithContext(ctx)

	events, runErr := h.runner.Run(httpReq, req)
	if runErr != nil {
		if h.metrics != nil {
			h.metrics.RecordTransportError("connect", "runner_error")
		}
		return connect.NewError(connect.CodeInternal, runErr)
	}

	for ev := range events {
		if err := stream.Send(&ev); err != nil {
			if h.metrics != nil {
				h.metrics.RecordTransportError("connect", "send")
			}
			return err
		}
	}
	return nil
}
