package controllers

import (
	"net/http"
	"time"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/handlers"

	"github.com/gin-gonic/gin"
)

// ReportsController handles report-related HTTP requests
type ReportsController struct {
	supabaseHandler *handlers.SupabaseHandler
}

// NewReportsController creates a new ReportsController instance
func NewReportsController(supabaseHandler *handlers.SupabaseHandler) *ReportsController {
	return &ReportsController{
		supabaseHandler: supabaseHandler,
	}
}

// GetReports returns usage reports for the dashboard
// @Summary Get usage reports
// @Description Retrieves comprehensive usage reports including token usage, costs, and lead generation metrics
// @Tags Reports
// @Accept json
// @Produce json
// @Param user_id query string true "User ID to get reports for"
// @Param start_date query string false "Start date for the report period (RFC3339 format)"
// @Param end_date query string false "End date for the report period (RFC3339 format)"
// @Success 200 {object} dto.ReportsResponse "Usage reports"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/reports [get]
func (c *ReportsController) GetReports(ctx *gin.Context) {
	// Parse request parameters
	userID := ctx.Query("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	var startDate, endDate *time.Time

	if startStr := ctx.Query("start_date"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Try date-only format
			t, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid start_date format, use RFC3339 or YYYY-MM-DD",
				})
				return
			}
		}
		startDate = &t
	}

	if endStr := ctx.Query("end_date"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			// Try date-only format
			t, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid end_date format, use RFC3339 or YYYY-MM-DD",
				})
				return
			}
			// Set to end of day
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		endDate = &t
	}

	// If no dates provided, default to last 30 days
	if startDate == nil {
		t := time.Now().AddDate(0, 0, -30)
		startDate = &t
	}
	if endDate == nil {
		t := time.Now()
		endDate = &t
	}

	// Get usage summary
	summary, err := c.supabaseHandler.GetUsageSummary(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get usage summary: " + err.Error(),
		})
		return
	}

	// Get usage by operation
	byOperation, err := c.supabaseHandler.GetUsageByOperation(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get usage by operation: " + err.Error(),
		})
		return
	}

	// Get usage by model
	byModel, err := c.supabaseHandler.GetUsageByModel(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get usage by model: " + err.Error(),
		})
		return
	}

	// Get daily usage
	dailyUsage, err := c.supabaseHandler.GetDailyUsage(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get daily usage: " + err.Error(),
		})
		return
	}

	// Get lead generation stats
	leadGenStats, err := c.supabaseHandler.GetLeadGenerationStats(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get lead generation stats: " + err.Error(),
		})
		return
	}

	// Calculate period info
	daysCount := int(endDate.Sub(*startDate).Hours() / 24)
	if daysCount < 1 {
		daysCount = 1
	}

	response := dto.ReportsResponse{
		Summary:        *summary,
		ByOperation:    byOperation,
		ByModel:        byModel,
		DailyUsage:     dailyUsage,
		LeadGeneration: *leadGenStats,
		Period: dto.ReportPeriod{
			StartDate: startDate.Format("2006-01-02"),
			EndDate:   endDate.Format("2006-01-02"),
			DaysCount: daysCount,
		},
	}

	// Ensure slices are not nil for JSON serialization
	if response.ByOperation == nil {
		response.ByOperation = []dto.OperationStats{}
	}
	if response.ByModel == nil {
		response.ByModel = []dto.ModelUsage{}
	}
	if response.DailyUsage == nil {
		response.DailyUsage = []dto.DailyUsage{}
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUsageSummary returns a quick usage summary
// @Summary Get usage summary
// @Description Retrieves a quick summary of token usage and costs
// @Tags Reports
// @Accept json
// @Produce json
// @Param user_id query string true "User ID to get summary for"
// @Param start_date query string false "Start date (RFC3339 or YYYY-MM-DD)"
// @Param end_date query string false "End date (RFC3339 or YYYY-MM-DD)"
// @Success 200 {object} dto.UsageSummary "Usage summary"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/reports/summary [get]
func (c *ReportsController) GetUsageSummary(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	var startDate, endDate *time.Time

	if startStr := ctx.Query("start_date"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid start_date format",
				})
				return
			}
		}
		startDate = &t
	}

	if endStr := ctx.Query("end_date"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid end_date format",
				})
				return
			}
		}
		endDate = &t
	}

	summary, err := c.supabaseHandler.GetUsageSummary(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get usage summary: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, summary)
}

// GetDailyUsage returns daily usage statistics
// @Summary Get daily usage
// @Description Retrieves usage statistics aggregated by day for charts
// @Tags Reports
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Param start_date query string false "Start date (RFC3339 or YYYY-MM-DD)"
// @Param end_date query string false "End date (RFC3339 or YYYY-MM-DD)"
// @Success 200 {array} dto.DailyUsage "Daily usage statistics"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/reports/daily [get]
func (c *ReportsController) GetDailyUsage(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	var startDate, endDate *time.Time

	if startStr := ctx.Query("start_date"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid start_date format",
				})
				return
			}
		}
		startDate = &t
	}

	if endStr := ctx.Query("end_date"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid end_date format",
				})
				return
			}
		}
		endDate = &t
	}

	dailyUsage, err := c.supabaseHandler.GetDailyUsage(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get daily usage: " + err.Error(),
		})
		return
	}

	if dailyUsage == nil {
		dailyUsage = []dto.DailyUsage{}
	}

	ctx.JSON(http.StatusOK, dailyUsage)
}

// GetOperationStats returns usage statistics by operation type
// @Summary Get operation statistics
// @Description Retrieves usage statistics grouped by AI operation type
// @Tags Reports
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Param start_date query string false "Start date (RFC3339 or YYYY-MM-DD)"
// @Param end_date query string false "End date (RFC3339 or YYYY-MM-DD)"
// @Success 200 {array} dto.OperationStats "Operation statistics"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/reports/operations [get]
func (c *ReportsController) GetOperationStats(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	var startDate, endDate *time.Time

	if startStr := ctx.Query("start_date"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", startStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid start_date format",
				})
				return
			}
		}
		startDate = &t
	}

	if endStr := ctx.Query("end_date"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid end_date format",
				})
				return
			}
		}
		endDate = &t
	}

	stats, err := c.supabaseHandler.GetUsageByOperation(userID, startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get operation stats: " + err.Error(),
		})
		return
	}

	if stats == nil {
		stats = []dto.OperationStats{}
	}

	ctx.JSON(http.StatusOK, stats)
}
