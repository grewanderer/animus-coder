# Agent Core (Basic Mode)

This phase introduces a minimal agent loop with session memory and context injection.

## Components
- `Agent` (`internal/agent/agent.go`): Maintains sessions, builds prompts, and calls LLM providers via the registry.
- `Session`: Tracks conversation history per session ID.
- `ContextFile`: Optional file snippets passed into the user prompt.
- Prompt builders: `buildSystemPrompt` (role/instructions) and `buildUserPrompt` (user text + context block).

## Flow
0) (Optional) When `agent.enable_plan` is true, generate a short numbered plan once per session and cache it. The plan is injected into subsequent prompts and streamed as a `plan` event to the client.
1) Resolve model through the LLM registry (default if unspecified).
2) Build messages: system prompt → cached plan (if any) → previous reflection (if any) → previous history → new user prompt (with context).
3) Call provider `Chat` (non-streaming) with configured tokens/temperature.
4) Append user/assistant messages to session history.
5) After each step, optionally run reflection (`agent.enable_reflect`) which is stored in history and streamed as `reflect`.
6) When a step is done or finishes naturally, optionally run configured tests (`agent.enable_test_run` + `agent.test_command`) via sandboxed terminal and stream as `test`.
7) Repeat up to `agent.max_steps` or until finish criteria hit (`finish_reason` or `[done]` token).

## Usage (programmatic)
```go
reg, _ := configbuilder.BuildRegistryFromConfig(cfg)
agent := agent.New(reg, cfg.Agent)
resp, _ := agent.Run(ctx, agent.Request{
    SessionID: "s1",
    Prompt: "Explain file",
    Context: []agent.ContextFile{{Path: "main.go", Content: "..."}},
})
fmt.Println(resp.Message.Content)
```

## Notes
- Streaming and tool-calls are implemented end-to-end: model responses can include JSON tool-call descriptors that are executed mid-run and streamed as `tool` events; CLI renders tokens/messages/plan/reflect/test/tool events as they arrive.
- Temperatures/max_tokens prefer agent config, then model settings, then defaults.
- Planning runs once per session, uses the same model as execution, and is cached; disable via `agent.enable_plan: false` for single-shot behaviour.
- Reflection is a lightweight critique after each step (when enabled) and feeds back into the next prompt via history. Tool outputs from the step and the latest test run summary are included in the reflection prompt to improve follow-up actions. Reflection now requests structured JSON `{quality, issues[], recommendations[], block_apply, notes}`; the raw message is still streamed while the daemon attempts to parse the JSON for downstream use. If `block_apply` is true, the run finishes with `finish_reason=blocked_by_reflect`.
- Reflection policy (`agent.reflection_policy`): `block_on_critical` (default) stops the run when `block_apply` is true; `warn_only` and `never_block` ignore the block flag but still stream the critique.
- Self-diff: when `agent.enable_self_diff` is true, the reflection prompt includes a simple self-diff between the previous assistant response and the current one to encourage critique of changed plans/actions.
- Test runs: results include exit code, raw output, and a simple parser extracts failing test names into `failing_tests`/`test_summary` fields on `test` events.
- Test retries/timeouts: configure `agent.test_retries` (number of extra attempts) and `agent.test_timeout_seconds` (per-attempt timeout) to keep test-driven loops bounded.
- Context files can be injected via CLI `--context` or RunTaskRequest.context_paths; total bytes are capped by `agent.max_context_bytes` (truncates with `[truncated]`). Directories are summarized, and when no context is provided the daemon auto-loads a small, relevance-biased set (prompt-mentioned files + repo defaults like README/go.mod).
- Test-run executes the configured `agent.test_command` through the sandboxed terminal only when enabled; failures surface in the `test` event but do not abort the stream. Test output is also fed into the reflection prompt to drive the next step.
- Tool-calls: model responses can include JSON tool call descriptors, which are executed before the next step and streamed as `tool` events.
