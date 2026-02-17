# Field Mapping and Transformation Guide

This guide explains how to use the field mapper transformer to rename, format, and select fields during data pipeline processing.

## Overview

The field mapper transformer allows you to:
- **Rename fields** - Map source field names to different destination names
- **Format values** - Convert data types and apply transformations
- **Extract data** - Use regex patterns to extract portions of field values
- **Set defaults** - Provide default values for missing fields
- **Filter fields** - Include only specific fields or exclude sensitive data
- **Handle edge cases** - Deal with null values, empty strings, and type conversions

## Configuration

Add a transformer section to your configuration file:

```json
{
  "pipeline": {"name": "my-pipeline"},
  "source": {...},
  "sink": {...},
  "transformer": {
    "type": "fieldmapper",
    "settings": {
      "mappings": [...],
      "include_all": false,
      "exclude_fields": [],
      "strict_mode": false
    }
  }
}
```

### Configuration Options

#### `mappings` (array of objects)

Each mapping defines how to transform a source field:

```json
{
  "source": "firstName",           // Source field name (required)
  "destination": "first_name",     // Destination field name (optional, defaults to source)
  "format": "lowercase",           // Format transformation (optional)
  "default": "N/A",               // Default value if field is missing (optional)
  "required": false,              // Fail if field is missing (optional, default: false)
  "extract": "^([^@]+)@"          // Regex pattern to extract value (optional)
}
```

#### `include_all` (boolean)

- `false` (default): Only include mapped fields in output
- `true`: Include all source fields, plus mapped fields

#### `exclude_fields` (array of strings)

List of field names to exclude from output (only applies when `include_all: true`):

```json
"exclude_fields": ["password", "ssn", "credit_card"]
```

#### `strict_mode` (boolean)

- `false` (default): Skip fields with errors, continue processing
- `true`: Fail entire transformation if any field has an error

## Format Types

### String Formats

- **`uppercase`** - Convert to uppercase
  ```json
  {"source": "name", "format": "uppercase"}
  // "john doe" → "JOHN DOE"
  ```

- **`lowercase`** - Convert to lowercase
  ```json
  {"source": "email", "format": "lowercase"}
  // "John@EXAMPLE.com" → "john@example.com"
  ```

- **`trim`** - Remove leading/trailing whitespace
  ```json
  {"source": "name", "format": "trim"}
  // "  John  " → "John"
  ```

- **`titlecase`** - Capitalize first letter of each word
  ```json
  {"source": "name", "format": "titlecase"}
  // "john doe" → "John Doe"
  ```

### Type Conversions

- **`string`** - Convert to string
  ```json
  {"source": "age", "format": "string"}
  // 30 → "30"
  ```

- **`int`** - Convert to integer
  ```json
  {"source": "age", "format": "int"}
  // "30" → 30
  ```

- **`float`** - Convert to floating point
  ```json
  {"source": "price", "format": "float"}
  // "29.99" → 29.99
  ```

- **`date`** or **`datetime`** - Parse date/time
  ```json
  {"source": "created_at", "format": "date"}
  // "2023-01-15T10:30:00Z" → time.Time object
  ```
  
  Supported date formats:
  - RFC3339: `2006-01-02T15:04:05Z07:00`
  - ISO datetime: `2006-01-02 15:04:05`
  - ISO date: `2006-01-02`

## Field Extraction with Regex

Extract portions of field values using regex patterns:

```json
{
  "source": "email",
  "destination": "username",
  "extract": "^([^@]+)@"
}
// "john.doe@example.com" → "john.doe"
```

```json
{
  "source": "phone",
  "destination": "area_code",
  "extract": "^\\((\\d{3})\\)"
}
// "(555)123-4567" → "555"
```

**Note:** Use the first capture group `()` for extracted value.

## Examples

### Example 1: Basic Field Renaming

```json
{
  "transformer": {
    "type": "fieldmapper",
    "settings": {
      "mappings": [
        {"source": "firstName", "destination": "first_name"},
        {"source": "lastName", "destination": "last_name"},
        {"source": "emailAddress", "destination": "email"}
      ],
      "include_all": false
    }
  }
}
```

**Input:**
```json
{"firstName": "John", "lastName": "Doe", "emailAddress": "john@example.com", "age": 30}
```

**Output:**
```json
{"first_name": "John", "last_name": "Doe", "email": "john@example.com"}
```

### Example 2: Field Formatting

