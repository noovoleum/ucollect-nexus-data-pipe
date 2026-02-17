package transform

import (
	"testing"
	"time"

	"github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
)

func TestFieldMapperBasicMapping(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{Source: "firstName", Destination: "first_name"},
			{Source: "lastName", Destination: "last_name"},
		},
		IncludeAll: false,
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	event := pipeline.Event{
		Data: map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       30,
		},
	}

	result, err := mapper.Transform(event)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.Data["first_name"] != "John" {
		t.Errorf("Expected first_name=John, got %v", result.Data["first_name"])
	}
	if result.Data["last_name"] != "Doe" {
		t.Errorf("Expected last_name=Doe, got %v", result.Data["last_name"])
	}
	if _, exists := result.Data["age"]; exists {
		t.Errorf("age should not be included when IncludeAll is false")
	}
}

func TestFieldMapperIncludeAll(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{Source: "name", Destination: "full_name"},
		},
		IncludeAll: true,
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	event := pipeline.Event{
		Data: map[string]interface{}{
			"name": "John Doe",
			"age":  30,
			"city": "NYC",
		},
	}

	result, err := mapper.Transform(event)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.Data["full_name"] != "John Doe" {
		t.Errorf("Expected full_name=John Doe, got %v", result.Data["full_name"])
	}
	if result.Data["age"] != 30 {
		t.Errorf("Expected age=30, got %v", result.Data["age"])
	}
	if result.Data["city"] != "NYC" {
		t.Errorf("Expected city=NYC, got %v", result.Data["city"])
	}
}

func TestFieldMapperExcludeFields(t *testing.T) {
	config := FieldMapperConfig{
		IncludeAll:    true,
		ExcludeFields: []string{"password", "ssn"},
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	event := pipeline.Event{
		Data: map[string]interface{}{
			"name":     "John",
			"password": "secret123",
			"ssn":      "123-45-6789",
			"email":    "john@example.com",
		},
	}

	result, err := mapper.Transform(event)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if _, exists := result.Data["password"]; exists {
		t.Errorf("password should be excluded")
	}
	if _, exists := result.Data["ssn"]; exists {
		t.Errorf("ssn should be excluded")
	}
	if result.Data["name"] != "John" {
		t.Errorf("name should be included")
	}
	if result.Data["email"] != "john@example.com" {
		t.Errorf("email should be included")
	}
}

func TestFieldMapperFormatting(t *testing.T) {
	tests := []struct {
		name     string
		mapping  FieldMapping
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "uppercase",
			mapping:  FieldMapping{Source: "name", Format: "uppercase"},
			input:    "john doe",
			expected: "JOHN DOE",
		},
		{
			name:     "lowercase",
			mapping:  FieldMapping{Source: "name", Format: "lowercase"},
			input:    "JOHN DOE",
			expected: "john doe",
		},
		{
			name:     "trim",
			mapping:  FieldMapping{Source: "name", Format: "trim"},
			input:    "  john  ",
			expected: "john",
		},
		{
			name:     "string",
			mapping:  FieldMapping{Source: "age", Format: "string"},
			input:    30,
			expected: "30",
		},
		{
			name:     "int",
			mapping:  FieldMapping{Source: "age", Format: "int"},
			input:    "30",
			expected: 30,
		},
		{
			name:     "float",
			mapping:  FieldMapping{Source: "price", Format: "float"},
			input:    "29.99",
			expected: 29.99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := FieldMapperConfig{
				Mappings: []FieldMapping{tt.mapping},
			}

			mapper, err := NewFieldMapper(config)
			if err != nil {
				t.Fatalf("Failed to create mapper: %v", err)
			}

			event := pipeline.Event{
				Data: map[string]interface{}{
					tt.mapping.Source: tt.input,
				},
			}

			result, err := mapper.Transform(event)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Transform() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				destName := tt.mapping.Destination
				if destName == "" {
					destName = tt.mapping.Source
				}
				if result.Data[destName] != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result.Data[destName])
				}
			}
		})
	}
}

func TestFieldMapperExtraction(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{
				Source:      "email",
				Destination: "username",
				Extract:     `^([^@]+)@`,
			},
			{
				Source:      "phone",
				Destination: "area_code",
				Extract:     `^\((\d{3})\)`,
			},
		},
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	event := pipeline.Event{
		Data: map[string]interface{}{
			"email": "john.doe@example.com",
			"phone": "(555)123-4567",
		},
	}

	result, err := mapper.Transform(event)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.Data["username"] != "john.doe" {
		t.Errorf("Expected username=john.doe, got %v", result.Data["username"])
	}
	if result.Data["area_code"] != "555" {
		t.Errorf("Expected area_code=555, got %v", result.Data["area_code"])
	}
}

func TestFieldMapperDefaultValues(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{Source: "status", Destination: "status", Default: "pending"},
			{Source: "priority", Destination: "priority", Default: "normal"},
		},
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	event := pipeline.Event{
		Data: map[string]interface{}{
			"name": "Test",
			// status and priority are missing
		},
	}

	result, err := mapper.Transform(event)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.Data["status"] != "pending" {
		t.Errorf("Expected status=pending (default), got %v", result.Data["status"])
	}
	if result.Data["priority"] != "normal" {
		t.Errorf("Expected priority=normal (default), got %v", result.Data["priority"])
	}
}

