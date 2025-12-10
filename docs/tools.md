# Tools Layer

This layer exposes basic, sandbox-aware tooling for agent usage.

## Filesystem
- Path-guarded operations rooted at a base directory (`PathGuard`).
- `ReadFile`, `WriteFile`, `ListDir`, and substring `Search` with max result cap.
- Write operations respect `allowWrite` flag.

## Terminal
- Command execution with allow/deny lists and global `AllowExecution` flag.
- Timeout per command; configurable working directory.

## Git
- `git status --short`
- `git apply` with `dry_run` support (enforced when writes are disabled); backups are taken before real applies and tracked in a stack with lineage.
- `git.restore_backup` reverts the latest backup or a specific id; `git.list_backups` lists stack ids; `git.preview_backup` shows backup content.

## Semantic
- Lightweight tokenizer-based `semantic.search` tool to find relevant files by overlap with a query (top-k; capped by config).
- Enabled via `tools.enable_semantic`; limits are controlled by `semantic_max_files` and `semantic_max_file_bytes`.

## Registry
- `tools.Registry` bundles filesystem, terminal, and git tools for easy wiring and schema exposure (`/tools/schemas`).
- Schemas currently descriptive only; validation is minimal (required/type checks).
- Minimal validation is enforced in tool execution (required fields, sandbox flags, dry-run restriction on git apply when disabled); tool schemas are checked for type/required fields.
- Tool calls can be provided via `RunTaskRequest.tools` (CLI flag `--tools` as JSON array).
- Git apply will default to dry-run when sandbox writes are disabled; if dry-run-only is enforced, non-dry calls are rejected and backups (patch copies) are saved before apply for potential rollback.

## Notes / TODO
- Integrate globbing, richer apply_patch validation, and git-aware safety (backups, stash).
- Enforce sandbox profiles from config (white/blacklists) in agent tool calls.
- Add JSON schemas for tool-call validation and richer error mappers.
