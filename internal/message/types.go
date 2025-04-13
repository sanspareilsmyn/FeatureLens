package message

import (
	"fmt"
	"time"
)

// DynamicMessage represents a message with arbitrary key-value pairs,
// typically parsed from JSON.
type DynamicMessage map[string]interface{}

// GetFloat64 retrieves a float64 value for a given key.
// Handles missing keys, null values, and potential integer-to-float conversion.
// Returns the value pointer and true if successful, otherwise (nil, false).
func (dm DynamicMessage) GetFloat64(key string) (*float64, bool) {
	val, exists := dm[key]
	if !exists || val == nil {
		return nil, false
	}

	// Try direct assertion first (most common case with JSON numbers)
	if fVal, ok := val.(float64); ok {
		return &fVal, true
	}

	// Handle potential integer types if the map wasn't strictly from JSON unmarshal
	switch v := val.(type) {
	case int:
		fVal := float64(v)
		return &fVal, true
	case int64:
		fVal := float64(v)
		return &fVal, true
	case float32:
		fVal := float64(v)
		return &fVal, true
	}

	// Value exists but is not a convertible numeric type
	return nil, false
}

// HasNonNull checks if a key exists and its value is not explicitly null.
func (dm DynamicMessage) HasNonNull(key string) bool {
	val, exists := dm[key]
	return exists && val != nil
}

// GetTime attempts to retrieve a time.Time value for a given key.
// Assumes the timestamp is stored as a string parsable by common formats.
// Returns the time pointer and true if successful, otherwise (nil, false).
func (dm DynamicMessage) GetTime(key string) (*time.Time, bool) {
	val, exists := dm[key]
	if !exists || val == nil {
		return nil, false
	}

	timeStr, ok := val.(string)
	if !ok {
		// Value exists but is not a string
		return nil, false
	}

	// Define common timestamp formats to try
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00", // RFC3339 without nano part
		"2006-01-02 15:04:05",       // Common space-separated format
	}

	// Attempt parsing with each defined format
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return &t, true
		}
	}

	// None of the defined formats matched
	return nil, false
}

// GetFieldSnippet returns a string snippet of a field's value, useful for logging.
// It handles missing keys and truncates long values.
func (dm DynamicMessage) GetFieldSnippet(fieldName string, maxLength int) string {
	value, exists := dm[fieldName]
	if !exists {
		return "<missing>"
	}

	strValue := fmt.Sprintf("%v", value)

	// Ensure maxLength is sensible before slicing
	if maxLength <= 0 {
		return "..."
	}

	// Truncate if the string representation exceeds the max length
	if len(strValue) > maxLength {
		return strValue[:maxLength] + "..."
	}

	return strValue
}
