package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/services"

	"github.com/gin-gonic/gin"
)

// automationControllerLog provides structured logging for automation controller
func automationControllerLog(level, msg string, fields map[string]interface{}) {
	fieldStr := ""
	for k, v := range fields {
		fieldStr += fmt.Sprintf(" %s=%v", k, v)
	}
	log.Printf("[AutomationController] [%s] %s%s", level, msg, fieldStr)
}

// SupabaseWebhookPayload wraps the actual record data from Supabase webhooks
type SupabaseWebhookPayload struct {
	Type      string                 `json:"type"`       // INSERT, UPDATE, DELETE
	Table     string                 `json:"table"`      // Table name
	Schema    string                 `json:"schema"`     // Schema name (usually "public")
	Record    map[string]interface{} `json:"record"`     // The actual record data
	OldRecord map[string]interface{} `json:"old_record"` // Previous record (for UPDATE/DELETE)
}

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
	requestTime := time.Now()
	clientIP := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	automationControllerLog("INFO", "Webhook received: automation-task", map[string]interface{}{
		"endpoint":    "/webhooks/automation-task",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"received_at": requestTime.Format(time.RFC3339),
	})

	// Validate auth
	if !c.validateAuth(ctx) {
		automationControllerLog("WARN", "Unauthorized request rejected", map[string]interface{}{
			"endpoint":  "/webhooks/automation-task",
			"client_ip": clientIP,
		})
		return
	}

	// Read raw body for potential re-parsing
	rawBody, _ := io.ReadAll(ctx.Request.Body)
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))

	// Log raw payload for debugging
	automationControllerLog("DEBUG", "Raw webhook payload", map[string]interface{}{
		"payload_size": len(rawBody),
		"payload":      string(rawBody),
	})

	// Try to parse as Supabase webhook format first
	var task dto.AutomationTask
	var webhook SupabaseWebhookPayload

	if err := json.Unmarshal(rawBody, &webhook); err == nil && webhook.Record != nil {
		// Supabase webhook format - extract record
		automationControllerLog("INFO", "Parsing Supabase webhook format", map[string]interface{}{
			"type":   webhook.Type,
			"table":  webhook.Table,
			"schema": webhook.Schema,
		})
		task = parseAutomationTaskFromRecord(webhook.Record)
	} else {
		// Try direct format (for manual API calls)
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
		if err := ctx.ShouldBindJSON(&task); err != nil {
			automationControllerLog("ERROR", "Failed to parse task payload", map[string]interface{}{
				"endpoint":  "/webhooks/automation-task",
				"client_ip": clientIP,
				"error":     err.Error(),
			})
			ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid task payload"})
			return
		}
	}

	leadCount := len(task.LeadIDs)
	if task.LeadID != nil {
		leadCount++
	}

	automationControllerLog("INFO", "Task accepted for processing", map[string]interface{}{
		"task_id":             task.ID,
		"user_id":             task.UserID,
		"task_type":           task.TaskType,
		"lead_count":          leadCount,
		"priority":            task.Priority,
		"business_profile_id": task.BusinessProfileID,
		"client_ip":           clientIP,
	})

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
	requestTime := time.Now()
	clientIP := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	automationControllerLog("INFO", "Webhook received: lead-created", map[string]interface{}{
		"endpoint":    "/webhooks/lead-created",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"received_at": requestTime.Format(time.RFC3339),
	})

	// Validate auth
	if !c.validateAuth(ctx) {
		automationControllerLog("WARN", "Unauthorized request rejected", map[string]interface{}{
			"endpoint":  "/webhooks/lead-created",
			"client_ip": clientIP,
		})
		return
	}

	// Read raw body for potential re-parsing
	rawBody, _ := io.ReadAll(ctx.Request.Body)
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))

	// Log raw payload for debugging
	automationControllerLog("DEBUG", "Raw webhook payload", map[string]interface{}{
		"payload_size": len(rawBody),
		"payload":      string(rawBody),
	})

	// Try to parse as Supabase webhook format first
	var lead dto.Lead
	var webhook SupabaseWebhookPayload

	if err := json.Unmarshal(rawBody, &webhook); err == nil && webhook.Record != nil {
		// Supabase webhook format - extract record
		automationControllerLog("INFO", "Parsing Supabase webhook format", map[string]interface{}{
			"type":   webhook.Type,
			"table":  webhook.Table,
			"schema": webhook.Schema,
		})
		lead = parseLeadFromRecord(webhook.Record)
	} else {
		// Try direct format (for manual API calls)
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
		if err := ctx.ShouldBindJSON(&lead); err != nil {
			automationControllerLog("ERROR", "Failed to parse lead payload", map[string]interface{}{
				"endpoint":  "/webhooks/lead-created",
				"client_ip": clientIP,
				"error":     err.Error(),
			})
			ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid lead payload"})
			return
		}
	}

	automationControllerLog("INFO", "Lead created - checking for auto-enrichment", map[string]interface{}{
		"lead_id":      lead.ID,
		"user_id":      lead.UserID,
		"company_name": lead.CompanyName,
		"website":      lead.Website,
		"client_ip":    clientIP,
	})

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
	requestTime := time.Now()
	clientIP := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	automationControllerLog("INFO", "Webhook received: batch-enrichment", map[string]interface{}{
		"endpoint":    "/webhooks/batch-enrichment",
		"client_ip":   clientIP,
		"user_agent":  userAgent,
		"received_at": requestTime.Format(time.RFC3339),
	})

	// Validate auth
	if !c.validateAuth(ctx) {
		automationControllerLog("WARN", "Unauthorized request rejected", map[string]interface{}{
			"endpoint":  "/webhooks/batch-enrichment",
			"client_ip": clientIP,
		})
		return
	}

	var request dto.AutomationTaskCreate
	if err := ctx.ShouldBindJSON(&request); err != nil {
		automationControllerLog("ERROR", "Failed to parse batch request", map[string]interface{}{
			"endpoint":  "/webhooks/batch-enrichment",
			"client_ip": clientIP,
			"error":     err.Error(),
		})
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "Invalid batch request"})
		return
	}

	// Validate task type
	if request.TaskType == "" {
		automationControllerLog("WARN", "Missing task_type in request", map[string]interface{}{
			"user_id":   request.UserID,
			"client_ip": clientIP,
		})
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "task_type is required"})
		return
	}

	// Validate leads
	leadCount := len(request.LeadIDs)
	if request.LeadID != nil {
		leadCount++
	}
	if leadCount == 0 {
		automationControllerLog("WARN", "No leads provided in batch request", map[string]interface{}{
			"user_id":   request.UserID,
			"task_type": request.TaskType,
			"client_ip": clientIP,
		})
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "at least one lead_id is required"})
		return
	}

	automationControllerLog("INFO", "Batch enrichment request accepted", map[string]interface{}{
		"user_id":             request.UserID,
		"task_type":           request.TaskType,
		"lead_count":          leadCount,
		"business_profile_id": request.BusinessProfileID,
		"priority":            request.Priority,
		"client_ip":           clientIP,
	})

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
		// Don't log the actual auth header for security reasons
		hasAuth := authHeader != ""
		automationControllerLog("WARN", "Authentication failed", map[string]interface{}{
			"has_auth_header": hasAuth,
			"client_ip":       ctx.ClientIP(),
			"path":            ctx.Request.URL.Path,
		})
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

