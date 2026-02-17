package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	// Create a new registry for testing to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Clear default registry for test
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = oldRegistry
		// Clear our tracking registry
		registryMu.Lock()
		delete(metricsRegistry, "test-pipeline-new")
		registryMu.Unlock()
	}()
	
	m := NewMetrics("test-pipeline-new")
	
	if m == nil {
		t.Fatal("Expected metrics to be created")
	}
	
	if m.EventsProcessed == nil {
		t.Error("EventsProcessed counter should not be nil")
	}
	
	if m.EventsErrored == nil {
		t.Error("EventsErrored counter should not be nil")
	}
	
	if m.ProcessingDuration == nil {
		t.Error("ProcessingDuration histogram should not be nil")
	}
	
	if m.PipelineStatus == nil {
		t.Error("PipelineStatus gauge should not be nil")
	}
	
	if m.SourceConnected == nil {
		t.Error("SourceConnected gauge should not be nil")
	}
	
	if m.SinkConnected == nil {
		t.Error("SinkConnected gauge should not be nil")
	}
}

func TestRecordEventProcessed(t *testing.T) {
	// Create a new registry for testing
	reg := prometheus.NewRegistry()
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = oldRegistry
		registryMu.Lock()
		delete(metricsRegistry, "test-pipeline-events")
		registryMu.Unlock()
	}()
	
	m := NewMetrics("test-pipeline-events")
	
	if m == nil {
		t.Fatal("Expected metrics to be created")
	}
	
	// Record some events
	m.RecordEventProcessed("test-pipeline-events", "insert")
	m.RecordEventProcessed("test-pipeline-events", "insert")
	m.RecordEventProcessed("test-pipeline-events", "update")
	
	// Verify the counter was incremented
	count := testutil.CollectAndCount(m.EventsProcessed)
	if count == 0 {
		t.Error("Expected events to be recorded")
	}
}

func TestRecordEventError(t *testing.T) {
	reg := prometheus.NewRegistry()
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = oldRegistry
		registryMu.Lock()
		delete(metricsRegistry, "test-pipeline-errors")
		registryMu.Unlock()
	}()
	
	m := NewMetrics("test-pipeline-errors")
	
	if m == nil {
		t.Fatal("Expected metrics to be created")
	}
	
	// Record some errors
	m.RecordEventError("test-pipeline-errors", "source", "connection_error")
	m.RecordEventError("test-pipeline-errors", "sink", "write_error")
	
	// Verify the counter was incremented
	count := testutil.CollectAndCount(m.EventsErrored)
	if count == 0 {
		t.Error("Expected errors to be recorded")
	}
}

func TestSetPipelineRunning(t *testing.T) {
	reg := prometheus.NewRegistry()
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = oldRegistry
		registryMu.Lock()
		delete(metricsRegistry, "test-pipeline-running")
		registryMu.Unlock()
	}()
	
	m := NewMetrics("test-pipeline-running")
	
	if m == nil {
		t.Fatal("Expected metrics to be created")
	}
	
	// Test setting pipeline to running
	m.SetPipelineRunning(true)
	
	// Test setting pipeline to stopped
	m.SetPipelineRunning(false)
}

func TestSetSourceConnected(t *testing.T) {
	reg := prometheus.NewRegistry()
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = oldRegistry
		registryMu.Lock()
		delete(metricsRegistry, "test-pipeline-source")
		registryMu.Unlock()
	}()
	
	m := NewMetrics("test-pipeline-source")
	
	if m == nil {
		t.Fatal("Expected metrics to be created")
	}
	
	// Test setting source connected
	m.SetSourceConnected(true)
	m.SetSourceConnected(false)
}

func TestSetSinkConnected(t *testing.T) {
	reg := prometheus.NewRegistry()
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = oldRegistry
		registryMu.Lock()
		delete(metricsRegistry, "test-pipeline-sink")
		registryMu.Unlock()
	}()
	
	m := NewMetrics("test-pipeline-sink")
	
	if m == nil {
		t.Fatal("Expected metrics to be created")
	}
	
	// Test setting sink connected
	m.SetSinkConnected(true)
	m.SetSinkConnected(false)
}

func TestRecordProcessingDuration(t *testing.T) {
	reg := prometheus.NewRegistry()
	oldRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = oldRegistry
		registryMu.Lock()
		delete(metricsRegistry, "test-pipeline-duration")
		registryMu.Unlock()
	}()
	
	m := NewMetrics("test-pipeline-duration")
	
	if m == nil {
		t.Fatal("Expected metrics to be created")
	}
	
	// Record some durations
	m.RecordProcessingDuration("test-pipeline-duration", "source", 0.5)
	m.RecordProcessingDuration("test-pipeline-duration", "sink", 0.3)
	m.RecordProcessingDuration("test-pipeline-duration", "transform", 0.1)
	
	// Verify the histogram was updated
	count := testutil.CollectAndCount(m.ProcessingDuration)
	if count == 0 {
		t.Error("Expected durations to be recorded")
	}
}
