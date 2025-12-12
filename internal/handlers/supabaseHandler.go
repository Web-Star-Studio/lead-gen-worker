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

// GetBusinessProfile retrieves a business profile by its ID
func (h *SupabaseHandler) GetBusinessProfile(id string) (*dto.BusinessProfile, error) {
	log.Printf("[SupabaseHandler] GetBusinessProfile: id=%s", id)

	data, _, err := h.client.From("business_profiles").Select("*", "exact", false).Eq("id", id).Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to get BusinessProfile: %v", err)
		return nil, fmt.Errorf("failed to get business profile: %w", err)
	}

	var profiles []dto.BusinessProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		log.Printf("[SupabaseHandler] Failed to parse BusinessProfile response: %v", err)
		return nil, fmt.Errorf("failed to parse business profile response: %w", err)
	}

	if len(profiles) == 0 {
		return nil, fmt.Errorf("business profile not found with id %s", id)
	}

	log.Printf("[SupabaseHandler] Found BusinessProfile: %s", profiles[0].CompanyName)
	return &profiles[0], nil
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

// InsertPreCallReport inserts or updates a pre-call report for a lead (UPSERT)
func (h *SupabaseHandler) InsertPreCallReport(leadID, content string) error {
	log.Printf("[SupabaseHandler] InsertPreCallReport (upsert): lead_id=%s", leadID)

	insertData := map[string]interface{}{
		"lead_id": leadID,
		"content": content,
	}

	// Use upsert=true with onConflict="lead_id" to update if exists
	_, _, err := h.client.From("pre_call_reports").Insert(insertData, true, "lead_id", "", "").Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to upsert pre-call report: %v", err)
		return fmt.Errorf("failed to upsert pre-call report: %w", err)
	}

	log.Printf("[SupabaseHandler] Pre-call report upserted successfully")
	return nil
}

// LeadHasEmail checks if a lead already has an email generated
func (h *SupabaseHandler) LeadHasEmail(leadID string) (bool, error) {
	data, _, err := h.client.From("emails").
		Select("id", "exact", false).
		Eq("lead_id", leadID).
		Limit(1, "").
		Execute()
	if err != nil {
		return false, fmt.Errorf("failed to check for existing email: %w", err)
	}

	var emails []map[string]interface{}
	if err := json.Unmarshal(data, &emails); err != nil {
		return false, fmt.Errorf("failed to parse email check response: %w", err)
	}

	return len(emails) > 0, nil
}

// LeadHasPreCallReport checks if a lead already has a pre-call report generated
func (h *SupabaseHandler) LeadHasPreCallReport(leadID string) (bool, error) {
	data, _, err := h.client.From("pre_call_reports").
		Select("id", "exact", false).
		Eq("lead_id", leadID).
		Limit(1, "").
		Execute()
	if err != nil {
		return false, fmt.Errorf("failed to check for existing pre-call report: %w", err)
	}

	var reports []map[string]interface{}
	if err := json.Unmarshal(data, &reports); err != nil {
		return false, fmt.Errorf("failed to parse pre-call check response: %w", err)
	}

	return len(reports) > 0, nil
}

// InsertColdEmail inserts a cold email for a lead into the emails table
func (h *SupabaseHandler) InsertColdEmail(email *dto.ColdEmailRecord) (string, error) {
	log.Printf("[SupabaseHandler] InsertColdEmail: lead_id=%s, subject=%s", email.LeadID, email.Subject)

	insertData := map[string]interface{}{
		"lead_id":  email.LeadID,
		"subject":  email.Subject,
		"body":     email.Body,
		"status":   "draft", // Default status
		"to_email": email.ToEmail,
	}

	if email.BusinessProfileID != nil && *email.BusinessProfileID != "" {
		insertData["business_profile_id"] = *email.BusinessProfileID
	}
	if email.FromName != "" {
		insertData["from_name"] = email.FromName
	}
	if email.FromEmail != "" {
		insertData["from_email"] = email.FromEmail
	}
	if email.ReplyTo != "" {
		insertData["reply_to"] = email.ReplyTo
	}

	data, _, err := h.client.From("emails").Insert(insertData, false, "", "", "").Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to insert cold email: %v", err)
		return "", fmt.Errorf("failed to insert cold email: %w", err)
	}

	// Parse response to get the generated ID
	var inserted []map[string]interface{}
	if err := json.Unmarshal(data, &inserted); err != nil {
		log.Printf("[SupabaseHandler] Failed to parse insert response: %v", err)
		return "", fmt.Errorf("failed to parse insert response: %w", err)
	}

	if len(inserted) == 0 {
		return "", fmt.Errorf("no cold email was inserted")
	}

	emailID, ok := inserted[0]["id"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get cold email ID from response")
	}

	log.Printf("[SupabaseHandler] Cold email inserted successfully: id=%s", emailID)
	return emailID, nil
}