```json
{
  "transformer": {
    "type": "fieldmapper",
    "settings": {
      "mappings": [
        {"source": "email", "format": "lowercase"},
        {"source": "name", "format": "titlecase"},
        {"source": "age", "format": "int"}
      ]
    }
  }
}
```

**Input:**
```json
{"email": "JOHN@EXAMPLE.COM", "name": "john doe", "age": "30"}
```

**Output:**
```json
{"email": "john@example.com", "name": "John Doe", "age": 30}
```

### Example 3: Including All Fields with Exclusions

```json
{
  "transformer": {
    "type": "fieldmapper",
    "settings": {
      "mappings": [
        {"source": "email", "format": "lowercase"}
      ],
      "include_all": true,
      "exclude_fields": ["password", "ssn"]
    }
  }
}
```

**Input:**
```json
{
  "name": "John",
  "email": "JOHN@EXAMPLE.COM",
  "password": "secret123",
  "ssn": "123-45-6789"
}
```

**Output:**
```json
{"name": "John", "email": "john@example.com"}
```

### Example 4: Default Values

```json
{
  "transformer": {
    "type": "fieldmapper",
    "settings": {
      "mappings": [
        {"source": "status", "default": "pending"},
        {"source": "priority", "default": "normal"}
      ]
    }
  }
}
```

**Input:**
```json
{"name": "Task 1"}
```

**Output:**
```json
{"status": "pending", "priority": "normal"}
```

### Example 5: Extracting Usernames from Emails

```json
{
  "transformer": {
    "type": "fieldmapper",
    "settings": {
      "mappings": [
        {
          "source": "email",
          "destination": "username",
          "extract": "^([^@]+)@"
        },
        {
          "source": "email",
          "destination": "domain",
          "extract": "@(.+)$"
        }
      ]
    }
  }
}
```

**Input:**
```json
{"email": "john.doe@example.com"}
```

**Output:**
```json
{"username": "john.doe", "domain": "example.com"}
```

## Edge Cases Handling

### Null Values

Null values are treated as missing fields:

```json
{"name": null}  // Skipped unless default is provided
```

### Empty Strings

Empty strings are preserved (not treated as missing):

```json
{"name": ""}  // Remains as ""
```

### Zero Values

Numeric zero values are preserved:

```json
{"count": 0}  // Remains as 0
```

### Missing Required Fields

In strict mode, missing required fields cause an error:

```json
{
  "mappings": [
    {"source": "id", "required": true}
  ],
  "strict_mode": true
}
```

### Format Conversion Errors

In non-strict mode, fields with format errors are skipped:

```json
{"age": "not-a-number"}  // Skipped if format is "int" in non-strict mode
```

In strict mode, format errors cause the transformation to fail.

### Complex Nested Objects

The field mapper works on the top level of `event.Data`. For nested objects, you can:

1. Access the entire nested object as a value
2. Use preprocessing to flatten nested structures
3. Create a custom transformer for deep nesting

## Performance Considerations

- **Regex Extraction**: Compiled once during initialization, minimal overhead
- **Batch Processing**: Transformations are applied per event, suitable for streaming
- **Memory**: Field mapping creates a new data map, old data is replaced

## Troubleshooting

### Issue: Field not appearing in output

**Check:**
1. Is the field mapped in `mappings`?
2. If `include_all: false`, only mapped fields appear
3. Is the field in `exclude_fields`?
4. Did the format transformation fail? (Check strict_mode)

### Issue: Transformation fails with error

**Solution:**
1. Set `strict_mode: false` to skip problematic fields
2. Check regex patterns are valid
3. Verify format types match data types

### Issue: Extracted value is empty

**Check:**
1. Regex pattern matches the input format
2. Use capture groups `()` to extract specific parts
3. Test regex pattern with sample data

## Complete Configuration Example

See `examples/config-with-mapping.json` for a complete working example:

```bash
./data-pipe -config examples/config-with-mapping.json
```

## Combining with Other Features

Field mapping works seamlessly with:
- MongoDB change streams (renames MongoDB document fields)
- PostgreSQL sink (matches target table schema)
- Multiple transformers (chain transformers if needed)

## Best Practices

1. **Use lowercase for SQL columns** - Most databases prefer lowercase column names
2. **Trim strings** - Remove whitespace to avoid comparison issues
3. **Exclude sensitive fields** - Use `exclude_fields` for passwords, SSNs, etc.
4. **Set defaults carefully** - Ensure defaults match target schema types
5. **Test regex patterns** - Validate extraction patterns with sample data
6. **Use non-strict mode** - Start with strict_mode: false, enable only when needed
7. **Document mappings** - Keep a record of field transformations for your team
