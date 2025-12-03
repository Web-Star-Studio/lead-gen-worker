package controllers

import (
	"context"
	"log"
	"net/http"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/services"

	"github.com/gin-gonic/gin"
)

// AutomationController handles automation webhook requests
type AutomationController struct {
	webhookSecret string
	processor     *services.AutomationProcessor
}

// NewAutomationController creates a new AutomationController
func NewAutomationController(webhookSecret string, processor *services.AutomationProcessor) *AutomationController {
	return &AutomationController{
		webhookSecret: webhookSecret,
		processor:     processor,
	}
}

// HandleAutomationTask handles POST /webhooks/automation-task
// Called when a new automation_task is created (from frontend or Supabase trigger)
// @Summary Handle automation task webhook
// @Description Receives webhook when a new automation task is created
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token with webhook secret"
// @Param payload body dto.AutomationTask true "Automation task payload"
// @Success 200 {object} map[string]string "Task accepted"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Router /webhooks/automation-task [post]
func (c *AutomationController) HandleAutomationTask(ctx *gin.Context) {
	// Validate auth
	if !c.validateAuth(ctx) {
		return
	}

	var task dto.AutomationTask
	if err := ctx.ShouldBindJSON(&task); err != nil {
		log.Printf("[AutomationController] Failed to parse task payload: %v", err)
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid task payload"})
		return
	}

	leadCount := len(task.LeadIDs)
	if task.LeadID != nil {
		leadCount++
	}

	log.Printf("[AutomationController] Task received: id=%s, type=%s, leads=%d",
		task.ID, task.TaskType, leadCount)

	ctx.JSON(http.StatusOK, gin.H{"status": "accepted", "task_id": task.ID})

	// Process in background
	go c.processor.ProcessTask(context.Background(), &task)
}

// HandleLeadCreated handles POST /webhooks/lead-created
// Called when a new lead is created (for auto-enrichment based on user config)
// @Summary Handle lead created webhook
// @Description Receives webhook when a new lead is created for auto-enrichment
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token with webhook secret"
// @Param payload body dto.Lead true "Lead payload"
// @Success 200 {object} map[string]string "Lead accepted"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Router /webhooks/lead-created [post]
func (c *AutomationController) HandleLeadCreated(ctx *gin.Context) {
	// Validate auth
	if !c.validateAuth(ctx) {
		return
	}

	var lead dto.Lead
	if err := ctx.ShouldBindJSON(&lead); err != nil {
		log.Printf("[AutomationController] Failed to parse lead payload: %v", err)
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid lead payload"})
		return
	}

	log.Printf("[AutomationController] Lead created webhook: id=%s, company=%s, user=%s",
		lead.ID, lead.CompanyName, lead.UserID)

	ctx.JSON(http.StatusOK, gin.H{"status": "accepted", "lead_id": lead.ID})

	// Check for auto-enrichment in background
	go c.processor.ProcessLeadCreated(context.Background(), &lead)
}

// HandleBatchEnrichment handles POST /webhooks/batch-enrichment
// Manual trigger to enrich multiple leads at once
// @Summary Handle batch enrichment request
// @Description Manually trigger enrichment for a batch of leads
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token with webhook secret"
// @Param payload body dto.AutomationTaskCreate true "Batch enrichment request"
// @Success 200 {object} map[string]string "Batch accepted"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Router /webhooks/batch-enrichment [post]
func (c *AutomationController) HandleBatchEnrichment(ctx *gin.Context) {
	// Validate auth
	if !c.validateAuth(ctx) {
		return
	}

	var request dto.AutomationTaskCreate
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Printf("[AutomationController] Failed to parse batch request: %v", err)
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid batch request"})
		return
	}

	// Validate task type
	if request.TaskType == "" {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "task_type is required"})
		return
	}

	// Validate leads
	leadCount := len(request.LeadIDs)
	if request.LeadID != nil {
		leadCount++
	}
	if leadCount == 0 {
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "at least one lead_id is required"})
		return
	}

	log.Printf("[AutomationController] Batch enrichment: type=%s, leads=%d, user=%s",
		request.TaskType, leadCount, request.UserID)

	// Set default priority if not provided
	if request.Priority == 0 {
		request.Priority = dto.TaskPriorityLow // Manual batch = low priority
	}

	// Create task from request
	task := &dto.AutomationTask{
		ID:                generateTaskID(),
		UserID:            request.UserID,
		TaskType:          request.TaskType,
		LeadID:            request.LeadID,
		LeadIDs:           request.LeadIDs,
		BusinessProfileID: request.BusinessProfileID,
		Priority:          request.Priority,
		Status:            dto.TaskStatusPending,
		ItemsTotal:        leadCount,
		MaxRetries:        services.MaxRetries,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "accepted",
		"task_id": task.ID,
		"leads":   leadCount,
	})

	// Process in background
	go c.processor.ProcessTask(context.Background(), task)
}

func (c *AutomationController) validateAuth(ctx *gin.Context) bool {
	authHeader := ctx.GetHeader("Authorization")
	expectedAuth := "Bearer " + c.webhookSecret

	if authHeader != expectedAuth {
		log.Printf("[AutomationController] Unauthorized request")
		ctx.JSON(http.StatusUnauthorized, dto.ErrorResponse{Error: "Unauthorized"})
		return false
	}
	return true
}

func generateTaskID() string {
	return "task-" + randomString(16)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
