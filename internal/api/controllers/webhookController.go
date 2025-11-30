package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/services"

	"github.com/gin-gonic/gin"
)

// WebhookController handles Supabase database webhook requests
type WebhookController struct {
	webhookSecret string
	processor     *services.JobProcessor
}

// NewWebhookController creates a new WebhookController instance
func NewWebhookController(webhookSecret string, processor *services.JobProcessor) *WebhookController {
	return &WebhookController{
		webhookSecret: webhookSecret,
		processor:     processor,
	}
}

// HandleJobCreated handles POST /webhooks/job-created
// This endpoint is called by Supabase when a new job is inserted
// @Summary Handle job created webhook
// @Description Receives Supabase database webhook when a new job is created
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token with webhook secret"
// @Param payload body dto.WebhookPayload true "Webhook payload from Supabase"
// @Success 200 {object} map[string]string "Webhook accepted"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Router /webhooks/job-created [post]
func (c *WebhookController) HandleJobCreated(ctx *gin.Context) {
	// 1. Validate Authorization header
	authHeader := ctx.GetHeader("Authorization")
	expectedAuth := "Bearer " + c.webhookSecret

	if authHeader != expectedAuth {
		log.Printf("[WebhookController] Unauthorized request: invalid Authorization header")
		ctx.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized: invalid webhook secret",
		})
		return
	}

	// 2. Parse webhook payload
	var payload dto.WebhookPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		log.Printf("[WebhookController] Failed to parse webhook payload: %v", err)
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid webhook payload",
		})
		return
	}

	log.Printf("[WebhookController] Received webhook: type=%s, table=%s, schema=%s",
		payload.Type, payload.Table, payload.Schema)

	// 3. Ignore if not INSERT on jobs table
	if strings.ToUpper(payload.Type) != "INSERT" || payload.Table != "jobs" {
		log.Printf("[WebhookController] Ignoring webhook: not an INSERT on jobs table")
		ctx.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	// 4. Parse Job from record
	var job dto.Job
	if err := json.Unmarshal(payload.Record, &job); err != nil {
		log.Printf("[WebhookController] Failed to parse job record: %v", err)
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid job record in webhook payload",
		})
		return
	}

	log.Printf("[WebhookController] Job received: id=%s, icp_name=%s, region=%s, lead_quantity=%d",
		job.ID, job.ICPName, job.Region, job.LeadQuantity)

	// 5. Respond 200 immediately (non-blocking)
	ctx.JSON(http.StatusOK, gin.H{
		"status": "accepted",
		"job_id": job.ID,
	})

	// 6. Process job in background
	go c.processor.ProcessJob(context.Background(), &job)
}