func TestFieldMapperRequiredFields(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{Source: "id", Destination: "id", Required: true},
			{Source: "name", Destination: "name", Required: true},
		},
		StrictMode: true,
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	// Missing required field
	event := pipeline.Event{
		Data: map[string]interface{}{
			"id": "123",
			// name is missing
		},
	}

	_, err = mapper.Transform(event)
	if err == nil {
		t.Errorf("Expected error for missing required field, got nil")
	}
}

func TestFieldMapperStrictMode(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{Source: "age", Destination: "age", Format: "int"},
		},
		StrictMode: true,
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	// Invalid int value
	event := pipeline.Event{
		Data: map[string]interface{}{
			"age": "not-a-number",
		},
	}

	_, err = mapper.Transform(event)
	if err == nil {
		t.Errorf("Expected error in strict mode for invalid format, got nil")
	}
}

func TestFieldMapperNonStrictMode(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{Source: "age", Destination: "age", Format: "int"},
			{Source: "name", Destination: "name"},
		},
		StrictMode: false,
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	// Invalid int value - should skip this field but continue
	event := pipeline.Event{
		Data: map[string]interface{}{
			"age":  "not-a-number",
			"name": "John",
		},
	}

	result, err := mapper.Transform(event)
	if err != nil {
		t.Fatalf("Transform should not fail in non-strict mode: %v", err)
	}

	// age should be skipped due to error
	if _, exists := result.Data["age"]; exists {
		t.Errorf("age should be skipped due to formatting error")
	}
	// name should still be processed
	if result.Data["name"] != "John" {
		t.Errorf("name should still be processed")
	}
}

func TestFieldMapperDateParsing(t *testing.T) {
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{Source: "created_at", Format: "date"},
		},
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	tests := []string{
		"2023-01-15T10:30:00Z",
		"2023-01-15 10:30:00",
		"2023-01-15",
	}

	for _, dateStr := range tests {
		event := pipeline.Event{
			Data: map[string]interface{}{
				"created_at": dateStr,
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed for date %s: %v", dateStr, err)
		}

		if _, ok := result.Data["created_at"].(time.Time); !ok {
			t.Errorf("Expected time.Time for date %s, got %T", dateStr, result.Data["created_at"])
		}
	}
}

func TestFieldMapperEdgeCases(t *testing.T) {
	t.Run("nil values", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "nullable", Destination: "nullable"},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}
		event := pipeline.Event{
			Data: map[string]interface{}{
				"nullable": nil,
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		// nil values should be skipped
		if _, exists := result.Data["nullable"]; exists {
			t.Errorf("nil values should be skipped")
		}
	})

	t.Run("empty strings", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "empty", Destination: "empty"},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}
		event := pipeline.Event{
			Data: map[string]interface{}{
				"empty": "",
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if result.Data["empty"] != "" {
			t.Errorf("Empty strings should be preserved")
		}
	})

	t.Run("numeric zero values", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "count", Destination: "count"},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}
		event := pipeline.Event{
			Data: map[string]interface{}{
				"count": 0,
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if result.Data["count"] != 0 {
			t.Errorf("Zero values should be preserved")
		}
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "field", Extract: "[invalid(regex"},
			},
		}

		_, err := NewFieldMapper(config)
		if err == nil {
			t.Errorf("Expected error for invalid regex pattern")
		}
	})
}

func TestFieldMapperBooleanFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"true string", "true", true},
		{"false string", "false", false},
		{"yes", "yes", true},
		{"no", "no", false},
		{"y", "y", true},
		{"n", "n", false},
		{"1", "1", true},
		{"0", "0", false},
		{"True", "True", true},
		{"FALSE", "FALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := FieldMapperConfig{
				Mappings: []FieldMapping{
					{Source: "active", Format: "bool"},
				},
			}

			mapper, err := NewFieldMapper(config)
			if err != nil {
				t.Fatalf("Failed to create mapper: %v", err)
			}

			event := pipeline.Event{
				Data: map[string]interface{}{
					"active": tt.input,
				},
			}

			result, err := mapper.Transform(event)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			if result.Data["active"] != tt.expected {
				t.Errorf("Expected active=%v, got %v", tt.expected, result.Data["active"])
			}
		})
	}

	t.Run("invalid boolean value", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "active", Format: "bool"},
			},
			StrictMode: false,
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}

		event := pipeline.Event{
			Data: map[string]interface{}{
				"active": "not-a-boolean",
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform should not fail in non-strict mode: %v", err)
		}

		// Field should be skipped
		if _, exists := result.Data["active"]; exists {
			t.Errorf("Invalid boolean field should be skipped in non-strict mode")
		}
	})
}

