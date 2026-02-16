package transform

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
)

// FieldMapping defines how to map a single field
type FieldMapping struct {
	Source      string `json:"source"`      // Source field name
	Destination string `json:"destination"` // Destination field name
	Format      string `json:"format"`      // Format type: "string", "int", "float", "date", "uppercase", "lowercase", "trim"
	Default     string `json:"default"`     // Default value if source is missing or null
	Required    bool   `json:"required"`    // If true, error if field is missing
	Extract     string `json:"extract"`     // Regex pattern to extract from source value
}

// FieldMapperConfig contains field mapping configuration
type FieldMapperConfig struct {
	Mappings      []FieldMapping `json:"mappings"`
	IncludeAll    bool           `json:"include_all"`    // Include all unmapped fields
	ExcludeFields []string       `json:"exclude_fields"` // Fields to exclude (if include_all is true)
	StrictMode    bool           `json:"strict_mode"`    // Fail on any mapping error
}

// FieldMapper is a transformer that maps and formats fields
type FieldMapper struct {
	config     FieldMapperConfig
	extractors map[string]*regexp.Regexp
}

// NewFieldMapper creates a new field mapper transformer
func NewFieldMapper(config FieldMapperConfig) (*FieldMapper, error) {
	fm := &FieldMapper{
		config:     config,
		extractors: make(map[string]*regexp.Regexp),
	}

	// Compile regex patterns for extraction
	for _, mapping := range config.Mappings {
		if mapping.Extract != "" {
			re, err := regexp.Compile(mapping.Extract)
			if err != nil {
				return nil, fmt.Errorf("invalid extract pattern for field %s: %w", mapping.Source, err)
			}
			fm.extractors[mapping.Source] = re
		}
	}

	return fm, nil
}

// Transform transforms an event by mapping and formatting fields
func (f *FieldMapper) Transform(event pipeline.Event) (pipeline.Event, error) {
	newData := make(map[string]interface{})
	errors := make([]string, 0)

	// Apply mappings
	for _, mapping := range f.config.Mappings {
		value, exists := event.Data[mapping.Source]

		// Handle missing required fields
		if !exists || value == nil {
			if mapping.Required {
				errors = append(errors, fmt.Sprintf("required field '%s' is missing", mapping.Source))
				if f.config.StrictMode {
					return event, fmt.Errorf("required field '%s' is missing", mapping.Source)
				}
			}
			// Use default value if provided
			if mapping.Default != "" {
				value = mapping.Default
			} else {
				continue
			}
		}

		// Extract using regex if specified
		if extractor, ok := f.extractors[mapping.Source]; ok {
			strValue := fmt.Sprintf("%v", value)
			matches := extractor.FindStringSubmatch(strValue)
			if len(matches) > 1 {
				value = matches[1] // Use first capture group
			} else if len(matches) > 0 {
				value = matches[0] // Use full match
			} else {
				if mapping.Required && f.config.StrictMode {
					return event, fmt.Errorf("extraction pattern failed for field '%s'", mapping.Source)
				}
				continue
			}
		}

		// Format the value
		formattedValue, err := f.formatValue(value, mapping.Format)
		if err != nil {
			errors = append(errors, fmt.Sprintf("formatting error for field '%s': %v", mapping.Source, err))
			if f.config.StrictMode {
				return event, fmt.Errorf("formatting error for field '%s': %w", mapping.Source, err)
			}
			continue
		}

		// Use destination name if provided, otherwise use source name
		destName := mapping.Destination
		if destName == "" {
			destName = mapping.Source
		}
		newData[destName] = formattedValue
	}

	// Handle unmapped fields
	if f.config.IncludeAll {
		excludeMap := make(map[string]bool)
		for _, field := range f.config.ExcludeFields {
			excludeMap[field] = true
		}

		// Create map of mapped source fields
		mappedSources := make(map[string]bool)
		for _, mapping := range f.config.Mappings {
			mappedSources[mapping.Source] = true
		}

		// Include unmapped fields
		for key, value := range event.Data {
			if !mappedSources[key] && !excludeMap[key] {
				newData[key] = value
			}
		}
	}

	// Log non-fatal errors if any
	if len(errors) > 0 && !f.config.StrictMode {
		// In a production setting, these would be logged
		// For now, they are collected but not returned as errors
		for _, errMsg := range errors {
			// TODO: Add logger support to FieldMapper for better error visibility
			_ = errMsg
		}
	}

	event.Data = newData
	return event, nil
}

// formatValue formats a value according to the specified format
func (f *FieldMapper) formatValue(value interface{}, format string) (interface{}, error) {
	if format == "" {
		return value, nil
	}

	strValue := fmt.Sprintf("%v", value)

	switch format {
	case "string":
		return strValue, nil

	case "int":
		var intVal int
		_, err := fmt.Sscanf(strValue, "%d", &intVal)
		if err != nil {
			return nil, fmt.Errorf("cannot convert to int: %w", err)
		}
		return intVal, nil

	case "float":
		var floatVal float64
		_, err := fmt.Sscanf(strValue, "%f", &floatVal)
		if err != nil {
			return nil, fmt.Errorf("cannot convert to float: %w", err)
		}
		return floatVal, nil

	case "date", "datetime":
		// Try parsing common date formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, layout := range formats {
			if t, err := time.Parse(layout, strValue); err == nil {
				return t, nil
			}
		}
		return nil, fmt.Errorf("cannot parse date: %s", strValue)

	case "uppercase":
		return strings.ToUpper(strValue), nil

	case "lowercase":
		return strings.ToLower(strValue), nil

	case "trim":
		return strings.TrimSpace(strValue), nil

	case "titlecase":
		// Simple implementation without deprecated strings.Title
		words := strings.Fields(strings.ToLower(strValue))
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + word[1:]
			}
		}
		return strings.Join(words, " "), nil

	default:
		return value, nil
	}
}
