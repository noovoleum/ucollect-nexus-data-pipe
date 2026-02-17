package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// metricsRegistry keeps track of registered metrics to prevent duplicates
	metricsRegistry = make(map[string]bool)
	registryMu      sync.Mutex
)

// Metrics holds all Prometheus metrics for the data pipeline
type Metrics struct {
	EventsProcessed    *prometheus.CounterVec
	EventsErrored      *prometheus.CounterVec
	ProcessingDuration *prometheus.HistogramVec
	PipelineStatus     prometheus.Gauge
	SourceConnected    prometheus.Gauge
	SinkConnected      prometheus.Gauge
	registry           *prometheus.Registry
}

// NewMetrics creates and registers all pipeline metrics
func NewMetrics(pipelineName string) *Metrics {
	registryMu.Lock()
	defer registryMu.Unlock()

	// Check if metrics for this pipeline already exist
	if metricsRegistry[pipelineName] {
		// Return nil to signal that metrics already exist
		// Caller should handle this gracefully
		return nil
	}

	m := &Metrics{
		EventsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datapipe_events_processed_total",
				Help: "Total number of events processed by operation type",
			},
			[]string{"pipeline", "operation"},
		),
		EventsErrored: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "datapipe_events_errored_total",
				Help: "Total number of events that failed processing",
			},
			[]string{"pipeline", "component", "error_type"},
		),
		ProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "datapipe_event_processing_duration_seconds",
				Help:    "Time taken to process events",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"pipeline", "component"},
		),
		PipelineStatus: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datapipe_pipeline_status",
				Help: "Pipeline status: 1 for running, 0 for stopped",
				ConstLabels: prometheus.Labels{
					"pipeline": pipelineName,
				},
			},
		),
		SourceConnected: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datapipe_source_connected",
				Help: "Source connection status: 1 for connected, 0 for disconnected",
				ConstLabels: prometheus.Labels{
					"pipeline": pipelineName,
				},
			},
		),
		SinkConnected: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "datapipe_sink_connected",
				Help: "Sink connection status: 1 for connected, 0 for disconnected",
				ConstLabels: prometheus.Labels{
					"pipeline": pipelineName,
				},
			},
		),
	}

	metricsRegistry[pipelineName] = true
	return m
}

// RecordEventProcessed records a successfully processed event
func (m *Metrics) RecordEventProcessed(pipelineName, operation string) {
	m.EventsProcessed.WithLabelValues(pipelineName, operation).Inc()
}

// RecordEventError records an event processing error
func (m *Metrics) RecordEventError(pipelineName, component, errorType string) {
	m.EventsErrored.WithLabelValues(pipelineName, component, errorType).Inc()
}

// RecordProcessingDuration records the duration of event processing
func (m *Metrics) RecordProcessingDuration(pipelineName, component string, duration float64) {
	m.ProcessingDuration.WithLabelValues(pipelineName, component).Observe(duration)
}

// SetPipelineRunning sets the pipeline status to running (1) or stopped (0)
func (m *Metrics) SetPipelineRunning(running bool) {
	if running {
		m.PipelineStatus.Set(1)
	} else {
		m.PipelineStatus.Set(0)
	}
}

// SetSourceConnected sets the source connection status
func (m *Metrics) SetSourceConnected(connected bool) {
	if connected {
		m.SourceConnected.Set(1)
	} else {
		m.SourceConnected.Set(0)
	}
}

// SetSinkConnected sets the sink connection status
func (m *Metrics) SetSinkConnected(connected bool) {
	if connected {
		m.SinkConnected.Set(1)
	} else {
		m.SinkConnected.Set(0)
	}
}