// ============================================================================
// AUTOMATION METHODS
// ============================================================================

// GetLeadByID retrieves a lead by its ID
func (h *SupabaseHandler) GetLeadByID(id string) (*dto.Lead, error) {
	log.Printf("[SupabaseHandler] GetLeadByID: id=%s", id)

	data, _, err := h.client.From("leads").
		Select("*", "", false).
		Eq("id", id).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get lead: %w", err)
	}

	var lead dto.Lead
	if err := json.Unmarshal(data, &lead); err != nil {
		return nil, fmt.Errorf("failed to parse lead: %w", err)
	}

	return &lead, nil
}

// UpdateLeadEnrichment updates a lead with enriched data
func (h *SupabaseHandler) UpdateLeadEnrichment(leadID string, data *ExtractedData) error {
	log.Printf("[SupabaseHandler] UpdateLeadEnrichment: lead_id=%s", leadID)

	updateData := map[string]interface{}{}

	if data.Contact != "" {
		updateData["contact_name"] = data.Contact
	}
	if data.ContactRole != "" {
		updateData["contact_role"] = data.ContactRole
	}
	if len(data.Emails) > 0 {
		updateData["emails"] = data.Emails
	}
	if len(data.Phones) > 0 {
		updateData["phones"] = data.Phones
	}
	if data.Address != "" {
		updateData["address"] = data.Address
	}
	if len(data.SocialMedia) > 0 {
		updateData["social_media"] = data.SocialMedia
	}

	if len(updateData) == 0 {
		log.Printf("[SupabaseHandler] No enrichment data to update for lead %s", leadID)
		return nil
	}

	_, _, err := h.client.From("leads").
		Update(updateData, "", "").
		Eq("id", leadID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update lead enrichment: %w", err)
	}

	log.Printf("[SupabaseHandler] Lead enrichment updated: id=%s, fields=%d", leadID, len(updateData))
	return nil
}

// UpdateLeadStatus updates the status of a lead
func (h *SupabaseHandler) UpdateLeadStatus(leadID string, status string) error {
	log.Printf("[SupabaseHandler] UpdateLeadStatus: lead_id=%s, status=%s", leadID, status)

	updateData := map[string]interface{}{
		"status": status,
	}

	_, _, err := h.client.From("leads").
		Update(updateData, "", "").
		Eq("id", leadID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update lead status: %w", err)
	}

	return nil
}

// GetAutomationConfig retrieves a user's automation config
func (h *SupabaseHandler) GetAutomationConfig(userID string) (*dto.AutomationConfig, error) {
	log.Printf("[SupabaseHandler] GetAutomationConfig: user_id=%s", userID)

	data, _, err := h.client.From("automation_configs").
		Select("*", "", false).
		Eq("user_id", userID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get automation config: %w", err)
	}

	var config dto.AutomationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse automation config: %w", err)
	}

	return &config, nil
}

// GetAutomationTaskStatus gets the current status of an automation task
func (h *SupabaseHandler) GetAutomationTaskStatus(taskID string) (string, error) {
	data, _, err := h.client.From("automation_tasks").
		Select("status", "", false).
		Eq("id", taskID).
		Single().
		Execute()
	if err != nil {
		return "", fmt.Errorf("failed to get automation task status: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse task status: %w", err)
	}

	return result["status"], nil
}

// UpdateAutomationTaskStatus updates the status and progress of an automation task
func (h *SupabaseHandler) UpdateAutomationTaskStatus(taskID string, status string, total, succeeded, failed int, errorMsg *string) error {
	log.Printf("[SupabaseHandler] UpdateAutomationTaskStatus: task_id=%s, status=%s, total=%d, succeeded=%d, failed=%d",
		taskID, status, total, succeeded, failed)

	updateData := map[string]interface{}{
		"status":          status,
		"items_total":     total,
		"items_processed": succeeded + failed,
		"items_succeeded": succeeded,
		"items_failed":    failed,
	}

	if status == "processing" {
		updateData["started_at"] = time.Now().Format(time.RFC3339)
	}

	if status == "completed" || status == "failed" {
		updateData["completed_at"] = time.Now().Format(time.RFC3339)
	}

	if errorMsg != nil {
		updateData["error_message"] = *errorMsg
	}

	_, _, err := h.client.From("automation_tasks").
		Update(updateData, "", "").
		Eq("id", taskID).
		Execute()
	if err != nil {
		return fmt.Errorf("failed to update automation task: %w", err)
	}

	return nil
}

