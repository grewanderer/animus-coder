````md
<p align="center">
  <img src="assets/banner.svg" width="680" alt="Animus Coder Banner"/>
</p>

<h1 align="center">Animus-Coder — Autonomous Coding Agent</h1>
<p align="center">
  Open, controllable, self-reflective coding automation system designed for real engineering workflows.
</p>

---

## 🚀 Overview

**Animus-Coder** is an autonomous coding agent capable of planning, editing, generating patches, running tests, critiquing itself, and streaming activity in real time.

It ships with:

- **Daemon (`mycodexd`)** — RPC server, sandbox, metrics, agent runtime  
- **CLI (`mycodex`)** — interactive task runner with streaming output  
- **Agent Core** — modular reasoning, multi-model strategy, observability, and tool governance  

This project demonstrates production-level patterns: **multi-step reasoning**, **safe tool execution**, **sandboxing**, **Connect RPC**, **Prometheus metrics**, **fallback model strategies**, and **LLM-driven patch generation**.

---

## ✅ Features

### 🧠 Agent Loop
- Multi-step **Plan → Act → Reflect → Test** loop  
- Session history and structured reasoning  
- Optional self-diff and block-on-critical critic policy  

### 🛠 Tooling Layer
- Filesystem read/write/search/structure  
- Terminal execution (with guardrails)  
- Git: `apply_patch`, preview/backup/restore, dry-run enforcement  
- Lightweight semantic search tool  
- Validated tool schemas + sandbox boundaries  

### 🔒 Sandboxing
- Allow/deny execution lists  
- Network/write guards  
- Execution timeouts  
- Safe patch application flow  

### ⚙️ Multi-Model Strategy
- Independent models for planner/coder/critic  
- Ordered fallbacks  
- Cost-aware budgeting (`expensive: true`)  
- Prometheus-tracked usage/failure metrics  

### 🔌 Streaming RPC
- **Connect** (bidi, cancel-aware)  
- **NDJSON fallback**  
- Streamed events: tokens, messages, plan, reflect, test outputs, tool activity  

### 📈 Observability
- Prometheus: requests, latencies, token counts, errors  
- Model usage/failures  
- Transport-session metrics  

---

## 🏁 Quick Start

**Requirements:** Go 1.22+

### 1. Configure
```bash
cp configs/config.example.yaml configs/config.yaml
# edit providers/models (API keys), sandbox, tool settings, strategy, agent params
````

### 2. Run the daemon

```bash
go run ./cmd/mycodexd
# or
MYCODEX_SERVER_TRANSPORT=ndjson go run ./cmd/mycodexd
```

### 3. Submit a task

```bash
go run ./cmd/mycodex -- run "refactor the handler to add caching" \
  --context internal/rpc/agent \
  --planner-model cheap-planner \
  --critic-model reliable-critic
```

Useful flags:
`--model` (coder), `--planner-model`, `--critic-model`, `--context`, `--tools` (JSON snippet of pre-seeded tool calls)

---

## ⚙️ Configuration Overview

### Providers / Models

* Define logical models
* Tag expensive ones (`expensive: true`)
* Override per role (planner/coder/critic)

### Agent Settings

* `max_steps`, `max_tokens`, `enable_plan`, `enable_reflect`
* `enable_self_diff`, `test_command`, `test_retries`, `test_timeout_seconds`
* Context limits (`max_context_bytes`)

### Tools & Sandbox

* Allowed/denied commands
* Write/network guards
* Exec timeout configuration

### Strategy Engine

* `default_model`, `planner_model`, `coder_model`, `critic_model`
* Ordered fallback chains
* Budget for expensive model usage

### Server

* Transport: `connect` (default) or `ndjson`
* `metrics_enabled`
* `addr`

Refer to:
`configs/config.example.yaml` → **full configuration template**

---

## 🧩 Architecture

<p align="center">
  <img src="assets/architecture.svg" width="720" alt="Animus Coder Architecture"/>
</p>

```text
CLI ────── stream events ──► Daemon (Connect/NDJSON)
  ▲                                 │
  │                                 ▼
