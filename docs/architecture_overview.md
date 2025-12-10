# Architecture Overview

This project implements a production-grade replacement for Codex CLI with a split architecture:

- `mycodex` CLI: handles user commands, session setup, config selection, and local prompts.
- `mycodexd` daemon: long-running backend that will host Agent Core, LLM providers, sandbox, tools, and observability surfaces.
- Shared libraries under `internal/`: config loading/validation, logging, daemon HTTP surfaces, and (later) agent/tooling packages.

Data flows:

- CLI → daemon: user prompt, flags, and context (streaming planned).
- Daemon → LLM providers: calls over provider adapters (OpenAI, OpenRouter, Ollama, vLLM/LM Studio).
- Agent Core → tools: filesystem/terminal/git operations executed in a sandbox wrapper.
- Daemon → observability: structured logs, metrics endpoint, and future tracing hooks.

Current state:

- Repository skeleton is in place.
- Config loader supports YAML + ENV overrides with validation.
- Minimal CLI (`mycodex`) exposes `version` and `doctor`.
- Minimal daemon (`mycodexd`) serves `/health` and `/metrics` placeholders with graceful shutdown.
