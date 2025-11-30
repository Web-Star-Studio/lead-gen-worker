package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"webstar/noturno-leadgen-worker/internal/dto"

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

// GetICP retrieves an ICP by its ID
func (h *SupabaseHandler) GetICP(id string) (*dto.ICP, error) {
	log.Printf("[SupabaseHandler] GetICP: id=%s", id)

	data, _, err := h.client.From("icps").Select("*", "exact", false).Eq("id", id).Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to get ICP: %v", err)
		return nil, fmt.Errorf("failed to get ICP: %w", err)
	}

	var icps []dto.ICP
	if err := json.Unmarshal(data, &icps); err != nil {
		log.Printf("[SupabaseHandler] Failed to parse ICP response: %v", err)
		return nil, fmt.Errorf("failed to parse ICP response: %w", err)
	}

	if len(icps) == 0 {
		return nil, fmt.Errorf("ICP not found with id %s", id)
	}

	log.Printf("[SupabaseHandler] Found ICP: %s", icps[0].Name)
	return &icps[0], nil
}

// UpdateJobStatus updates the status and related fields of a job
func (h *SupabaseHandler) UpdateJobStatus(jobID string, status string, leadsGenerated *int, errorMessage *string) error {
	log.Printf("[SupabaseHandler] UpdateJobStatus: jobID=%s, status=%s", jobID, status)

	update := map[string]interface{}{
		"status": status,
	}

	now := time.Now().UTC()

	switch status {
	case "processing":
		update["started_at"] = now.Format(time.RFC3339)
	case "completed":
		update["completed_at"] = now.Format(time.RFC3339)
		if leadsGenerated != nil {
			update["leads_generated"] = *leadsGenerated
		}
	case "failed":
		update["completed_at"] = now.Format(time.RFC3339)
		if errorMessage != nil {
			update["error_message"] = *errorMessage
		}
	}

	_, _, err := h.client.From("jobs").Update(update, "", "").Eq("id", jobID).Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to update job status: %v", err)
		return fmt.Errorf("failed to update job status: %w", err)
	}

	log.Printf("[SupabaseHandler] Job status updated successfully")
	return nil
}

// InsertLead inserts a new lead and returns the generated ID
func (h *SupabaseHandler) InsertLead(lead *dto.Lead) (string, error) {
	log.Printf("[SupabaseHandler] InsertLead: company=%s, job_id=%s", lead.CompanyName, lead.JobID)

	insertData := map[string]interface{}{
		"job_id":       lead.JobID,
		"user_id":      lead.UserID,
		"company_name": lead.CompanyName,
		"contact_name": lead.ContactName,
		"source":       lead.Source,
	}

	if len(lead.Emails) > 0 {
		insertData["emails"] = lead.Emails
	}
	if len(lead.Phones) > 0 {
		insertData["phones"] = lead.Phones
	}
	if lead.Website != nil {
		insertData["website"] = *lead.Website
	}
	if lead.ContactRole != "" {
		insertData["contact_role"] = lead.ContactRole
	}
	if lead.Address != "" {
		insertData["address"] = lead.Address
	}
	if len(lead.SocialMedia) > 0 {
		insertData["social_media"] = lead.SocialMedia
	}

	data, _, err := h.client.From("leads").Insert(insertData, false, "", "", "").Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to insert lead: %v", err)
		return "", fmt.Errorf("failed to insert lead: %w", err)
	}

	// Parse response to get the generated ID
	var inserted []map[string]interface{}
	if err := json.Unmarshal(data, &inserted); err != nil {
		log.Printf("[SupabaseHandler] Failed to parse insert response: %v", err)
		return "", fmt.Errorf("failed to parse insert response: %w", err)
	}

	if len(inserted) == 0 {
		return "", fmt.Errorf("no lead was inserted")
	}

	leadID, ok := inserted[0]["id"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get lead ID from response")
	}

	log.Printf("[SupabaseHandler] Lead inserted successfully: id=%s", leadID)
	return leadID, nil
}

// InsertPreCallReport inserts a pre-call report for a lead
func (h *SupabaseHandler) InsertPreCallReport(leadID, content string) error {
	log.Printf("[SupabaseHandler] InsertPreCallReport: lead_id=%s", leadID)

	insertData := map[string]interface{}{
		"lead_id": leadID,
		"content": content,
	}

	_, _, err := h.client.From("pre_call_reports").Insert(insertData, false, "", "", "").Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to insert pre-call report: %v", err)
		return fmt.Errorf("failed to insert pre-call report: %w", err)
	}

	log.Printf("[SupabaseHandler] Pre-call report inserted successfully")
	return nil
}
