package dto

import "time"

// OperationType represents the type of AI operation performed
type OperationType string

const (
	OperationDataExtraction  OperationType = "data_extraction"
	OperationPreCallReport   OperationType = "pre_call_report"
	OperationColdEmail       OperationType = "cold_email"
	OperationWebsiteScraping OperationType = "website_scraping"
)

// UsageMetric represents a single AI usage record
// @Description Record of a single AI operation for usage tracking
type UsageMetric struct {
	ID              string        `json:"id"`
	UserID          string        `json:"user_id"`
	JobID           *string       `json:"job_id,omitempty"`
	LeadID          *string       `json:"lead_id,omitempty"`
	OperationType   OperationType `json:"operation_type"`
	Model           string        `json:"model"`
	InputTokens     int           `json:"input_tokens"`
	OutputTokens    int           `json:"output_tokens"`
	TotalTokens     int           `json:"total_tokens"`
	EstimatedCostUS float64       `json:"estimated_cost_usd"`
	DurationMs      int64         `json:"duration_ms"`
	Success         bool          `json:"success"`
	ErrorMessage    *string       `json:"error_message,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
}

// UsageMetricInput is the input for creating a new usage metric
type UsageMetricInput struct {
	UserID          string        `json:"user_id"`
	JobID           *string       `json:"job_id,omitempty"`
	LeadID          *string       `json:"lead_id,omitempty"`
	OperationType   OperationType `json:"operation_type"`
	Model           string        `json:"model"`
	InputTokens     int           `json:"input_tokens"`
	OutputTokens    int           `json:"output_tokens"`
	TotalTokens     int           `json:"total_tokens"`
	EstimatedCostUS float64       `json:"estimated_cost_usd"`
	DurationMs      int64         `json:"duration_ms"`
	Success         bool          `json:"success"`
	ErrorMessage    *string       `json:"error_message,omitempty"`
}

// ReportsRequest contains the filters for the reports endpoint
// @Description Filters for fetching usage reports
type ReportsRequest struct {
	// UserID filters by user (required for non-admin)
	UserID string `form:"user_id" json:"user_id"`
	// StartDate filters metrics from this date (inclusive)
	StartDate *time.Time `form:"start_date" json:"start_date,omitempty"`
	// EndDate filters metrics until this date (inclusive)
	EndDate *time.Time `form:"end_date" json:"end_date,omitempty"`
	// OperationType filters by operation type
	OperationType *OperationType `form:"operation_type" json:"operation_type,omitempty"`
	// GroupBy specifies aggregation: "day", "week", "month", or "operation"
	GroupBy string `form:"group_by" json:"group_by,omitempty"`
}

// OperationStats contains statistics for a specific operation type
// @Description Statistics for a specific AI operation type
type OperationStats struct {
	OperationType     OperationType `json:"operation_type"`
	TotalCalls        int           `json:"total_calls"`
	SuccessfulCalls   int           `json:"successful_calls"`
	FailedCalls       int           `json:"failed_calls"`
	SuccessRate       float64       `json:"success_rate"`
	TotalInputTokens  int           `json:"total_input_tokens"`
	TotalOutputTokens int           `json:"total_output_tokens"`
	TotalTokens       int           `json:"total_tokens"`
	TotalCostUSD      float64       `json:"total_cost_usd"`
	AvgDurationMs     float64       `json:"avg_duration_ms"`
}

// DailyUsage contains usage statistics for a specific day
// @Description Usage statistics aggregated by day
type DailyUsage struct {
	Date            string  `json:"date"`
	TotalCalls      int     `json:"total_calls"`
	SuccessfulCalls int     `json:"successful_calls"`
	FailedCalls     int     `json:"failed_calls"`
	TotalTokens     int     `json:"total_tokens"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
}

// ModelUsage contains usage statistics for a specific model
// @Description Usage statistics by AI model
type ModelUsage struct {
	Model            string  `json:"model"`
	TotalCalls       int     `json:"total_calls"`
	TotalTokens      int     `json:"total_tokens"`
	TotalCostUSD     float64 `json:"total_cost_usd"`
	AvgTokensPerCall float64 `json:"avg_tokens_per_call"`
}