// GetPreCallReportForLead retrieves the pre-call report content for a lead
func (h *SupabaseHandler) GetPreCallReportForLead(leadID string) (string, error) {
	log.Printf("[SupabaseHandler] GetPreCallReportForLead: lead_id=%s", leadID)

	// Get the most recent pre-call report for this lead
	data, _, err := h.client.From("pre_call_reports").
		Select("content", "", false).
		Eq("lead_id", leadID).
		Single().
		Execute()
	if err != nil {
		return "", fmt.Errorf("failed to get pre-call report: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse pre-call report: %w", err)
	}

	return result["content"], nil
}

// InsertAutomationTask creates a new automation task in the database
func (h *SupabaseHandler) InsertAutomationTask(task *dto.AutomationTask) (string, error) {
	log.Printf("[SupabaseHandler] InsertAutomationTask: type=%s, user=%s", task.TaskType, task.UserID)

	insertData := map[string]interface{}{
		"user_id":     task.UserID,
		"task_type":   task.TaskType,
		"priority":    task.Priority,
		"status":      "pending",
		"items_total": task.ItemsTotal,
		"max_retries": task.MaxRetries,
	}

	if task.LeadID != nil {
		insertData["lead_id"] = *task.LeadID
	}
	if len(task.LeadIDs) > 0 {
		insertData["lead_ids"] = task.LeadIDs
	}
	if task.BusinessProfileID != nil {
		insertData["business_profile_id"] = *task.BusinessProfileID
	}

	data, _, err := h.client.From("automation_tasks").Insert(insertData, false, "", "", "").Execute()
	if err != nil {
		return "", fmt.Errorf("failed to insert automation task: %w", err)
	}

	var inserted []map[string]interface{}
	if err := json.Unmarshal(data, &inserted); err != nil {
		return "", fmt.Errorf("failed to parse insert response: %w", err)
	}

	if len(inserted) == 0 {
		return "", fmt.Errorf("no task was inserted")
	}

	taskID, ok := inserted[0]["id"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get task ID from response")
	}

	log.Printf("[SupabaseHandler] Automation task inserted: id=%s", taskID)
	return taskID, nil
}

// ============================================================================
// USAGE METRICS AND REPORTS METHODS
// ============================================================================

// InsertUsageMetric inserts a usage metric record
func (h *SupabaseHandler) InsertUsageMetric(metric *dto.UsageMetricInput) error {
	log.Printf("[SupabaseHandler] InsertUsageMetric: user=%s, type=%s, tokens=%d",
		metric.UserID, metric.OperationType, metric.TotalTokens)

	// Skip tracking for system operations (user_id is required in DB)
	if metric.UserID == "" || metric.UserID == "system" || !isValidUUID(metric.UserID) {
		log.Printf("[SupabaseHandler] Skipping usage metric for non-user operation (user=%s)", metric.UserID)
		return nil
	}

	insertData := map[string]interface{}{
		"user_id":            metric.UserID,
		"operation_type":     metric.OperationType,
		"model":              metric.Model,
		"input_tokens":       metric.InputTokens,
		"output_tokens":      metric.OutputTokens,
		"total_tokens":       metric.TotalTokens,
		"estimated_cost_usd": metric.EstimatedCostUS,
		"duration_ms":        metric.DurationMs,
		"success":            metric.Success,
	}

	if metric.JobID != nil {
		insertData["job_id"] = *metric.JobID
	}
	if metric.LeadID != nil {
		insertData["lead_id"] = *metric.LeadID
	}
	if metric.ErrorMessage != nil {
		insertData["error_message"] = *metric.ErrorMessage
	}

	_, _, err := h.client.From("usage_metrics").Insert(insertData, false, "", "", "").Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to insert usage metric: %v", err)
		return fmt.Errorf("failed to insert usage metric: %w", err)
	}

	return nil
}

