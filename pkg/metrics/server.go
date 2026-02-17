package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server provides HTTP endpoints for metrics and health checks
type Server struct {
	server *http.Server
	logger *log.Logger
	health HealthChecker
}

// HealthChecker interface for checking pipeline health
type HealthChecker interface {
	IsHealthy() bool
	GetStatus() HealthStatus
}

// HealthStatus represents the health status of the pipeline
type HealthStatus struct {
	Healthy          bool   `json:"healthy"`
	PipelineRunning  bool   `json:"pipeline_running"`
	SourceConnected  bool   `json:"source_connected"`
	SinkConnected    bool   `json:"sink_connected"`
	LastEventTime    string `json:"last_event_time,omitempty"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
}

// NewServer creates a new metrics HTTP server
func NewServer(addr string, health HealthChecker, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}

	mux := http.NewServeMux()
	
	s := &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		logger: logger,
		health: health,
	}

	// Register handlers
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/ready", s.readinessHandler)
	mux.HandleFunc("/", s.rootHandler)

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Printf("Starting metrics server on %s", s.server.Addr)
	
	// Create a channel to receive startup errors
	errChan := make(chan error, 1)
	
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
	
	// Wait a brief moment to catch immediate errors (e.g., port already in use)
	select {
	case err := <-errChan:
		return fmt.Errorf("failed to start server: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
		return nil
	}
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Println("Shutting down metrics server...")
	return s.server.Shutdown(ctx)
}

// healthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if s.health == nil {
		http.Error(w, "Health checker not configured", http.StatusInternalServerError)
		return
	}

	status := s.health.GetStatus()
	
	w.Header().Set("Content-Type", "application/json")
	
	if status.Healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	if err := json.NewEncoder(w).Encode(status); err != nil {
		s.logger.Printf("Error encoding health status: %v", err)
	}
}

// readinessHandler handles readiness probe requests
func (s *Server) readinessHandler(w http.ResponseWriter, r *http.Request) {
	if s.health == nil {
		http.Error(w, "Health checker not configured", http.StatusInternalServerError)
		return
	}

	// For readiness, we check if connections are established
	status := s.health.GetStatus()
	
	if status.SourceConnected && status.SinkConnected {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ready")); err != nil {
			s.logger.Printf("Error writing readiness response: %v", err)
		}
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte("not ready")); err != nil {
			s.logger.Printf("Error writing readiness response: %v", err)
		}
	}
}

// rootHandler provides basic information about available endpoints
func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Data Pipe Metrics</title>
</head>
<body>
    <h1>Data Pipe Metrics & Monitoring</h1>
    <ul>
        <li><a href="/metrics">Metrics (Prometheus format)</a></li>
        <li><a href="/health">Health Check (JSON)</a></li>
        <li><a href="/ready">Readiness Probe</a></li>
    </ul>
</body>
</html>
`
	if _, err := w.Write([]byte(html)); err != nil {
		s.logger.Printf("Error writing root response: %v", err)
	}
}
