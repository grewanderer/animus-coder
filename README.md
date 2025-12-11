<p align="center">
  <img src="assets/banner.svg" width="640" alt="Animus Coder Banner"/>
</p>

<h1 align="center">Animus-Coder</h1>
<p align="center">Autonomous coding agent for real engineering workflows.</p>

---

## Overview

Animus-Coder is a compact autonomous coding engine built around a predictable multi-step reasoning loop:

**Plan → Act → Reflect → Test**

It executes tasks end-to-end with structured planning, safe tool actions, patch generation, test execution, and streamed progress via Connect RPC.

Designed for real development environments: reproducible, observable, controllable.

---

## Core Features

### ⚙️ Agent Runtime
- Structured planning & reflection  
- Self-diffing and critic safety gates  
- Deterministic session state

### 🛠 Tool System
- Filesystem (read/write/search)  
- Terminal exec (sandboxed)  
- Git patching (preview, backup, restore)  
- Lightweight semantic search

### 🔐 Safety & Sandboxing
- Allow/deny exec and write  
- Network/write guards  
- Dry-run enforcement

### 🌐 Model Strategy
- Independent planner/coder/critic models  
- Ordered fallbacks  
- Cost-aware budgeting

### 📡 IO & Observability
- Connect/NDJSON streaming  
- Prometheus metrics (latency, errors, token usage)

---

## Architecture

- **CLI** — streaming tasks, cancel-aware  
- **Daemon** — transport, schema, sandbox registry, metrics  
- **Agent Core** — reasoning pipeline & model orchestration  
- **Tools** — validated operations with safety rails  
- **LLM Providers** — planner / coder / critic behaviors  
- **Metrics** — visibility into system health

---

## Quickstart

```bash
# 1. Configure
cp configs/config.example.yaml configs/config.yaml

# 2. Run daemon
go run ./cmd/mycodexd

# 3. Submit a task
go run ./cmd/mycodex -- run "refactor caching layer" \
  --context internal/rpc/agent \
  --planner-model cheap-planner
```

---

## Project Status

- Roadmap → `roadmap.json`  
- Completed → `done.json`

Upcoming:
- deeper semantic search  
- patch lineage graph  
- long-term memory  
- IDE integration

---

## Contributing

PRs welcome.

```bash
GOCACHE=$PWD/.gocache go test ./...
```
