package controllers

import (
	"context"
	"log"
	"net/http"

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

	// 2. Parse job payload directly (custom format from frontend)
	var job dto.Job
	if err := ctx.ShouldBindJSON(&job); err != nil {
		log.Printf("[WebhookController] Failed to parse job payload: %v", err)
		ctx.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid job payload",
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
