package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics bundles Prometheus collectors for the agent/daemon.
type Metrics struct {
	registry      *prometheus.Registry
	AgentRequests *prometheus.CounterVec
	AgentDuration *prometheus.HistogramVec
	AgentTokens   *prometheus.CounterVec
	ActiveSession *prometheus.GaugeVec
	TransportErrs *prometheus.CounterVec
	ModelUsage    *prometheus.CounterVec
	ModelFailures *prometheus.CounterVec
}

// NewMetrics constructs a metrics registry with agent collectors.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	reqs := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mycodex_agent_requests_total",
		Help: "Total agent run requests",
	}, []string{"finish_reason"})

	durs := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mycodex_agent_duration_seconds",
		Help:    "Agent run duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"finish_reason"})

	tokens := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mycodex_agent_tokens_total",
		Help: "Tokens (approx words) emitted by agent",
	}, []string{"finish_reason"})

	active := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mycodex_transport_active_sessions",
		Help: "Active streaming sessions by transport",
	}, []string{"transport"})

	trErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mycodex_transport_errors_total",
		Help: "Transport-level errors (handler/streaming) by transport and reason",
	}, []string{"transport", "reason"})

	modelUsage := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mycodex_model_usage_total",
		Help: "Model selections by role",
	}, []string{"role", "model"})

	modelFailures := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mycodex_model_failures_total",
		Help: "Model failures by role and model",
	}, []string{"role", "model"})

	reg.MustRegister(reqs, durs, tokens, active, trErrors, modelUsage, modelFailures)

	return &Metrics{
		registry:      reg,
		AgentRequests: reqs,
		AgentDuration: durs,
		AgentTokens:   tokens,
		ActiveSession: active,
		TransportErrs: trErrors,
		ModelUsage:    modelUsage,
		ModelFailures: modelFailures,
	}
}

// Registry returns the underlying Prometheus registry.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// RecordAgentRun records counts and duration.
func (m *Metrics) RecordAgentRun(finishReason string, duration time.Duration, tokenCount int) {
	if m == nil {
		return
	}
	if finishReason == "" {
		finishReason = "unknown"
	}
	m.AgentRequests.WithLabelValues(finishReason).Inc()
	m.AgentDuration.WithLabelValues(finishReason).Observe(duration.Seconds())
	m.AgentTokens.WithLabelValues(finishReason).Add(float64(tokenCount))
}

// IncActiveSessions increments the active session gauge.
func (m *Metrics) IncActiveSessions(transport string) {
	if m == nil {
		return
	}
	m.ActiveSession.WithLabelValues(transport).Inc()
}

// DecActiveSessions decrements the active session gauge.
func (m *Metrics) DecActiveSessions(transport string) {
	if m == nil {
		return
	}
	m.ActiveSession.WithLabelValues(transport).Dec()
}

// RecordTransportError records a transport-level error.
func (m *Metrics) RecordTransportError(transport, reason string) {
	if m == nil {
		return
	}
	if transport == "" {
		transport = "unknown"
	}
	if reason == "" {
		reason = "unknown"
	}
	m.TransportErrs.WithLabelValues(transport, reason).Inc()
}

// RecordModelUsage increments usage counter for a role/model selection.
func (m *Metrics) RecordModelUsage(role, model string) {
	if m == nil {
		return
	}
	if role == "" {
		role = "unknown"
	}
	if model == "" {
		model = "unknown"
	}
	m.ModelUsage.WithLabelValues(role, model).Inc()
}

// RecordModelFailure increments failure counter for a role/model selection.
func (m *Metrics) RecordModelFailure(role, model string) {
	if m == nil {
		return
	}
	if role == "" {
		role = "unknown"
	}
	if model == "" {
		model = "unknown"
	}
	m.ModelFailures.WithLabelValues(role, model).Inc()
}
