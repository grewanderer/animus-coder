# Animus-coder (MyCodex) — Autonomous Coding Agent

An open, controllable autonomous coding agent that plans, edits, runs tests, critiques itself, and streams progress to your terminal. The project ships a daemon (`mycodexd`), a CLI (`mycodex`), and a modular agent core wired to tools, sandbox, observability, and a multi-model strategy layer.

## What’s implemented
- Multi-step Plan → Act → Reflect → Test loop with session history.
- Structured planning and reflection (critic) steps, optional self-diff, and block-on-critical policy.
- Test runner with retries/timeouts, parsing of failing tests, and streamed test events.
- Tool-calling: filesystem (read/write/search/structure), terminal exec, git apply_patch with backups/list/preview/restore, semantic.search (lightweight relevance search).
- Safe sandboxing: allow/deny commands, write/network guards, apply_patch dry-run enforcement.
- Context loader with size caps, auto-discovery from prompt, and semantic hits.
- Multi-model strategy layer: per-role planner/coder/critic selection, CLI overrides, ordered fallbacks, and cost-aware “expensive” budgeting with Prometheus usage/failure metrics.
- Streaming RPC: Connect bidi by default (cancel-aware) with legacy NDJSON compatibility; events include tokens/messages/plan/reflect/test/tool.
- Observability: Prometheus metrics (/metrics) for requests, duration, tokens, transport sessions/errors, and model usage/failures.

## Quick start
Requires Go 1.22+.

1) Configure
```bash
cp configs/config.example.yaml configs/config.yaml
# edit providers/models (API keys), sandbox, strategy, agent settings
```

2) Run the daemon (Connect/h2c by default)
```bash
go run ./cmd/mycodexd
# or MYCODEX_SERVER_TRANSPORT=ndjson go run ./cmd/mycodexd
```

3) Send a task from another terminal
```bash
go run ./cmd/mycodex -- run "refactor the handler to add caching" \
  --context internal/rpc/agent \
  --planner-model cheap-planner \
  --critic-model reliable-critic
```
Flags: `--model` (coder), `--planner-model`, `--critic-model`, `--context` paths, `--tools` (JSON pre-seeded tool calls).

## Configuration highlights
- Providers/Models: define logical models and mark `expensive: true` where needed.
- Agent: `max_steps`, `max_tokens`, `enable_plan`, `enable_reflect`, `enable_self_diff`, `enable_test_run`, `test_command`, `test_retries`, `test_timeout_seconds`, `max_context_bytes`.
- Tools/Sandbox: exec/git/file write toggles, allow/deny command lists, network/write guards, exec timeouts.
- Strategy: `default_model`, `planner_model`, `coder_model`, `critic_model`, `overrides`, `fallbacks`, `max_expensive`.
- Server: `transport` (`connect` or `ndjson`), `metrics_enabled`, `addr`.

See `configs/config.example.yaml` for a full template.

## Architecture (brief)
- CLI: streams RunTask events (tokens, plan, reflect, test, tool) and forwards cancel.
- Daemon: hosts Connect and NDJSON transports, tool schema endpoint, metrics, and builds sandbox/tool registry and strategy engine.
- Agent core: keeps session history, builds prompts, and calls providers via the registry.
- Tools: filesystem/terminal/git/semantic with validation and safety rails (backups, dry-run enforcement).
- Observability: Prometheus registry and metrics helpers.

## Roadmap & status
- Completed phases and in-progress plans live in `roadmap.json` with delivered items tracked in `done.json`.
- Upcoming milestones include FSM-based tool governance, richer semantic indexing, patch stack lineage, long-term memory, and UI/IDE surfaces.

## Contributing
Issues and PRs are welcome. Start with `docs/` (architecture, agent, tools, strategy, tests, observability) and keep changes reproducible (`GOCACHE=$PWD/.gocache go test ./...`).

## License
MIT (add a LICENSE file before distributing).
