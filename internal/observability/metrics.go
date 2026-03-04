package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the Rubix node
type Metrics struct {
	LedgerWritesTotal          *prometheus.CounterVec
	LedgerWriteErrorsTotal     *prometheus.CounterVec
	LedgerWriteDurationSeconds prometheus.Histogram
	TokenLockWaitSeconds       prometheus.Histogram
	StateCorruptionDetectedTotal prometheus.Counter
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)
	return &Metrics{
		LedgerWritesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "ledger_writes_total",
			Help: "Total number of ledger write operations",
		}, []string{"status"}),
		LedgerWriteErrorsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "ledger_write_errors_total",
			Help: "Total number of ledger write errors by type",
		}, []string{"error_type"}),
		LedgerWriteDurationSeconds: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "ledger_write_duration_seconds",
			Help:    "Duration of ledger write operations in seconds",
			Buckets: []float64{.001, .005, .01, .05, .1, .5, 1, 5},
		}),
		TokenLockWaitSeconds: factory.NewHistogram(prometheus.HistogramOpts{
			Name:    "token_lock_wait_seconds",
			Help:    "Time spent waiting for token row locks",
			Buckets: []float64{.001, .005, .01, .05, .1, .5, 1},
		}),
		StateCorruptionDetectedTotal: factory.NewCounter(prometheus.CounterOpts{
			Name: "state_corruption_detected_total",
			Help: "Number of state corruption events detected at startup",
		}),
	}
}

// RegisterMetrics creates metrics using the provided registry
func RegisterMetrics(reg prometheus.Registerer) *Metrics {
	return NewMetrics(reg)
}
