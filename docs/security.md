# Security & Sandbox

Current state:
- Filesystem operations are path-guarded to the daemon working directory; absolute paths and traversal are rejected.
- Terminal execution can be disabled globally and constrained via allow/deny lists; timeouts apply to each command.
- Network commands are auto-denied when `sandbox.allow_network` is false (curl, wget, ping, nc/netcat, telnet, ssh/scp/sftp) even if allowlists are empty.
- Sandbox constructor (`internal/tools/sandbox.go`) wires tools according to config flags (`sandbox.enabled`, `allow_write`, `allow_network`, `allowed_commands`, `denied_commands`, `tools.allow_exec`).
- Git apply is forced into dry-run when writes are disallowed; backups are taken before real applies with restore/preview/list tooling.
- Tool validation checks schemas, sandbox flags, and dry-run requirements before execution.

Planned:
- Docker-based sandbox and stricter syscalls/resource limits.
- JSON-schema validation of tool calls and dry-run modes for destructive operations.
- Advanced network enforcement (per-command overrides, proxying, and telemetry).
