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

<svg width="820" height="360" viewBox="0 0 820 360" xmlns="http://www.w3.org/2000/svg">

  <style>
    .box { fill:#0f172a; stroke:#38bdf8; stroke-width:1.2; rx:10; }
    .text { fill:#e2e8f0; font-family: system-ui; font-size:13px; text-anchor:middle; }
    .sub { fill:#94a3b8; font-family: system-ui; font-size:11px; text-anchor:middle; }
    .arrow { stroke:#38bdf8; stroke-width:1.6; marker-end:url(#arrowhead); }
  </style>

  <defs>
    <marker id="arrowhead" markerWidth="8" markerHeight="6" refX="6" refY="3" orient="auto">
      <polygon points="0 0, 0 6, 7 3" fill="#38bdf8"/>
    </marker>
  </defs>

  <!-- CLI -->
  <rect x="40" y="40" width="180" height="60" class="box"/>
  <text x="130" y="70" class="text">CLI</text>
  <text x="130" y="88" class="sub">Tasks • Context • Cancel</text>

  <!-- Daemon -->
  <rect x="320" y="40" width="180" height="60" class="box"/>
  <text x="410" y="70" class="text">Daemon</text>
  <text x="410" y="88" class="sub">Connect • NDJSON • Metrics</text>

  <!-- Observability -->
  <rect x="600" y="40" width="180" height="60" class="box"/>
  <text x="690" y="70" class="text">Observability</text>
  <text x="690" y="88" class="sub">Prometheus</text>

  <!-- Arrows top row -->
  <line x1="220" y1="70" x2="320" y2="70" class="arrow"/>
  <line x1="500" y1="70" x2="600" y2="70" class="arrow"/>

  <!-- Agent Core -->
  <rect x="250" y="150" width="320" height="80" class="box"/>
  <text x="410" y="178" class="text">Agent Core</text>
  <text x="410" y="196" class="sub">Plan • Act • Reflect • Test</text>

  <!-- Down arrows -->
  <line x1="410" y1="100" x2="410" y2="150" class="arrow"/>

  <!-- Tools -->
  <rect x="40" y="260" width="240" height="70" class="box"/>
  <text x="160" y="292" class="text">Tool Registry</text>
  <text x="160" y="310" class="sub">FS • Terminal • Git • Semantic</text>

  <!-- LLM providers -->
  <rect x="540" y="260" width="240" height="70" class="box"/>
  <text x="660" y="292" class="text">LLM Providers</text>
  <text x="660" y="310" class="sub">Planner • Coder • Critic</text>

  <!-- Arrows bottom -->
  <line x1="330" y1="230" x2="160" y2="260" class="arrow"/>
  <line x1="490" y1="230" x2="660" y2="260" class="arrow"/>

</svg>


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
