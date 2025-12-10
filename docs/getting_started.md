# Getting Started

## Prerequisites
- Go 1.22+ installed.
- (Optional) `golangci-lint` for linting in CI/local runs.

## Setup
1) Copy `configs/config.example.yaml` to `configs/config.yaml` and update provider credentials.  
2) Export overrides via environment variables (`MYCODEX_LOGGING_LEVEL=debug`, `MYCODEX_AGENT_MAX_STEPS=12`, etc.).  
3) Build the binaries:

```bash
make build
```

## Usage

### CLI
```bash
./mycodex --version
./mycodex doctor --config configs/config.yaml
```

### Daemon
```bash
./mycodexd --config configs/config.yaml
# Health checks
curl -s http://localhost:8080/health
curl -s http://localhost:8080/metrics
```

## Development
- `make fmt` – format code via `gofmt`.
- `make lint` – run `golangci-lint` (requires tool installed).
- `make test` – run unit tests.
- `make run-daemon` – run daemon with example config.
