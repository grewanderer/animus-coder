# Test-Driven Workflow (early)

Current behaviour:
- Configure test runs with `agent.enable_test_run`, `agent.test_command`, `agent.test_retries`, and `agent.test_timeout_seconds`.
- On completion (or when `finish_reason` indicates stop), the daemon runs the configured test command via the sandboxed terminal tool.
- Test events include: exit code, raw output, `test_attempts`, parsed `failing_tests`, and `test_summary` when the parser can extract failing names from output.
- Reflections receive test context (including failing tests and summary) to inform next steps.

Usage example:
```
mycodex run "fix the bug" --config configs/config.yaml
# with agent.test_command set (e.g., "go test ./...") and enable_test_run: true
```

Notes / next steps:
- Parsing is heuristic; richer language-specific parsers and subset selection (per pattern) are planned.
- Iterative test-fix loops and strategy policies will be added to re-run tests mid-turn when needed.
