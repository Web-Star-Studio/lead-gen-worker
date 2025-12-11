package handlers

import (
	"log"
	"sync"
	"time"

	"webstar/noturno-leadgen-worker/internal/dto"
)

const (
	// CharsPerToken is the approximate number of characters per token for estimation
	CharsPerToken = 4
)

// UsageTrackerHandler tracks AI usage metrics
type UsageTrackerHandler struct {
	supabase *SupabaseHandler
	pricing  map[string]dto.TokenPricing
	mu       sync.Mutex
}

// NewUsageTrackerHandler creates a new UsageTrackerHandler
func NewUsageTrackerHandler(supabase *SupabaseHandler) *UsageTrackerHandler {
	return &UsageTrackerHandler{
		supabase: supabase,
		pricing:  dto.DefaultTokenPricing(),
	}
}

// EstimateTokens estimates token count from text length
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + CharsPerToken - 1) / CharsPerToken
}

// CalculateCost calculates the estimated cost for a given operation
func (h *UsageTrackerHandler) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := h.pricing[model]
	if !ok {
		// Default to flash pricing if model not found
		pricing = h.pricing["gemini-2.5-flash"]
	}

	inputCost := float64(inputTokens) * pricing.InputPricePerMTok / 1_000_000
	outputCost := float64(outputTokens) * pricing.OutputPricePerMTok / 1_000_000

	return inputCost + outputCost
}

// TrackOperationInput contains the data needed to track an operation
type TrackOperationInput struct {
	UserID        string
	JobID         *string
	LeadID        *string
	OperationType dto.OperationType
	Model         string
	InputText     string
	OutputText    string
	StartTime     time.Time
	Success       bool
	ErrorMessage  *string
}

// TrackOperation records an AI operation for usage tracking
func (h *UsageTrackerHandler) TrackOperation(input TrackOperationInput) error {
	if h.supabase == nil {
		log.Printf("[UsageTracker] Supabase not configured, skipping tracking")
		return nil
	}

	inputTokens := EstimateTokens(input.InputText)
	outputTokens := EstimateTokens(input.OutputText)
	totalTokens := inputTokens + outputTokens
	durationMs := time.Since(input.StartTime).Milliseconds()
	cost := h.CalculateCost(input.Model, inputTokens, outputTokens)

	metric := dto.UsageMetricInput{
		UserID:          input.UserID,
		JobID:           input.JobID,
		LeadID:          input.LeadID,
		OperationType:   input.OperationType,
		Model:           input.Model,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		TotalTokens:     totalTokens,
		EstimatedCostUS: cost,
		DurationMs:      durationMs,
		Success:         input.Success,
		ErrorMessage:    input.ErrorMessage,
	}

	if err := h.supabase.InsertUsageMetric(&metric); err != nil {
		log.Printf("[UsageTracker] Failed to insert usage metric: %v", err)
		return err
	}

	log.Printf("[UsageTracker] Tracked %s: tokens=%d (in=%d, out=%d), cost=$%.6f, duration=%dms, success=%v",
		input.OperationType, totalTokens, inputTokens, outputTokens, cost, durationMs, input.Success)

	return nil
}

// TrackDataExtraction is a convenience method for tracking data extraction operations
func (h *UsageTrackerHandler) TrackDataExtraction(userID string, jobID, leadID *string, model, inputText, outputText string, startTime time.Time, success bool, errorMsg *string) {
	_ = h.TrackOperation(TrackOperationInput{
		UserID:        userID,
		JobID:         jobID,
		LeadID:        leadID,
		OperationType: dto.OperationDataExtraction,
		Model:         model,
		InputText:     inputText,
		OutputText:    outputText,
		StartTime:     startTime,
		Success:       success,
		ErrorMessage:  errorMsg,
	})
}

// TrackPreCallReport is a convenience method for tracking pre-call report operations
func (h *UsageTrackerHandler) TrackPreCallReport(userID string, jobID, leadID *string, model, inputText, outputText string, startTime time.Time, success bool, errorMsg *string) {
	_ = h.TrackOperation(TrackOperationInput{
		UserID:        userID,
		JobID:         jobID,
		LeadID:        leadID,
		OperationType: dto.OperationPreCallReport,
		Model:         model,
		InputText:     inputText,
		OutputText:    outputText,
		StartTime:     startTime,
		Success:       success,
		ErrorMessage:  errorMsg,
	})
}

// TrackColdEmail is a convenience method for tracking cold email operations
func (h *UsageTrackerHandler) TrackColdEmail(userID string, jobID, leadID *string, model, inputText, outputText string, startTime time.Time, success bool, errorMsg *string) {
	_ = h.TrackOperation(TrackOperationInput{
		UserID:        userID,
		JobID:         jobID,
		LeadID:        leadID,
		OperationType: dto.OperationColdEmail,
		Model:         model,
		InputText:     inputText,
		OutputText:    outputText,
		StartTime:     startTime,
		Success:       success,
		ErrorMessage:  errorMsg,
	})
}

// TrackWebsiteScraping is a convenience method for tracking website scraping operations
func (h *UsageTrackerHandler) TrackWebsiteScraping(userID string, jobID, leadID *string, inputURL string, outputSize int, startTime time.Time, success bool, errorMsg *string) {
	// For scraping, we don't have AI tokens, but we track the operation
	durationMs := time.Since(startTime).Milliseconds()

	if h.supabase == nil {
		log.Printf("[UsageTracker] Supabase not configured, skipping tracking")
		return
	}

	metric := dto.UsageMetricInput{
		UserID:          userID,
		JobID:           jobID,
		LeadID:          leadID,
		OperationType:   dto.OperationWebsiteScraping,
		Model:           "firecrawl",
		InputTokens:     0,
		OutputTokens:    outputSize / CharsPerToken,
		TotalTokens:     outputSize / CharsPerToken,
		EstimatedCostUS: 0, // Firecrawl has its own pricing
		DurationMs:      durationMs,
		Success:         success,
		ErrorMessage:    errorMsg,
	}

	if err := h.supabase.InsertUsageMetric(&metric); err != nil {
		log.Printf("[UsageTracker] Failed to insert scraping metric: %v", err)
	}
}
