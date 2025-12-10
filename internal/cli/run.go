package cli

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"

	"github.com/animus-coder/animus-coder/internal/rpc"
	agentrpc "github.com/animus-coder/animus-coder/internal/rpc/agent"
	"github.com/animus-coder/animus-coder/internal/rpc/connectjson"
)

// NewRunCmd wires the run command to stream events from the daemon.
func NewRunCmd(opts *Options) *cobra.Command {
	var toolJSON string
	var contextPaths []string
	var modelOverride string
	var plannerModel string
	var criticModel string

	cmd := &cobra.Command{
		Use:   "run \"<prompt>\"",
		Short: "Send a prompt to the daemon and stream response tokens",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(opts)
			if err != nil {
				return err
			}

			prompt := args[0]
			if strings.TrimSpace(prompt) == "" {
				return fmt.Errorf("prompt cannot be empty")
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			sessionID := fmt.Sprintf("cli-%d", time.Now().UnixNano())
			corrID := fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano())

			reqBody := rpc.RunTaskRequest{
				SessionID:     sessionID,
				CorrelationID: corrID,
				Model:         modelOverride,
				Prompt:        prompt,
				Tools:         parseToolCalls(toolJSON),
				ContextPaths:  contextPaths,
				PlannerModel:  plannerModel,
				CriticModel:   criticModel,
			}

			baseURL := daemonURL(cfg.Server.Addr)
			switch strings.ToLower(strings.TrimSpace(cfg.Server.Transport)) {
			case "ndjson":
				return runNDJSON(ctx, cmd, baseURL+"/agent/run", reqBody)
			default:
				return runConnect(ctx, cmd, baseURL+agentrpc.ConnectRunTaskProcedure, reqBody)
			}
		},
	}

	cmd.Flags().StringVar(&toolJSON, "tools", "", "JSON array of tool calls to execute before agent (optional)")
	cmd.Flags().StringSliceVar(&contextPaths, "context", nil, "Context file paths to load and send with the prompt (repeatable or comma-separated)")
	cmd.Flags().StringVar(&modelOverride, "model", "", "Override coder model id for this run")
	cmd.Flags().StringVar(&plannerModel, "planner-model", "", "Override planner model id for this run")
	cmd.Flags().StringVar(&criticModel, "critic-model", "", "Override critic model id for this run")
	return cmd
}

func daemonURL(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}

func parseToolCalls(raw string) []rpc.ToolCall {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var calls []rpc.ToolCall
	if err := json.Unmarshal([]byte(raw), &calls); err != nil {
		return nil
	}
	return calls
}

func runNDJSON(ctx context.Context, cmd *cobra.Command, url string, reqBody rpc.RunTaskRequest) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var evt rpc.RunTaskEvent
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			return fmt.Errorf("decode event: %w", err)
		}
		if err := renderEvent(cmd, evt); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func runConnect(ctx context.Context, cmd *cobra.Command, url string, reqBody rpc.RunTaskRequest) error {
	client := connect.NewClient[rpc.RunTaskStreamRequest, rpc.RunTaskEvent](buildH2CClient(), url, connect.WithCodec(connectjson.Codec{}))
	stream := client.CallBidiStream(ctx)

	if err := stream.Send(&rpc.RunTaskStreamRequest{Run: &reqBody}); err != nil {
		return err
	}

	// propagate cancellation to the daemon.
	go func() {
		<-ctx.Done()
		_ = stream.Send(&rpc.RunTaskStreamRequest{Cancel: true, SessionID: reqBody.SessionID, CorrelationID: reqBody.CorrelationID})
		_ = stream.CloseRequest()
	}()

	for {
		evt, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if err := renderEvent(cmd, *evt); err != nil {
			return err
		}
	}
	_ = stream.CloseRequest()
	return stream.CloseResponse()
}

func renderEvent(cmd *cobra.Command, evt rpc.RunTaskEvent) error {
	switch evt.Type {
	case "tool":
		fmt.Fprintf(cmd.OutOrStdout(), "[tool %s] %s\n", evt.ToolName, evt.ToolOutput)
	case "plan":
		fmt.Fprintf(cmd.OutOrStdout(), "[plan]\n%s\n", evt.Message)
	case "reflect":
		fmt.Fprintf(cmd.OutOrStdout(), "[reflect]\n%s\n", evt.Message)
		if evt.Critique != nil {
			if data, err := json.MarshalIndent(evt.Critique, "", "  "); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "[critique]\n%s\n", string(data))
			}
		}
	case "test":
		status := "ok"
		if evt.ExitCode != 0 || evt.Error != "" {
			status = "fail"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[test %s exit=%d attempts=%d]\n", status, evt.ExitCode, evt.TestAttempts)
		if len(evt.FailingTests) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Failing: %s\n", strings.Join(evt.FailingTests, ", "))
		}
		if evt.TestSummary != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Summary: %s\n", evt.TestSummary)
		}
		fmt.Fprintln(cmd.OutOrStdout(), evt.Message)
		if evt.Error != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "[test error] %s\n", evt.Error)
		}
	case "token":
		fmt.Fprint(cmd.OutOrStdout(), evt.Token+" ")
	case "message":
		fmt.Fprintln(cmd.OutOrStdout(), evt.Message)
	case "done":
		fmt.Fprintln(cmd.OutOrStdout(), "\n[done]")
	case "error":
		return fmt.Errorf("daemon error: %s", evt.Error)
	}
	return nil
}

func buildH2CClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		},
	}
}
