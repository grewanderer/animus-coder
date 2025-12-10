# Strategy Engine

MyCodex can select different models per role (planner/coder/critic), apply per-run overrides, and fall back when a model misbehaves or exceeds cost budgets.

## Configuration
- `strategy.default_model`: fallback model id (optional; defaults to registry default).
- `strategy.planner_model`, `strategy.coder_model`, `strategy.critic_model`: per-role model ids.
- `strategy.overrides`: map of arbitrary role/hints to model ids.
- `strategy.fallbacks`: ordered list of model ids to try when selection or execution fails (walked sequentially).
- `strategy.max_expensive`: limit on uses of models marked `expensive` (per RunTask).
- Model configs can mark `expensive: true` for cost-aware strategies.
- CLI overrides: `--model` (coder), `--planner-model`, `--critic-model` for one-off runs.

## Behaviour
- AgentRunner uses the strategy engine to pick planner/coder/critic models; if unset, registry defaults are used.
- `fallbacks` are tried when selection fails, the chosen model errors, or when expensive model budgets are exceeded.
- Registry tracks `expensive` flags for cost-aware selection; budget enforcement is applied before each role call.
- Prometheus metrics record model usage and failures per role; debug logs note selections and fallbacks when a logger is attached.
- Backward compatible: when strategy fields are unset, the default model is used.
