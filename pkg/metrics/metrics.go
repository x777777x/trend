package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Master metrics
var (
	// MasterUp — 1 when the master process is alive
	MasterUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "master_up",
		Help: "Whether the master process is up (1 = alive).",
	})

	// MasterStaleTasks — per (cluster_name, task_type) count of stale calc_instance rows
	MasterStaleTasks = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "master_stale_tasks",
		Help: "Number of calc_instance entries whose last_time has not been updated beyond the stale threshold.",
	}, []string{"cluster_name", "task_type"})
)

// Worker metrics
var (
	// WorkerUp — 1 when the worker process is alive
	WorkerUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "worker_up",
		Help: "Whether the worker process is up (1 = alive).",
	})

	// WorkerTaskFailuresTotal — counter incremented on each task execution failure
	WorkerTaskFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "worker_task_failures_total",
		Help: "Total number of task execution failures on this worker.",
	}, []string{"cluster_name", "task_type"})
)

var registered bool

// RegisterAll registers all metrics with the default prometheus registry.
// Idempotent — safe to call multiple times.
func RegisterAll() {
	if registered {
		return
	}
	prometheus.MustRegister(MasterUp)
	prometheus.MustRegister(MasterStaleTasks)
	prometheus.MustRegister(WorkerUp)
	prometheus.MustRegister(WorkerTaskFailuresTotal)
	registered = true
}

// Handler returns an http.Handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
