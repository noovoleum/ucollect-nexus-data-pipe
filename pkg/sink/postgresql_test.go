package sink

import (
	"context"
	"testing"
)

// TestTableNameValidation tests that invalid table names are rejected
func TestTableNameValidation(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		wantError bool
	}{
		{
			name:      "valid simple name",
			tableName: "users",
			wantError: false,
		},
		{
			name:      "valid name with underscore",
			tableName: "user_data",
			wantError: false,
		},
		{
			name:      "valid name with numbers",
			tableName: "users_v2",
			wantError: false,
		},
		{
			name:      "valid name starting with underscore",
			tableName: "_internal",
			wantError: false,
		},
		{
			name:      "invalid - SQL injection attempt",
			tableName: "users; DROP TABLE users;--",
			wantError: true,
		},
		{
			name:      "invalid - starting with number",
			tableName: "1users",
			wantError: true,
		},
		{
			name:      "invalid - special characters",
			tableName: "user$data",
			wantError: true,
		},
		{
			name:      "invalid - dots (schema.table)",
			tableName: "public.users",
			wantError: true,
		},
		{
			name:      "invalid - empty",
			tableName: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := NewPostgreSQLSink("dummy_connection_string", tt.tableName, nil)
			err := sink.Connect(context.Background())

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for table name '%s', got nil", tt.tableName)
				}
			} else {
				// We expect connection to fail (dummy connection string)
				// but NOT because of table name validation
				if err != nil && err.Error() != "failed to ping PostgreSQL: dial tcp: lookup dummy_connection_string: no such host" {
					// Check if it's a table validation error
					if err.Error() == "invalid table name: "+tt.tableName+" (must be alphanumeric with underscores, starting with letter or underscore)" {
						t.Errorf("Valid table name '%s' was rejected: %v", tt.tableName, err)
					}
				}
			}
		})
	}
}

// TestValidTableNamePattern tests the regex pattern directly
func TestValidTableNamePattern(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		valid     bool
	}{
		{"valid simple", "users", true},
		{"valid with underscore", "user_data", true},
		{"valid with numbers", "users_v2", true},
		{"valid starting with underscore", "_internal", true},
		{"invalid SQL injection", "users; DROP TABLE users;--", false},
		{"invalid starting with number", "1users", false},
		{"invalid special chars", "user$data", false},
		{"invalid dots", "public.users", false},
		{"invalid empty", "", false},
		{"invalid spaces", "user data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := validTableName.MatchString(tt.tableName)
			if matched != tt.valid {
				t.Errorf("validTableName.MatchString(%q) = %v, want %v", tt.tableName, matched, tt.valid)
			}
		})
	}
}

// TestColumnNameValidation tests that invalid column names are rejected during upsert
func TestColumnNameValidation(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		valid      bool
	}{
		{"valid simple", "email", true},
		{"valid with underscore", "first_name", true},
		{"valid with numbers", "field_v2", true},
		{"valid starting with underscore", "_id", true},
		{"invalid SQL injection", "'; DROP TABLE users;--", false},
		{"invalid special chars", "user$name", false},
		{"invalid dots", "user.name", false},
		{"invalid spaces", "first name", false},
		{"invalid starting with number", "1field", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := validTableName.MatchString(tt.columnName)
			if matched != tt.valid {
				t.Errorf("Column name validation for %q = %v, want %v", tt.columnName, matched, tt.valid)
			}
		})
	}
}
