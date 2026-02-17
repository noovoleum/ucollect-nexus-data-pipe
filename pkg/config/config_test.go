package config

import (
	"os"
	"testing"
)

// TestLoadFromFile tests loading configuration from file
func TestLoadFromFile(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configJSON := `{
		"pipeline": {
			"name": "test-pipeline"
		},
		"source": {
			"type": "mongodb",
			"settings": {
				"uri": "mongodb://localhost:27017",
				"database": "testdb",
				"collection": "testcol"
			}
		},
		"sink": {
			"type": "postgresql",
			"settings": {
				"connection_string": "host=localhost",
				"table": "testtable"
			}
		}
	}`

	if _, err := tmpFile.Write([]byte(configJSON)); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load configuration
	cfg, err := LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	// Verify pipeline configuration
	if cfg.Pipeline.Name != "test-pipeline" {
		t.Errorf("Expected pipeline name 'test-pipeline', got '%s'", cfg.Pipeline.Name)
	}

	// Verify source configuration
	if cfg.Source.Type != "mongodb" {
		t.Errorf("Expected source type 'mongodb', got '%s'", cfg.Source.Type)
	}

	if cfg.Source.GetString("uri") != "mongodb://localhost:27017" {
		t.Errorf("Expected source URI 'mongodb://localhost:27017', got '%s'", cfg.Source.GetString("uri"))
	}

	// Verify sink configuration
	if cfg.Sink.Type != "postgresql" {
		t.Errorf("Expected sink type 'postgresql', got '%s'", cfg.Sink.Type)
	}

	if cfg.Sink.GetString("table") != "testtable" {
		t.Errorf("Expected sink table 'testtable', got '%s'", cfg.Sink.GetString("table"))
	}
}

// TestGetString tests the GetString helper method
func TestGetString(t *testing.T) {
	source := SourceConfig{
		Type: "test",
		Settings: map[string]interface{}{
			"string_value": "test",
			"int_value":    123,
		},
	}

	// Test existing string value
	if val := source.GetString("string_value"); val != "test" {
		t.Errorf("Expected 'test', got '%s'", val)
	}

	// Test non-string value
	if val := source.GetString("int_value"); val != "" {
		t.Errorf("Expected empty string for non-string value, got '%s'", val)
	}

	// Test non-existent key
	if val := source.GetString("nonexistent"); val != "" {
		t.Errorf("Expected empty string for nonexistent key, got '%s'", val)
	}
}
