package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/supabase-community/supabase-go"
)

// SupabaseHandler handles database operations using Supabase
type SupabaseHandler struct {
	client *supabase.Client
}

// QueryResult represents the result of a database query
type QueryResult struct {
	// Data contains the rows returned from the query
	Data []map[string]interface{}
	// Count is the total count of rows (if requested)
	Count int64
}

// NewSupabaseHandler creates a new SupabaseHandler instance
// url is the Supabase project URL (e.g., "https://xxx.supabase.co")
// key is the Supabase anon or service role key
func NewSupabaseHandler(url, key string) (*SupabaseHandler, error) {
	if url == "" {
		return nil, fmt.Errorf("supabase URL is required")
	}
	if key == "" {
		return nil, fmt.Errorf("supabase key is required")
	}

	log.Printf("[SupabaseHandler] Initializing with URL: %s", url)

	client, err := supabase.NewClient(url, key, &supabase.ClientOptions{})
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to create client: %v", err)
		return nil, fmt.Errorf("failed to create supabase client: %w", err)
	}

	log.Printf("[SupabaseHandler] Successfully created Supabase client")

	return &SupabaseHandler{
		client: client,
	}, nil
}

// GetRows retrieves rows from a table with optional column selection
// table is the name of the table to query
// columns is a comma-separated list of columns to select (use "*" for all)
func (h *SupabaseHandler) GetRows(table string, columns string) (*QueryResult, error) {
	if table == "" {
		return nil, fmt.Errorf("table name is required")
	}
	if columns == "" {
		columns = "*"
	}

	log.Printf("[SupabaseHandler] GetRows: table=%s, columns=%s", table, columns)

	data, count, err := h.client.From(table).Select(columns, "exact", false).Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Query failed: %v", err)
		return nil, fmt.Errorf("failed to query table %s: %w", table, err)
	}

	// Parse the JSON response into a slice of maps
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		log.Printf("[SupabaseHandler] Failed to parse response: %v", err)
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	log.Printf("[SupabaseHandler] Query successful: %d rows returned", len(rows))

	return &QueryResult{
		Data:  rows,
		Count: count,
	}, nil
}

// GetRowsWithFilter retrieves rows from a table with a filter condition
// table is the name of the table to query
// columns is a comma-separated list of columns to select (use "*" for all)
// filterColumn is the column to filter on
// filterValue is the value to match
func (h *SupabaseHandler) GetRowsWithFilter(table, columns, filterColumn string, filterValue interface{}) (*QueryResult, error) {
	if table == "" {
		return nil, fmt.Errorf("table name is required")
	}
	if columns == "" {
		columns = "*"
	}
	if filterColumn == "" {
		return nil, fmt.Errorf("filter column is required")
	}

	log.Printf("[SupabaseHandler] GetRowsWithFilter: table=%s, columns=%s, filter=%s=%v", table, columns, filterColumn, filterValue)

	// Convert filter value to string for the eq operation
	filterStr := fmt.Sprintf("%v", filterValue)

	data, count, err := h.client.From(table).Select(columns, "exact", false).Eq(filterColumn, filterStr).Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Query failed: %v", err)
		return nil, fmt.Errorf("failed to query table %s with filter: %w", table, err)
	}

	// Parse the JSON response into a slice of maps
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		log.Printf("[SupabaseHandler] Failed to parse response: %v", err)
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	log.Printf("[SupabaseHandler] Query successful: %d rows returned", len(rows))

	return &QueryResult{
		Data:  rows,
		Count: count,
	}, nil
}

// GetRowByID retrieves a single row by its ID
// table is the name of the table to query
// columns is a comma-separated list of columns to select (use "*" for all)
// id is the value of the id column to match
func (h *SupabaseHandler) GetRowByID(table, columns string, id interface{}) (map[string]interface{}, error) {
	result, err := h.GetRowsWithFilter(table, columns, "id", id)
	if err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no row found with id %v in table %s", id, table)
	}

	return result.Data[0], nil
}

// GetClient returns the underlying Supabase client for advanced operations
func (h *SupabaseHandler) GetClient() *supabase.Client {
	return h.client
}