Cancel & input            Agent Core (plan/act/reflect/test)
                                      │
                                      ▼
                               Tool Registry
                 (fs / terminal / git / semantic search)
                                      │
                                      ▼
                              LLM Providers
                                      │
                                      ▼
                           Observability (Prometheus)
```

### Components

* **CLI** — sends tasks, streams events, cancel-aware
* **Daemon** — serves transports, metrics, schema, sandbox/tool registry
* **Agent Core** — session state, reasoning, prompting, provider registry
* **Tools** — safe, validated, reversible operations
* **LLM Providers** — model abstraction layer with per-role strategies
* **Metrics** — Prometheus registry and instrumentation

---

## 🗺 Roadmap

* Planned phases: `roadmap.json`
* Completed items: `done.json`

Upcoming work:

* FSM-based tool governance
* Deeper semantic indexing
* Patch stack lineage
* Long-term memory modules
* IDE/Editor integrations

---

## 🤝 Contributing

Issues and PRs are welcome.

Before submitting:

```bash
GOCACHE=$PWD/.gocache go test ./...
```

See `docs/` for more details on:

* architecture
* agent
* tools
* strategy
* tests
* observability

---

## 📄 License

MIT (add a `LICENSE` file before distributing)

---

<p align="center">
  <img src="assets/logo.png" width="120" alt="Animus Coder Logo"/>
</p>
```

---

# assets/architecture.svg

