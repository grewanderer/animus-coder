package agent

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/animus-coder/animus-coder/internal/rpc"
	"github.com/animus-coder/animus-coder/internal/rpc/connectjson"
)

func TestConnectHandlerStreamsEvents(t *testing.T) {
	path, handler := NewConnectHandler(EchoRunner{}, nil)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot open listener in sandbox: %v", err)
	}

	server := httptest.NewUnstartedServer(h2c.NewHandler(mux, &http2.Server{}))
	server.Listener = ln
	server.Start()
	t.Cleanup(server.Close)

	client := connect.NewClient[rpc.RunTaskStreamRequest, rpc.RunTaskEvent](
		&http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, network, addr)
				},
			},
		},
		server.URL+path,
		connect.WithCodec(connectjson.Codec{}),
	)

	stream := client.CallBidiStream(context.Background())
	require.NoError(t, stream.Send(&rpc.RunTaskStreamRequest{
		Run: &rpc.RunTaskRequest{SessionID: "conn-1", Prompt: "hello world"},
	}))
	require.NoError(t, stream.CloseRequest())

	var messageSeen bool
	for {
		evt, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if evt.Type == "message" {
			messageSeen = true
			require.Equal(t, "conn-1", evt.SessionID)
		}
	}
	require.NoError(t, stream.CloseResponse())
	require.True(t, messageSeen)
}