// UsageSummary contains overall usage summary
// @Description Overall usage summary with key metrics
type UsageSummary struct {
	TotalCalls        int     `json:"total_calls"`
	SuccessfulCalls   int     `json:"successful_calls"`
	FailedCalls       int     `json:"failed_calls"`
	SuccessRate       float64 `json:"success_rate"`
	TotalInputTokens  int     `json:"total_input_tokens"`
	TotalOutputTokens int     `json:"total_output_tokens"`
	TotalTokens       int     `json:"total_tokens"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	AvgCostPerCall    float64 `json:"avg_cost_per_call"`
	AvgTokensPerCall  float64 `json:"avg_tokens_per_call"`
	AvgDurationMs     float64 `json:"avg_duration_ms"`
}

// LeadGenerationStats contains lead generation specific metrics
// @Description Statistics specific to lead generation pipeline
type LeadGenerationStats struct {
	TotalJobsProcessed    int     `json:"total_jobs_processed"`
	TotalLeadsGenerated   int     `json:"total_leads_generated"`
	TotalEmailsGenerated  int     `json:"total_emails_generated"`
	TotalReportsGenerated int     `json:"total_reports_generated"`
	AvgLeadsPerJob        float64 `json:"avg_leads_per_job"`
	AvgCostPerLead        float64 `json:"avg_cost_per_lead"`
}

// ReportsResponse contains the complete reports data for the dashboard
// @Description Complete reports response for dashboard visualization
type ReportsResponse struct {
	// Summary contains overall usage metrics
	Summary UsageSummary `json:"summary"`
	// ByOperation contains stats grouped by operation type
	ByOperation []OperationStats `json:"by_operation"`
	// ByModel contains stats grouped by AI model
	ByModel []ModelUsage `json:"by_model"`
	// DailyUsage contains usage over time
	DailyUsage []DailyUsage `json:"daily_usage"`
	// LeadGeneration contains lead-specific metrics
	LeadGeneration LeadGenerationStats `json:"lead_generation"`
	// Period indicates the date range of the report
	Period ReportPeriod `json:"period"`
}

// ReportPeriod indicates the time range of the report
// @Description Time range covered by the report
type ReportPeriod struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	DaysCount int    `json:"days_count"`
}

// TokenPricing contains pricing information for token estimation
type TokenPricing struct {
	Model              string
	InputPricePerMTok  float64 // Price per million input tokens
	OutputPricePerMTok float64 // Price per million output tokens
}

// DefaultTokenPricing returns pricing for supported models (Gemini + OpenRouter)
func DefaultTokenPricing() map[string]TokenPricing {
	return map[string]TokenPricing{
		// Google Gemini models (direct API)
		"gemini-2.5-flash": {
			Model:              "gemini-2.5-flash",
			InputPricePerMTok:  0.075,
			OutputPricePerMTok: 0.30,
		},
		"gemini-2.5-pro": {
			Model:              "gemini-2.5-pro",
			InputPricePerMTok:  1.25,
			OutputPricePerMTok: 10.00,
		},
		"gemini-2.5-pro-preview-06-05": {
			Model:              "gemini-2.5-pro-preview-06-05",
			InputPricePerMTok:  1.25,
			OutputPricePerMTok: 10.00,
		},
		// OpenRouter models
		"openai/gpt-5.2-chat": {
			Model:              "openai/gpt-5.2-chat",
			InputPricePerMTok:  1.75,
			OutputPricePerMTok: 14.00,
		},
		"google/gemini-2.5-flash": {
			Model:              "google/gemini-2.5-flash",
			InputPricePerMTok:  0.30,
			OutputPricePerMTok: 2.50,
		},
		"google/gemini-3-pro-preview": {
			Model:              "google/gemini-3-pro-preview",
			InputPricePerMTok:  2.00,
			OutputPricePerMTok: 12.00,
		},
	}
}
