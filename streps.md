🔥 P0: для Production Autonomous Agent
(1) Connect RPC bidi streaming

NDJSON — временно ok, но:

нет multiplexing,

нет cancellation propagation,

нет QoS.

(2) Reflection Phase (R из Plan-Act-Reflect)

Что нужно:

анализ результата step + tool events,

корректировка стратегии,

улучшение reasoning.

(3) Test Runner Integration

Чтобы агент мог:

запускать go test, npm test

анализировать ошибки тестов,

предлагать фиксы.

Пока тесты не интегрированы = агент не может точечно исправлять бизнес-логику.

(4) Structural Context Loader

Нужны:

автоматическое подгрузка файлов по запросам LLM,

file relevance scoring,

семантическое контекстное окно.

Пока контекст статичен.

🔥 P1: для полного production
(5) Docker sandbox

Для enterprise security.

(6) Semantic Token counter per model

Сейчас токены ≠ реальные tokens провайдера.

(7) Multi-model strategy

План:

planning = cheap reasoning (DeepSeek/Qwen)

execution = high precision model (GPT-5 or Claude)

(8) Full JSON schema enforcement for tools

Сейчас только basic validator.

(9) Full rollback orchestration

Ты сделал backup/restore, но:

❗ Нет multi-step rollback tree
❗ Нет patch lineage
❗ Нет conflict resolution