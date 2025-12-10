# Observability

Current coverage:
- Structured logging via zap (configured by `logging.level`/`logging.format`).
- Prometheus metrics served at `/metrics` when `server.metrics_enabled` is true.
- Agent metrics:
  - `mycodex_agent_requests_total{finish_reason}`
  - `mycodex_agent_duration_seconds{finish_reason}`
  - `mycodex_agent_tokens_total{finish_reason}`
  - `mycodex_model_usage_total{role,model}`
  - `mycodex_model_failures_total{role,model}`
- Metrics registry uses a dedicated Prometheus registry; daemon exposes it through `promhttp.Handler`.

Planned:
- Add token accounting from providers (prompt/completion).
- Trace spans (OpenTelemetry) and correlation IDs on RPC events.
- Dashboards (Grafana JSON) and CI artifacts for metrics/logs.
