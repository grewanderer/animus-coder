# Models & Providers

The LLM layer separates logical models from concrete providers so CLI/Agent code can address models by name.

## Concepts
- **Provider** – implementation that can execute chat completions (OpenAI-compatible HTTP, OpenRouter, vLLM/LM Studio via OpenAI, Ollama).
- **Model** – logical alias pointing to a provider plus physical model string (e.g., `default` → provider `openai`, model `gpt-4o-mini`).
- **Registry** – resolves a model name to its provider and settings.

## Configuration
Providers and models are defined in `config.yaml`:

```yaml
providers:
  openrouter:
    type: openrouter
    base_url: https://openrouter.ai/api/v1
    api_key: ${OPENROUTER_API_KEY}
    timeout: 30s
models:
  default:
    provider: openrouter
    model: openrouter/gpt-4o-mini
    temperature: 0.2
    max_tokens: 2048
    default: true
```

Supported provider types: `openai`, `openrouter`, `vllm`, `lmstudio`, `custom` (OpenAI-compatible), and `ollama`.

## API (internal/llm)
- `Provider` interface with `Chat` and `Stream`.
- `Registry` to register providers/models and resolve them.
- Providers implemented:
  - `openai.Provider`: OpenAI-compatible HTTP client.
  - `ollama.Provider`: minimal Ollama chat client.
- A `mock.Provider` is available for tests.

Streaming is currently simulated by returning the full response as a single chunk; true SSE/token streaming will be added alongside agent integration.
