# Context Handling

Basic context injection is supported by passing `ContextFile` entries into the agent request. These are inlined into the user prompt as a block:

```
Context:
File: path/to/file
<contents>
---
```

## CLI usage
- Use `mycodex run "<prompt>" --context file1.go --context dir/file2.txt` to include local files. Paths are resolved via sandbox path guard.
- `agent.max_context_bytes` (default: 32768) caps the total bytes loaded across all context files. Files that exceed the remaining budget are truncated with a `[truncated]` marker.
- Directories passed via `--context` are summarized into a short tree (depth-limited, skips common vendor/.git folders) so the model sees the repo structure without dumping every file.
- When no `--context` is provided, the daemon will auto-discover a small set of likely-relevant files: mentioned paths from the prompt (e.g., `main.go`), plus defaults like `README.md`, `go.mod`, `package.json`, and shallow workspace structure. When `tools.enable_semantic` is true, a lightweight `semantic.search` pass runs first to propose top matches based on the prompt.

## Behaviour
- Context is loaded by the daemon using the filesystem tool (path-guarded to the working directory).
- Loaded content is injected into the user prompt before model execution; planning also receives the same context.
- Context loading stops once the byte budget is hit; each file/structure block is also truncated to avoid runaway prompts.
- Semantic context is capped by `tools.semantic_max_files` and `tools.semantic_max_file_bytes`; results still respect `agent.max_context_bytes` when inlined.

Future work: structured file tree reading, ignore patterns, and agent-driven selective loading.