// parseLeadFromRecord parses a Lead from a Supabase webhook record
func parseLeadFromRecord(record map[string]interface{}) dto.Lead {
	lead := dto.Lead{}

	if id, ok := record["id"].(string); ok {
		lead.ID = id
	}
	if userID, ok := record["user_id"].(string); ok {
		lead.UserID = userID
	}
	if jobID, ok := record["job_id"].(string); ok {
		lead.JobID = jobID
	}
	if companyName, ok := record["company_name"].(string); ok {
		lead.CompanyName = companyName
	}
	if contactName, ok := record["contact_name"].(string); ok {
		lead.ContactName = contactName
	}
	if contactRole, ok := record["contact_role"].(string); ok {
		lead.ContactRole = contactRole
	}
	if website, ok := record["website"].(string); ok {
		lead.Website = &website
	}
	if address, ok := record["address"].(string); ok {
		lead.Address = address
	}
	if source, ok := record["source"].(string); ok {
		lead.Source = source
	}

	// Parse emails array
	if emails, ok := record["emails"].([]interface{}); ok {
		for _, e := range emails {
			if email, ok := e.(string); ok {
				lead.Emails = append(lead.Emails, email)
			}
		}
	}

	// Parse phones array
	if phones, ok := record["phones"].([]interface{}); ok {
		for _, p := range phones {
			if phone, ok := p.(string); ok {
				lead.Phones = append(lead.Phones, phone)
			}
		}
	}

	// Parse social_media map
	if socialMedia, ok := record["social_media"].(map[string]interface{}); ok {
		lead.SocialMedia = make(map[string]string)
		for k, v := range socialMedia {
			if str, ok := v.(string); ok {
				lead.SocialMedia[k] = str
			}
		}
	}

	automationControllerLog("DEBUG", "Parsed lead from record", map[string]interface{}{
		"lead_id":      lead.ID,
		"user_id":      lead.UserID,
		"company_name": lead.CompanyName,
		"website":      lead.Website,
	})

	return lead
}

// parseAutomationTaskFromRecord parses an AutomationTask from a Supabase webhook record
func parseAutomationTaskFromRecord(record map[string]interface{}) dto.AutomationTask {
	task := dto.AutomationTask{}

	if id, ok := record["id"].(string); ok {
		task.ID = id
	}
	if userID, ok := record["user_id"].(string); ok {
		task.UserID = userID
	}
	if taskType, ok := record["task_type"].(string); ok {
		task.TaskType = dto.TaskType(taskType)
	}
	if leadID, ok := record["lead_id"].(string); ok {
		task.LeadID = &leadID
	}
	if businessProfileID, ok := record["business_profile_id"].(string); ok {
		task.BusinessProfileID = &businessProfileID
	}
	if priority, ok := record["priority"].(float64); ok {
		task.Priority = dto.TaskPriority(int(priority))
	}
	if status, ok := record["status"].(string); ok {
		task.Status = dto.TaskStatus(status)
	}
	if itemsTotal, ok := record["items_total"].(float64); ok {
		task.ItemsTotal = int(itemsTotal)
	}
	if maxRetries, ok := record["max_retries"].(float64); ok {
		task.MaxRetries = int(maxRetries)
	}

	// Parse lead_ids array
	if leadIDs, ok := record["lead_ids"].([]interface{}); ok {
		for _, lid := range leadIDs {
			if leadID, ok := lid.(string); ok {
				task.LeadIDs = append(task.LeadIDs, leadID)
			}
		}
	}

	automationControllerLog("DEBUG", "Parsed automation task from record", map[string]interface{}{
		"task_id":    task.ID,
		"user_id":    task.UserID,
		"task_type":  task.TaskType,
		"lead_count": len(task.LeadIDs),
	})

	return task
}
