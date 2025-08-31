package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the Prometheus metrics
type Metrics struct {
	CommandsTotal     *prometheus.CounterVec
	CommandDuration   *prometheus.HistogramVec
	ConnectionsActive prometheus.Gauge
	KeysTotal         prometheus.Gauge
	MemoryUsage       prometheus.Gauge
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		CommandsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pulsedb_commands_total",
				Help: "Total number of commands executed",
			},
			[]string{"command", "status"},
		),
		CommandDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pulsedb_command_duration_seconds",
				Help:    "Duration of commands in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"command"},
		),
		ConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "pulsedb_connections_active",
				Help: "Number of active connections",
			},
		),
		KeysTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "pulsedb_keys_total",
				Help: "Total number of keys in the database",
			},
		),
		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "pulsedb_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),
	}
}

// IncrementCommand increments the command counter
func (m *Metrics) IncrementCommand(command, status string) {
	m.CommandsTotal.WithLabelValues(command, status).Inc()
}

// ObserveCommandDuration observes command duration
func (m *Metrics) ObserveCommandDuration(command string, duration float64) {
	m.CommandDuration.WithLabelValues(command).Observe(duration)
}

// SetActiveConnections sets the number of active connections
func (m *Metrics) SetActiveConnections(count float64) {
	m.ConnectionsActive.Set(count)
}

// SetKeysTotal sets the total number of keys
func (m *Metrics) SetKeysTotal(count float64) {
	m.KeysTotal.Set(count)
}

// SetMemoryUsage sets the memory usage
func (m *Metrics) SetMemoryUsage(bytes float64) {
	m.MemoryUsage.Set(bytes)
}