func TestFieldMapperNestedFields(t *testing.T) {
	t.Run("simple nested path", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "address", NestedPath: "address.city", Destination: "city"},
				{Source: "address", NestedPath: "address.zipcode", Destination: "zip"},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}

		event := pipeline.Event{
			Data: map[string]interface{}{
				"name": "John",
				"address": map[string]interface{}{
					"city":    "San Francisco",
					"zipcode": "94102",
					"state":   "CA",
				},
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if result.Data["city"] != "San Francisco" {
			t.Errorf("Expected city=San Francisco, got %v", result.Data["city"])
		}
		if result.Data["zip"] != "94102" {
			t.Errorf("Expected zip=94102, got %v", result.Data["zip"])
		}
	})

	t.Run("deeply nested path", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "user", NestedPath: "user.profile.contact.email", Destination: "email"},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}

		event := pipeline.Event{
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id": 123,
					"profile": map[string]interface{}{
						"contact": map[string]interface{}{
							"email": "john@example.com",
							"phone": "555-1234",
						},
					},
				},
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if result.Data["email"] != "john@example.com" {
			t.Errorf("Expected email=john@example.com, got %v", result.Data["email"])
		}
	})

	t.Run("missing nested path", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "address", NestedPath: "address.city", Destination: "city", Default: "Unknown"},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}

		event := pipeline.Event{
			Data: map[string]interface{}{
				"name": "John",
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if result.Data["city"] != "Unknown" {
			t.Errorf("Expected city=Unknown (default), got %v", result.Data["city"])
		}
	})

	t.Run("nested with format", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{Source: "user", NestedPath: "user.email", Destination: "email", Format: "lowercase"},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}

		event := pipeline.Event{
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"email": "JOHN@EXAMPLE.COM",
				},
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if result.Data["email"] != "john@example.com" {
			t.Errorf("Expected email=john@example.com, got %v", result.Data["email"])
		}
	})
}

func TestFieldMapperMultipleExtractFromSameSource(t *testing.T) {
	// Test for regex extractor collision fix - multiple mappings from same source with different extract patterns
	config := FieldMapperConfig{
		Mappings: []FieldMapping{
			{
				Source:      "email",
				Destination: "username",
				Extract:     "^([^@]+)@",
			},
			{
				Source:      "email",
				Destination: "domain",
				Extract:     "@(.+)$",
			},
		},
	}

	mapper, err := NewFieldMapper(config)
	if err != nil {
		t.Fatalf("Failed to create mapper: %v", err)
	}

	event := pipeline.Event{
		Data: map[string]interface{}{
			"email": "john.doe@example.com",
		},
	}

	result, err := mapper.Transform(event)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if result.Data["username"] != "john.doe" {
		t.Errorf("Expected username=john.doe, got %v", result.Data["username"])
	}
	if result.Data["domain"] != "example.com" {
		t.Errorf("Expected domain=example.com, got %v", result.Data["domain"])
	}
}

func TestFieldMapperJSONBCompatibility(t *testing.T) {
	// Test that nested objects can be extracted as-is for JSONB columns
	t.Run("extract nested object", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{
					Source:      "user",
					Destination: "user_data",
					// No nested_path - extract entire nested object
				},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}

		event := pipeline.Event{
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"id":    123,
					"email": "john@example.com",
					"profile": map[string]interface{}{
						"age":  30,
						"city": "SF",
					},
				},
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		// Should preserve the entire nested structure
		userData, ok := result.Data["user_data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected user_data to be map[string]interface{}, got %T", result.Data["user_data"])
		}

		if userData["id"] != 123 {
			t.Errorf("Expected id=123, got %v", userData["id"])
		}

		profile, ok := userData["profile"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected profile to be map[string]interface{}, got %T", userData["profile"])
		}

		if profile["city"] != "SF" {
			t.Errorf("Expected city=SF, got %v", profile["city"])
		}
	})

	t.Run("mix nested extraction and object preservation", func(t *testing.T) {
		config := FieldMapperConfig{
			Mappings: []FieldMapping{
				{
					Source:      "user",
					NestedPath:  "user.email",
					Destination: "email",
					Format:      "lowercase",
				},
				{
					Source:      "user",
					Destination: "metadata",
					// Extract entire user object for JSONB column
				},
			},
		}

		mapper, err := NewFieldMapper(config)
		if err != nil {
			t.Fatalf("Failed to create mapper: %v", err)
		}

		event := pipeline.Event{
			Data: map[string]interface{}{
				"user": map[string]interface{}{
					"email": "JOHN@EXAMPLE.COM",
					"id":    456,
				},
			},
		}

		result, err := mapper.Transform(event)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		// Email should be extracted and formatted
		if result.Data["email"] != "john@example.com" {
			t.Errorf("Expected email=john@example.com, got %v", result.Data["email"])
		}

		// Entire user object should be preserved
		metadata, ok := result.Data["metadata"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected metadata to be map[string]interface{}, got %T", result.Data["metadata"])
		}

		if metadata["id"] != 456 {
			t.Errorf("Expected id=456, got %v", metadata["id"])
		}
	})
}
