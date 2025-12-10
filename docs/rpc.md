# RPC Protocol (Connect default)

Transport is chosen by `server.transport` in config (`connect` by default, `ndjson` as legacy). Both surface the same payload shapes.

## Connect RunTask (default)
- Path: `/connect.agent.v1.AgentService/RunTask` (Connect bidi stream over HTTP/2, h2c enabled).
- Request stream: first message must include `RunTaskStreamRequest{ run: RunTaskRequest{ session_id, correlation_id?, model, prompt, tools?, context_paths? } }`. Session/correlation IDs are auto-generated when absent.
- Response stream: `RunTaskEvent` messages:
  - `plan`, `message`, `token`, `tool`, `reflect`, `test`, `error`, `done` (fields unchanged; events include `session_id` and `correlation_id`).
  - `reflect` events may include `critique` (parsed JSON) when reflection returns structured critique payload.
  - `test` events include `test_summary`, `failing_tests`, and `test_attempts` when the runner can parse failing test names from output.
- Cancellation: client sends `{ cancel: true, session_id, correlation_id }` on the same stream; daemon cancels the run.
- Tool schemas: `GET /tools/schemas` (fs/terminal/git descriptors).
- Metrics: active streaming sessions and transport errors are exported with `transport` labels alongside existing agent metrics.

## Legacy NDJSON
- Endpoint: `POST /agent/run`
- Request: `RunTaskRequest` JSON body (`session_id`, `correlation_id?`, `prompt`, optional `tools`, `context_paths`).
- Response: NDJSON stream of `RunTaskEvent` (same fields as Connect).
- Useful for environments without HTTP/2 support; remains available alongside Connect while migrating.