// GetUsageSummary retrieves aggregated usage summary for a user
func (h *SupabaseHandler) GetUsageSummary(userID string, startDate, endDate *time.Time) (*dto.UsageSummary, error) {
	log.Printf("[SupabaseHandler] GetUsageSummary: user=%s", userID)

	query := h.client.From("usage_metrics").
		Select("*", "", false).
		Eq("user_id", userID)

	if startDate != nil {
		query = query.Gte("created_at", startDate.Format(time.RFC3339))
	}
	if endDate != nil {
		query = query.Lte("created_at", endDate.Format(time.RFC3339))
	}

	data, _, err := query.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get usage metrics: %w", err)
	}

	var metrics []dto.UsageMetric
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse usage metrics: %w", err)
	}

	summary := &dto.UsageSummary{}
	var totalDuration int64

	for _, m := range metrics {
		summary.TotalCalls++
		if m.Success {
			summary.SuccessfulCalls++
		} else {
			summary.FailedCalls++
		}
		summary.TotalInputTokens += m.InputTokens
		summary.TotalOutputTokens += m.OutputTokens
		summary.TotalTokens += m.TotalTokens
		summary.TotalCostUSD += m.EstimatedCostUS
		totalDuration += m.DurationMs
	}

	if summary.TotalCalls > 0 {
		summary.SuccessRate = float64(summary.SuccessfulCalls) / float64(summary.TotalCalls) * 100
		summary.AvgCostPerCall = summary.TotalCostUSD / float64(summary.TotalCalls)
		summary.AvgTokensPerCall = float64(summary.TotalTokens) / float64(summary.TotalCalls)
		summary.AvgDurationMs = float64(totalDuration) / float64(summary.TotalCalls)
	}

	return summary, nil
}

// GetUsageByOperation retrieves usage statistics grouped by operation type
func (h *SupabaseHandler) GetUsageByOperation(userID string, startDate, endDate *time.Time) ([]dto.OperationStats, error) {
	log.Printf("[SupabaseHandler] GetUsageByOperation: user=%s", userID)

	query := h.client.From("usage_metrics").
		Select("*", "", false).
		Eq("user_id", userID)

	if startDate != nil {
		query = query.Gte("created_at", startDate.Format(time.RFC3339))
	}
	if endDate != nil {
		query = query.Lte("created_at", endDate.Format(time.RFC3339))
	}

	data, _, err := query.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get usage metrics: %w", err)
	}

	var metrics []dto.UsageMetric
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse usage metrics: %w", err)
	}

	// Group by operation type
	statsMap := make(map[dto.OperationType]*dto.OperationStats)
	durationMap := make(map[dto.OperationType]int64)

	for _, m := range metrics {
		stat, ok := statsMap[m.OperationType]
		if !ok {
			stat = &dto.OperationStats{OperationType: m.OperationType}
			statsMap[m.OperationType] = stat
		}

		stat.TotalCalls++
		if m.Success {
			stat.SuccessfulCalls++
		} else {
			stat.FailedCalls++
		}
		stat.TotalInputTokens += m.InputTokens
		stat.TotalOutputTokens += m.OutputTokens
		stat.TotalTokens += m.TotalTokens
		stat.TotalCostUSD += m.EstimatedCostUS
		durationMap[m.OperationType] += m.DurationMs
	}

	var result []dto.OperationStats
	for opType, stat := range statsMap {
		if stat.TotalCalls > 0 {
			stat.SuccessRate = float64(stat.SuccessfulCalls) / float64(stat.TotalCalls) * 100
			stat.AvgDurationMs = float64(durationMap[opType]) / float64(stat.TotalCalls)
		}
		result = append(result, *stat)
	}

	return result, nil
}

// GetUsageByModel retrieves usage statistics grouped by model
func (h *SupabaseHandler) GetUsageByModel(userID string, startDate, endDate *time.Time) ([]dto.ModelUsage, error) {
	log.Printf("[SupabaseHandler] GetUsageByModel: user=%s", userID)

	query := h.client.From("usage_metrics").
		Select("*", "", false).
		Eq("user_id", userID)

	if startDate != nil {
		query = query.Gte("created_at", startDate.Format(time.RFC3339))
	}
	if endDate != nil {
		query = query.Lte("created_at", endDate.Format(time.RFC3339))
	}

	data, _, err := query.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get usage metrics: %w", err)
	}

	var metrics []dto.UsageMetric
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse usage metrics: %w", err)
	}

	// Group by model
	statsMap := make(map[string]*dto.ModelUsage)

	for _, m := range metrics {
		stat, ok := statsMap[m.Model]
		if !ok {
			stat = &dto.ModelUsage{Model: m.Model}
			statsMap[m.Model] = stat
		}

		stat.TotalCalls++
		stat.TotalTokens += m.TotalTokens
		stat.TotalCostUSD += m.EstimatedCostUS
	}

	var result []dto.ModelUsage
	for _, stat := range statsMap {
		if stat.TotalCalls > 0 {
			stat.AvgTokensPerCall = float64(stat.TotalTokens) / float64(stat.TotalCalls)
		}
		result = append(result, *stat)
	}

	return result, nil
}