```svg
<svg width="960" height="540" viewBox="0 0 960 540" xmlns="http://www.w3.org/2000/svg" role="img" aria-labelledby="title desc">
  <title id="title">Animus Coder Architecture</title>
  <desc id="desc">High-level architecture showing CLI, Daemon, Agent Core, Tools, LLM Providers and Prometheus observability.</desc>

  <!-- Background -->
  <defs>
    <linearGradient id="bgGrad" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#050816"/>
      <stop offset="50%" stop-color="#0b1220"/>
      <stop offset="100%" stop-color="#020617"/>
    </linearGradient>
    <linearGradient id="boxGrad" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0%" stop-color="#1e293b"/>
      <stop offset="100%" stop-color="#0f172a"/>
    </linearGradient>
    <linearGradient id="accentGrad" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0%" stop-color="#22d3ee"/>
      <stop offset="100%" stop-color="#6366f1"/>
    </linearGradient>
    <filter id="softGlow" x="-50%" y="-50%" width="200%" height="200%">
      <feGaussianBlur stdDeviation="12" result="blur"/>
      <feColorMatrix in="blur" type="matrix" values="0 0 0 0 0.13  0 0 0 0 0.8  0 0 0 0 0.93  0 0 0 0.7 0"/>
    </filter>
  </defs>

  <rect width="960" height="540" fill="url(#bgGrad)" rx="24"/>

  <!-- Glow behind core -->
  <ellipse cx="480" cy="260" rx="260" ry="150" fill="#22d3ee" opacity="0.18" filter="url(#softGlow)"/>

  <!-- Title -->
  <text x="480" y="56" text-anchor="middle" fill="#e5e7eb" font-family="system-ui, -apple-system, BlinkMacSystemFont, sans-serif" font-size="22" font-weight="600">
    Animus Coder — High-Level Architecture
  </text>

  <!-- Row 1: CLI -->
  <rect x="70" y="110" width="200" height="70" rx="12" fill="url(#boxGrad)" stroke="#22d3ee" stroke-width="1.2"/>
  <text x="170" y="140" text-anchor="middle" fill="#e5e7eb" font-family="system-ui, sans-serif" font-size="14" font-weight="600">CLI</text>
  <text x="170" y="158" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Tasks • Context • Cancel</text>

  <!-- Row 1: Daemon -->
  <rect x="380" y="110" width="200" height="70" rx="12" fill="url(#boxGrad)" stroke="#38bdf8" stroke-width="1.2"/>
  <text x="480" y="138" text-anchor="middle" fill="#e5e7eb" font-family="system-ui, sans-serif" font-size="14" font-weight="600">Daemon</text>
  <text x="480" y="156" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Connect / NDJSON • Metrics</text>

  <!-- Row 1: Observability -->
  <rect x="690" y="110" width="200" height="70" rx="12" fill="url(#boxGrad)" stroke="#4ade80" stroke-width="1.2"/>
  <text x="790" y="138" text-anchor="middle" fill="#e5e7eb" font-family="system-ui, sans-serif" font-size="14" font-weight="600">Observability</text>
  <text x="790" y="156" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Prometheus • Grafana</text>

  <!-- Arrows row 1 -->
  <line x1="270" y1="145" x2="380" y2="145" stroke="url(#accentGrad)" stroke-width="1.6" marker-end="url(#arrow)"/>
  <line x1="580" y1="145" x2="690" y2="145" stroke="#4ade80" stroke-width="1.4" marker-end="url(#arrow)" stroke-dasharray="4 4"/>

  <!-- Row 2: Agent Core -->
  <rect x="300" y="230" width="360" height="96" rx="16" fill="url(#boxGrad)" stroke="#22d3ee" stroke-width="1.4"/>
  <text x="480" y="252" text-anchor="middle" fill="#e5e7eb" font-family="system-ui, sans-serif" font-size="14" font-weight="600">Agent Core</text>
  <text x="480" y="270" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Plan • Act • Reflect • Test • Session History</text>
  <text x="480" y="288" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Model Strategy • Budgeting • Error Policy</text>

  <!-- Arrow CLI -> Agent (via Daemon) -->
  <defs>
    <marker id="arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth">
      <path d="M0,0 L0,6 L9,3 z" fill="#38bdf8" />
    </marker>
  </defs>

  <line x1="170" y1="180" x2="170" y2="260" stroke="#38bdf8" stroke-width="1.6" marker-end="url(#arrow)" stroke-dasharray="4 4"/>
  <line x1="170" y1="260" x2="300" y2="260" stroke="#38bdf8" stroke-width="1.6" marker-end="url(#arrow)"/>

  <!-- Row 3: Tools & Providers -->
  <rect x="120" y="360" width="230" height="90" rx="14" fill="url(#boxGrad)" stroke="#38bdf8" stroke-width="1.2"/>
  <text x="235" y="384" text-anchor="middle" fill="#e5e7eb" font-family="system-ui, sans-serif" font-size="13" font-weight="600">Tool Registry</text>
  <text x="235" y="402" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Filesystem • Terminal • Git • Semantic</text>

  <rect x="610" y="360" width="230" height="90" rx="14" fill="url(#boxGrad)" stroke="#6366f1" stroke-width="1.2"/>
  <text x="725" y="384" text-anchor="middle" fill="#e5e7eb" font-family="system-ui, sans-serif" font-size="13" font-weight="600">LLM Providers</text>
  <text x="725" y="402" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">OpenAI • Qwen • Others</text>
  <text x="725" y="420" text-anchor="middle" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Planner • Coder • Critic models</text>

  <!-- Arrows Agent -> Tools/Providers -->
  <line x1="420" y1="326" x2="260" y2="360" stroke="url(#accentGrad)" stroke-width="1.6" marker-end="url(#arrow)"/>
  <line x1="540" y1="326" x2="610" y2="360" stroke="url(#accentGrad)" stroke-width="1.6" marker-end="url(#arrow)"/>

  <!-- Legend -->
  <rect x="40" y="470" width="320" height="54" rx="10" fill="#020617" opacity="0.7" stroke="#1f2937" stroke-width="1"/>
  <text x="56" y="490" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="11">Legend:</text>
  <circle cx="70" cy="503" r="4" fill="#38bdf8"/>
  <text x="82" y="506" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="10">Control & data flow</text>
  <circle cx="190" cy="503" r="4" fill="#4ade80"/>
  <text x="202" y="506" fill="#9ca3af" font-family="system-ui, sans-serif" font-size="10">Metrics / observability links</text>
</svg>
```
