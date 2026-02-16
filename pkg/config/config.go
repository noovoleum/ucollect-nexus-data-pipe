package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the pipeline configuration
type Config struct {
	Pipeline    PipelineConfig    `json:"pipeline"`
	Source      SourceConfig      `json:"source"`
	Sink        SinkConfig        `json:"sink"`
	Transformer TransformerConfig `json:"transformer,omitempty"`
}

// PipelineConfig contains pipeline-level settings
type PipelineConfig struct {
	Name string     `json:"name"`
	Sync SyncConfig `json:"sync,omitempty"`
}

// SyncConfig contains synchronization settings
type SyncConfig struct {
	InitialSync      bool   `json:"initial_sync"`       // Enable initial sync
	ForceInitialSync bool   `json:"force_initial_sync"` // Force initial sync even if data exists in sink
	TimestampField   string `json:"timestamp_field"`    // Field name to use for timestamp-based sync
	BatchSize        int    `json:"batch_size"`         // Batch size for initial sync (default: 1000)
}

// SourceConfig contains source configuration
type SourceConfig struct {
	Type     string                 `json:"type"` // mongodb, convex, etc.
	Settings map[string]interface{} `json:"settings"`
}

// SinkConfig contains sink configuration
type SinkConfig struct {
	Type     string                 `json:"type"` // postgresql, clickhouse, etc.
	Settings map[string]interface{} `json:"settings"`
}

// TransformerConfig contains transformer configuration
type TransformerConfig struct {
	Type     string                 `json:"type"` // passthrough, fieldmapper, etc.
	Settings map[string]interface{} `json:"settings"`
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetString safely retrieves a string from settings
func (s SourceConfig) GetString(key string) string {
	if val, ok := s.Settings[key].(string); ok {
		return val
	}
	return ""
}

// GetString safely retrieves a string from settings
func (s SinkConfig) GetString(key string) string {
	if val, ok := s.Settings[key].(string); ok {
		return val
	}
	return ""
}

// GetString safely retrieves a string from settings
func (t TransformerConfig) GetString(key string) string {
	if val, ok := t.Settings[key].(string); ok {
		return val
	}
	return ""
}

// GetBool safely retrieves a bool from settings
func (t TransformerConfig) GetBool(key string) bool {
	if val, ok := t.Settings[key].(bool); ok {
		return val
	}
	return false
}

// GetBool safely retrieves a bool from settings
func (s SourceConfig) GetBool(key string) bool {
	if val, ok := s.Settings[key].(bool); ok {
		return val
	}
	return false
}

// GetBool safely retrieves a bool from settings
func (s SinkConfig) GetBool(key string) bool {
	if val, ok := s.Settings[key].(bool); ok {
		return val
	}
	return false
}