// GetDailyUsage retrieves usage statistics aggregated by day
func (h *SupabaseHandler) GetDailyUsage(userID string, startDate, endDate *time.Time) ([]dto.DailyUsage, error) {
	log.Printf("[SupabaseHandler] GetDailyUsage: user=%s", userID)

	query := h.client.From("usage_metrics").
		Select("*", "", false).
		Eq("user_id", userID)

	if startDate != nil {
		query = query.Gte("created_at", startDate.Format(time.RFC3339))
	}
	if endDate != nil {
		query = query.Lte("created_at", endDate.Format(time.RFC3339))
	}

	data, _, err := query.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get usage metrics: %w", err)
	}

	var metrics []dto.UsageMetric
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to parse usage metrics: %w", err)
	}

	// Group by date
	statsMap := make(map[string]*dto.DailyUsage)

	for _, m := range metrics {
		dateStr := m.CreatedAt.Format("2006-01-02")
		stat, ok := statsMap[dateStr]
		if !ok {
			stat = &dto.DailyUsage{Date: dateStr}
			statsMap[dateStr] = stat
		}

		stat.TotalCalls++
		if m.Success {
			stat.SuccessfulCalls++
		} else {
			stat.FailedCalls++
		}
		stat.TotalTokens += m.TotalTokens
		stat.TotalCostUSD += m.EstimatedCostUS
	}

	// Convert to sorted slice
	var result []dto.DailyUsage
	for _, stat := range statsMap {
		result = append(result, *stat)
	}

	// Sort by date
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Date > result[j].Date {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// GetLeadGenerationStats retrieves lead generation specific statistics
func (h *SupabaseHandler) GetLeadGenerationStats(userID string, startDate, endDate *time.Time) (*dto.LeadGenerationStats, error) {
	log.Printf("[SupabaseHandler] GetLeadGenerationStats: user=%s", userID)

	stats := &dto.LeadGenerationStats{}

	// Get jobs count
	jobQuery := h.client.From("jobs").
		Select("id", "exact", false).
		Eq("user_id", userID)

	if startDate != nil {
		jobQuery = jobQuery.Gte("created_at", startDate.Format(time.RFC3339))
	}
	if endDate != nil {
		jobQuery = jobQuery.Lte("created_at", endDate.Format(time.RFC3339))
	}

	_, jobCount, err := jobQuery.Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to get jobs count: %v", err)
	} else {
		stats.TotalJobsProcessed = int(jobCount)
	}

	// Get leads count
	leadQuery := h.client.From("leads").
		Select("id", "exact", false).
		Eq("user_id", userID)

	if startDate != nil {
		leadQuery = leadQuery.Gte("created_at", startDate.Format(time.RFC3339))
	}
	if endDate != nil {
		leadQuery = leadQuery.Lte("created_at", endDate.Format(time.RFC3339))
	}

	_, leadCount, err := leadQuery.Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to get leads count: %v", err)
	} else {
		stats.TotalLeadsGenerated = int(leadCount)
	}

	// Get emails count
	emailQuery := h.client.From("emails").
		Select("id,lead_id", "exact", false)

	// Join with leads to filter by user
	emailData, emailCount, err := emailQuery.Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to get emails count: %v", err)
	} else {
		stats.TotalEmailsGenerated = int(emailCount)
		_ = emailData // Unused for now
	}

	// Get pre-call reports count from usage metrics
	reportQuery := h.client.From("usage_metrics").
		Select("id", "exact", false).
		Eq("user_id", userID).
		Eq("operation_type", string(dto.OperationPreCallReport)).
		Eq("success", "true")

	if startDate != nil {
		reportQuery = reportQuery.Gte("created_at", startDate.Format(time.RFC3339))
	}
	if endDate != nil {
		reportQuery = reportQuery.Lte("created_at", endDate.Format(time.RFC3339))
	}

	_, reportCount, err := reportQuery.Execute()
	if err != nil {
		log.Printf("[SupabaseHandler] Failed to get reports count: %v", err)
	} else {
		stats.TotalReportsGenerated = int(reportCount)
	}

	// Calculate averages
	if stats.TotalJobsProcessed > 0 {
		stats.AvgLeadsPerJob = float64(stats.TotalLeadsGenerated) / float64(stats.TotalJobsProcessed)
	}

	// Get total cost for cost per lead calculation
	summary, err := h.GetUsageSummary(userID, startDate, endDate)
	if err == nil && stats.TotalLeadsGenerated > 0 {
		stats.AvgCostPerLead = summary.TotalCostUSD / float64(stats.TotalLeadsGenerated)
	}

	return stats, nil
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
