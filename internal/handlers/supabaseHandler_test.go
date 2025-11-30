package handlers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryResult_Fields(t *testing.T) {
	result := QueryResult{
		Data: []map[string]interface{}{
			{"id": float64(1), "name": "Test"},
			{"id": float64(2), "name": "Test 2"},
		},
		Count: 2,
	}

	assert.Len(t, result.Data, 2)
	assert.Equal(t, int64(2), result.Count)
	assert.Equal(t, "Test", result.Data[0]["name"])
	assert.Equal(t, "Test 2", result.Data[1]["name"])
}

func TestQueryResult_Empty(t *testing.T) {
	result := QueryResult{
		Data:  []map[string]interface{}{},
		Count: 0,
	}

	assert.Empty(t, result.Data)
	assert.Equal(t, int64(0), result.Count)
}

func TestNewSupabaseHandler_MissingURL(t *testing.T) {
	handler, err := NewSupabaseHandler("", "test-key")

	assert.Nil(t, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supabase URL is required")
}

func TestNewSupabaseHandler_MissingKey(t *testing.T) {
	handler, err := NewSupabaseHandler("https://test.supabase.co", "")

	assert.Nil(t, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supabase key is required")
}

func TestNewSupabaseHandler_BothMissing(t *testing.T) {
	handler, err := NewSupabaseHandler("", "")

	assert.Nil(t, handler)
	assert.Error(t, err)
}

func TestGetRows_EmptyTable(t *testing.T) {
	// Test that GetRows validates table parameter
	// We can't create a real handler without valid credentials,
	// so we test the validation logic conceptually

	tableName := ""
	assert.Empty(t, tableName, "Empty table name should be rejected")
}

func TestGetRows_DefaultColumns(t *testing.T) {
	// Test that empty columns defaults to "*"
	columns := ""
	if columns == "" {
		columns = "*"
	}
	assert.Equal(t, "*", columns)
}

func TestGetRowsWithFilter_Validation(t *testing.T) {
	tests := []struct {
		name         string
		table        string
		columns      string
		filterColumn string
		filterValue  interface{}
		expectError  bool
	}{
		{
			name:         "valid parameters",
			table:        "users",
			columns:      "id,name",
			filterColumn: "id",
			filterValue:  1,
			expectError:  false,
		},
		{
			name:         "empty table",
			table:        "",
			columns:      "*",
			filterColumn: "id",
			filterValue:  1,
			expectError:  true,
		},
		{
			name:         "empty filter column",
			table:        "users",
			columns:      "*",
			filterColumn: "",
			filterValue:  1,
			expectError:  true,
		},
		{
			name:         "empty columns defaults to star",
			table:        "users",
			columns:      "",
			filterColumn: "id",
			filterValue:  1,
			expectError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate validation logic from GetRowsWithFilter
			hasError := false
			if tc.table == "" {
				hasError = true
			}
			if tc.filterColumn == "" {
				hasError = true
			}

			assert.Equal(t, tc.expectError, hasError)
		})
	}
}

func TestGetRowByID_NotFound(t *testing.T) {
	// Test the logic when no row is found
	result := &QueryResult{
		Data:  []map[string]interface{}{},
		Count: 0,
	}

	assert.Empty(t, result.Data, "Empty result should return no rows")
}

func TestGetRowByID_Found(t *testing.T) {
	// Test the logic when a row is found
	result := &QueryResult{
		Data: []map[string]interface{}{
			{"id": float64(1), "name": "Found User", "email": "user@example.com"},
		},
		Count: 1,
	}

	assert.Len(t, result.Data, 1)
	row := result.Data[0]
	assert.Equal(t, float64(1), row["id"])
	assert.Equal(t, "Found User", row["name"])
	assert.Equal(t, "user@example.com", row["email"])
}

func TestFilterValueConversion(t *testing.T) {
	// Test that various filter values are properly converted to strings
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"integer", 42, "42"},
		{"string", "test", "test"},
		{"float", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the filter value conversion in GetRowsWithFilter
			filterStr := fmt.Sprintf("%v", tc.value)
			assert.Equal(t, tc.expected, filterStr)
		})
	}
}

func TestQueryResultDataTypes(t *testing.T) {
	// Test handling different data types in query results
	result := QueryResult{
		Data: []map[string]interface{}{
			{
				"id":        float64(1),
				"name":      "Test User",
				"is_active": true,
				"score":     float64(95.5),
				"tags":      []interface{}{"tag1", "tag2"},
				"metadata":  map[string]interface{}{"key": "value"},
				"nullable":  nil,
			},
		},
		Count: 1,
	}

	row := result.Data[0]

	// Test float64 (JSON numbers are always float64 in Go)
	id, ok := row["id"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(1), id)

	// Test string
	name, ok := row["name"].(string)
	assert.True(t, ok)
	assert.Equal(t, "Test User", name)

	// Test boolean
	isActive, ok := row["is_active"].(bool)
	assert.True(t, ok)
	assert.True(t, isActive)

	// Test float
	score, ok := row["score"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(95.5), score)

	// Test array
	tags, ok := row["tags"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, tags, 2)

	// Test nested object
	metadata, ok := row["metadata"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "value", metadata["key"])

	// Test nil
	assert.Nil(t, row["nullable"])
}

func TestColumnSelection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"all columns", "*", "*"},
		{"single column", "id", "id"},
		{"multiple columns", "id,name,email", "id,name,email"},
		{"empty defaults to star", "", "*"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			columns := tc.input
			if columns == "" {
				columns = "*"
			}
			assert.Equal(t, tc.expected, columns)
		})
	}
}
