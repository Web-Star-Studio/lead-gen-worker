package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/handlers"
)

const (
	MaxConcurrentScrapes = 5 // Firecrawl limit
	MaxRetries           = 2
	RetryDelay           = 5 * time.Second
)

// AutomationLogger provides structured logging for automation operations
type AutomationLogger struct {
	prefix string
}

func (l *AutomationLogger) log(level, msg string, fields map[string]interface{}) {
	fieldStr := ""
	for k, v := range fields {
		fieldStr += fmt.Sprintf(" %s=%v", k, v)
	}
	log.Printf("[%s] [%s] %s%s", l.prefix, level, msg, fieldStr)
}

func (l *AutomationLogger) Info(msg string, fields map[string]interface{}) {
	l.log("INFO", msg, fields)
}

func (l *AutomationLogger) Warn(msg string, fields map[string]interface{}) {
	l.log("WARN", msg, fields)
}

func (l *AutomationLogger) Error(msg string, fields map[string]interface{}) {
	l.log("ERROR", msg, fields)
}

func (l *AutomationLogger) Debug(msg string, fields map[string]interface{}) {
	l.log("DEBUG", msg, fields)
}

var automationLog = &AutomationLogger{prefix: "AutomationProcessor"}

// AutomationProcessor handles automation tasks (enrichment, pre-call, email generation)
type AutomationProcessor struct {
	supabase             *handlers.SupabaseHandler
	firecrawlHandler     *handlers.FirecrawlHandler
	dataExtractorHandler *handlers.DataExtractorHandler
	preCallReportHandler *handlers.PreCallReportHandler
	coldEmailHandler     *handlers.ColdEmailHandler
}

// NewAutomationProcessor creates a new AutomationProcessor instance
func NewAutomationProcessor(
	supabase *handlers.SupabaseHandler,
	firecrawl *handlers.FirecrawlHandler,
	extractor *handlers.DataExtractorHandler,
	preCall *handlers.PreCallReportHandler,
	coldEmail *handlers.ColdEmailHandler,
) *AutomationProcessor {
	// Log handler capabilities on initialization
	automationLog.Info("Initializing AutomationProcessor", map[string]interface{}{
		"supabase_enabled":  supabase != nil,
		"firecrawl_enabled": firecrawl != nil,
		"extractor_enabled": extractor != nil,
		"precall_enabled":   preCall != nil,
		"coldemail_enabled": coldEmail != nil,
		"max_concurrent":    MaxConcurrentScrapes,
		"max_retries":       MaxRetries,
		"retry_delay_sec":   RetryDelay.Seconds(),
	})

	return &AutomationProcessor{
		supabase:             supabase,
		firecrawlHandler:     firecrawl,
		dataExtractorHandler: extractor,
		preCallReportHandler: preCall,
		coldEmailHandler:     coldEmail,
	}
}

// ProcessTask processes an automation task based on its type
func (p *AutomationProcessor) ProcessTask(ctx context.Context, task *dto.AutomationTask) {
	startTime := time.Now()

	// Check if task is already being processed or completed (prevent duplicate processing)
	currentStatus, err := p.supabase.GetAutomationTaskStatus(task.ID)
	if err != nil {
		automationLog.Warn("Could not verify task status, proceeding anyway", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
	} else if currentStatus != "pending" {
		automationLog.Info("Task already processed or in progress - skipping", map[string]interface{}{
			"task_id":        task.ID,
			"current_status": currentStatus,
		})
		return
	}

	automationLog.Info("═══════════════════════════════════════════════════════════", nil)
	automationLog.Info("TASK STARTED", map[string]interface{}{
		"task_id":             task.ID,
		"user_id":             task.UserID,
		"task_type":           task.TaskType,
		"priority":            task.Priority,
		"business_profile_id": task.BusinessProfileID,
		"started_at":          startTime.Format(time.RFC3339),
	})

	// Update status to processing
	if err := p.supabase.UpdateAutomationTaskStatus(task.ID, string(dto.TaskStatusProcessing), 0, 0, 0, nil); err != nil {
		automationLog.Error("Failed to update task status to processing", map[string]interface{}{
			"task_id": task.ID,
			"error":   err.Error(),
		})
		return
	}

	// Set user context on handlers for usage tracking
	if p.dataExtractorHandler != nil {
		p.dataExtractorHandler.SetUserContext(task.UserID, &task.ID)
		defer p.dataExtractorHandler.ClearUserContext()
	}
	if p.preCallReportHandler != nil {
		p.preCallReportHandler.SetUserContext(task.UserID, &task.ID)
		defer p.preCallReportHandler.ClearUserContext()
	}
	if p.coldEmailHandler != nil {
		p.coldEmailHandler.SetUserContext(task.UserID, &task.ID)
		defer p.coldEmailHandler.ClearUserContext()
	}

	// Collect lead IDs to process
	var leadIDs []string
	if task.LeadID != nil {
		leadIDs = append(leadIDs, *task.LeadID)
	}
	leadIDs = append(leadIDs, task.LeadIDs...)

	if len(leadIDs) == 0 {
		errMsg := "no leads to process"
		automationLog.Error("Task failed - no leads provided", map[string]interface{}{
			"task_id": task.ID,
			"user_id": task.UserID,
		})
		p.supabase.UpdateAutomationTaskStatus(task.ID, string(dto.TaskStatusFailed), 0, 0, 0, &errMsg)
		return
	}

	automationLog.Info("Processing leads batch", map[string]interface{}{
		"task_id":    task.ID,
		"user_id":    task.UserID,
		"lead_count": len(leadIDs),
		"task_type":  task.TaskType,
	})

	// Update total items
	p.supabase.UpdateAutomationTaskStatus(task.ID, string(dto.TaskStatusProcessing), len(leadIDs), 0, 0, nil)

	// Process based on task type
	var results []dto.EnrichmentResult

	switch task.TaskType {
	case dto.TaskTypeLeadEnrichment:
		results = p.processEnrichment(ctx, leadIDs, task.ID)
	case dto.TaskTypePreCallGeneration:
		results = p.processPreCallGeneration(ctx, leadIDs, task.BusinessProfileID, task.ID)
	case dto.TaskTypeEmailGeneration:
		results = p.processEmailGeneration(ctx, leadIDs, task.BusinessProfileID, task.ID)
	case dto.TaskTypeFullEnrichment:
		results = p.processFullEnrichment(ctx, leadIDs, task.BusinessProfileID, task.ID)
	default:
		errMsg := fmt.Sprintf("unknown task type: %s", task.TaskType)
		automationLog.Error("Unknown task type", map[string]interface{}{
			"task_id":   task.ID,
			"user_id":   task.UserID,
			"task_type": task.TaskType,
		})
		p.supabase.UpdateAutomationTaskStatus(task.ID, string(dto.TaskStatusFailed), 0, 0, 0, &errMsg)
		return
	}

	// Count results
	succeeded := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			succeeded++
		} else {
			failed++
		}
	}

	// Update final status
	status := dto.TaskStatusCompleted
	if failed == len(results) {
		status = dto.TaskStatusFailed
	}

	duration := time.Since(startTime)
	p.supabase.UpdateAutomationTaskStatus(task.ID, string(status), len(results), succeeded, failed, nil)

	automationLog.Info("TASK COMPLETED", map[string]interface{}{
		"task_id":      task.ID,
		"user_id":      task.UserID,
		"task_type":    task.TaskType,
		"status":       status,
		"total":        len(results),
		"succeeded":    succeeded,
		"failed":       failed,
		"duration_sec": duration.Seconds(),
		"avg_per_lead": func() float64 {
			if len(results) > 0 {
				return duration.Seconds() / float64(len(results))
			}
			return 0
		}(),
	})
	automationLog.Info("═══════════════════════════════════════════════════════════", nil)
}

// processEnrichment scrapes websites and extracts data for leads
func (p *AutomationProcessor) processEnrichment(ctx context.Context, leadIDs []string, taskID string) []dto.EnrichmentResult {
	results := make([]dto.EnrichmentResult, len(leadIDs))

	// Process with semaphore to limit concurrent scrapes
	sem := make(chan struct{}, MaxConcurrentScrapes)
	var wg sync.WaitGroup
	var mu sync.Mutex
	processedCount := 0

	for i, leadID := range leadIDs {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			result := p.enrichSingleLead(ctx, id)

			mu.Lock()
			results[idx] = result
			processedCount++
			// Update progress in real-time
			succeeded := 0
			failed := 0
			for _, r := range results[:processedCount] {
				if r.Success {
					succeeded++
				} else if r.Error != "" {
					failed++
				}
			}
			p.supabase.UpdateAutomationTaskStatus(taskID, string(dto.TaskStatusProcessing), len(leadIDs), succeeded, failed, nil)
			mu.Unlock()

			automationLog.Debug("Enrichment progress", map[string]interface{}{
				"task_id":   taskID,
				"processed": processedCount,
				"total":     len(leadIDs),
				"percent":   float64(processedCount) / float64(len(leadIDs)) * 100,
			})
		}(i, leadID)
	}

	wg.Wait()
	return results
}

// enrichSingleLead enriches a single lead with scraped data
func (p *AutomationProcessor) enrichSingleLead(ctx context.Context, leadID string) dto.EnrichmentResult {
	result := dto.EnrichmentResult{LeadID: leadID}

	// Get lead data
	lead, err := p.supabase.GetLeadByID(leadID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get lead: %v", err)
		return result
	}

	// Need website to scrape
	if lead.Website == nil || *lead.Website == "" {
		automationLog.Warn("Lead has no website - skipping enrichment", map[string]interface{}{
			"lead_id":      leadID,
			"company_name": lead.CompanyName,
		})
		result.Error = "lead has no website"
		return result
	}

	// Scrape website with retry
	var scraped *handlers.ScrapedPage
	var scrapeErr error
	scrapeStart := time.Now()
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			automationLog.Warn("Retrying scrape", map[string]interface{}{
				"lead_id": leadID,
				"website": *lead.Website,
				"attempt": attempt,
				"max":     MaxRetries,
			})
			time.Sleep(RetryDelay)
		}
		scraped, scrapeErr = p.firecrawlHandler.ScrapeURL(*lead.Website)
		if scrapeErr == nil && scraped.Success {
			break
		}
	}
	scrapeDuration := time.Since(scrapeStart)

	if scrapeErr != nil || !scraped.Success {
		errMsg := "unknown error"
		if scrapeErr != nil {
			errMsg = scrapeErr.Error()
		} else if scraped != nil {
			errMsg = scraped.Error
		}
		automationLog.Error("Failed to scrape website", map[string]interface{}{
			"lead_id":      leadID,
			"website":      *lead.Website,
			"error":        errMsg,
			"duration_sec": scrapeDuration.Seconds(),
		})
		result.Error = fmt.Sprintf("failed to scrape website: %s", errMsg)
		return result
	}

	// Extract data
	orgResult := handlers.OrganicResult{
		Link:           *lead.Website,
		Title:          lead.CompanyName,
		ScrapedContent: scraped.Markdown,
	}
	extracted := p.dataExtractorHandler.ExtractData(ctx, orgResult)
	if !extracted.Success {
		result.Error = fmt.Sprintf("failed to extract data: %s", extracted.Error)
		return result
	}

	// Update lead with enriched data
	if err := p.supabase.UpdateLeadEnrichment(leadID, extracted); err != nil {
		result.Error = fmt.Sprintf("failed to update lead: %v", err)
		return result
	}

	result.Success = true
	result.Enriched = true
	automationLog.Info("✓ Lead enriched successfully", map[string]interface{}{
		"lead_id":         leadID,
		"company_name":    lead.CompanyName,
		"website":         *lead.Website,
		"scrape_duration": scrapeDuration.Seconds(),
		"emails_found":    len(extracted.Emails),
		"phones_found":    len(extracted.Phones),
		"contact_found":   extracted.Contact != "",
	})
	return result
}

// processPreCallGeneration generates pre-call reports for leads
func (p *AutomationProcessor) processPreCallGeneration(ctx context.Context, leadIDs []string, businessProfileID *string, taskID string) []dto.EnrichmentResult {
	results := make([]dto.EnrichmentResult, len(leadIDs))

	// Get business profile if provided
	var profile *dto.BusinessProfile
	if businessProfileID != nil {
		var err error
		profile, err = p.supabase.GetBusinessProfile(*businessProfileID)
		if err != nil {
			automationLog.Warn("Could not get business profile", map[string]interface{}{
				"task_id":             taskID,
				"business_profile_id": *businessProfileID,
				"error":               err.Error(),
			})
		} else {
			automationLog.Info("Using business profile for pre-call generation", map[string]interface{}{
				"task_id":      taskID,
				"profile_id":   *businessProfileID,
				"company_name": profile.CompanyName,
			})
		}
	}

	// Set business profile on handler
	if profile != nil {
		p.preCallReportHandler.SetBusinessProfile(profile)
		defer p.preCallReportHandler.ClearBusinessProfile()
	}

	for i, leadID := range leadIDs {
		results[i] = p.generatePreCallForLead(ctx, leadID)

		// Update progress
		succeeded := 0
		failed := 0
		for _, r := range results[:i+1] {
			if r.Success {
				succeeded++
			} else if r.Error != "" {
				failed++
			}
		}
		p.supabase.UpdateAutomationTaskStatus(taskID, string(dto.TaskStatusProcessing), len(leadIDs), succeeded, failed, nil)
		automationLog.Debug("Pre-call progress", map[string]interface{}{
			"task_id":   taskID,
			"processed": i + 1,
			"total":     len(leadIDs),
			"percent":   float64(i+1) / float64(len(leadIDs)) * 100,
		})
	}

	return results
}

// generatePreCallForLead generates a pre-call report for a single lead
func (p *AutomationProcessor) generatePreCallForLead(ctx context.Context, leadID string) dto.EnrichmentResult {
	result := dto.EnrichmentResult{LeadID: leadID}

	// Check if lead already has a pre-call report (prevent duplicates)
	hasPreCall, err := p.supabase.LeadHasPreCallReport(leadID)
	if err != nil {
		automationLog.Warn("Could not check for existing pre-call report, proceeding anyway", map[string]interface{}{
			"lead_id": leadID,
			"error":   err.Error(),
		})
	} else if hasPreCall {
		automationLog.Info("Lead already has pre-call report - skipping generation", map[string]interface{}{
			"lead_id": leadID,
		})
		result.Success = true
		result.PreCall = true
		return result
	}

	// Get lead data
	lead, err := p.supabase.GetLeadByID(leadID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get lead: %v", err)
		return result
	}

	// Build organic result from lead for report generation
	orgResult := handlers.OrganicResult{
		Title: lead.CompanyName,
	}
	if lead.Website != nil {
		orgResult.Link = *lead.Website
	}

	// Build ExtractedData from lead data
	orgResult.ExtractedData = &handlers.ExtractedData{
		Success:     true,
		Company:     lead.CompanyName,
		Contact:     lead.ContactName,
		ContactRole: lead.ContactRole,
		Emails:      lead.Emails,
		Phones:      lead.Phones,
		Address:     lead.Address,
		SocialMedia: lead.SocialMedia,
	}

	// Try to get scraped content if we have website
	if lead.Website != nil && *lead.Website != "" {
		scraped, err := p.firecrawlHandler.ScrapeURL(*lead.Website)
		if err == nil && scraped.Success {
			orgResult.ScrapedContent = scraped.Markdown
		}
	}

	// If no scraped content but we have extra_data from CNPJ import, build rich content
	if orgResult.ScrapedContent == "" && lead.ExtraData != nil {
		orgResult.ScrapedContent = buildContentFromExtraData(lead)
		automationLog.Info("Using CNPJ data for pre-call generation", map[string]interface{}{
			"lead_id":          leadID,
			"cnpj":             lead.ExtraData.CNPJ,
			"cnae_description": lead.ExtraData.CNAEDescription,
		})
	}

	// Generate report with retry
	var report *handlers.PreCallReport
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[AutomationProcessor] Retry %d for pre-call lead %s", attempt, leadID)
			time.Sleep(RetryDelay)
		}
		report = p.preCallReportHandler.GenerateReport(ctx, orgResult)
		if report.Success {
			break
		}
	}

	if !report.Success {
		result.Error = fmt.Sprintf("failed to generate pre-call: %s", report.Error)
		return result
	}

	// Save to database
	if err := p.supabase.InsertPreCallReport(leadID, report.CompanySummary); err != nil {
		result.Error = fmt.Sprintf("failed to save pre-call: %v", err)
		return result
	}

	result.Success = true
	result.PreCall = true
	automationLog.Info("✓ Pre-call report generated", map[string]interface{}{
		"lead_id":      leadID,
		"company_name": lead.CompanyName,
	})
	return result
}

// processEmailGeneration generates cold emails for leads
func (p *AutomationProcessor) processEmailGeneration(ctx context.Context, leadIDs []string, businessProfileID *string, taskID string) []dto.EnrichmentResult {
	results := make([]dto.EnrichmentResult, len(leadIDs))

	// Get business profile if provided
	var profile *dto.BusinessProfile
	if businessProfileID != nil {
		var err error
		profile, err = p.supabase.GetBusinessProfile(*businessProfileID)
		if err != nil {
			automationLog.Warn("Could not get business profile", map[string]interface{}{
				"task_id":             taskID,
				"business_profile_id": *businessProfileID,
				"error":               err.Error(),
			})
		} else {
			automationLog.Info("Using business profile for email generation", map[string]interface{}{
				"task_id":      taskID,
				"profile_id":   *businessProfileID,
				"company_name": profile.CompanyName,
				"sender_name":  profile.SenderName,
			})
		}
	}

	// Set business profile on handler
	if profile != nil {
		p.coldEmailHandler.SetBusinessProfile(profile)
		defer p.coldEmailHandler.ClearBusinessProfile()
	}

	for i, leadID := range leadIDs {
		results[i] = p.generateEmailForLead(ctx, leadID, profile)

		// Update progress
		succeeded := 0
		failed := 0
		for _, r := range results[:i+1] {
			if r.Success {
				succeeded++
			} else if r.Error != "" {
				failed++
			}
		}
		p.supabase.UpdateAutomationTaskStatus(taskID, string(dto.TaskStatusProcessing), len(leadIDs), succeeded, failed, nil)
		automationLog.Debug("Email generation progress", map[string]interface{}{
			"task_id":   taskID,
			"processed": i + 1,
			"total":     len(leadIDs),
			"percent":   float64(i+1) / float64(len(leadIDs)) * 100,
		})
	}

	return results
}

// generateEmailForLead generates a cold email for a single lead
func (p *AutomationProcessor) generateEmailForLead(ctx context.Context, leadID string, profile *dto.BusinessProfile) dto.EnrichmentResult {
	result := dto.EnrichmentResult{LeadID: leadID}

	// Check if lead already has an email (prevent duplicates)
	hasEmail, err := p.supabase.LeadHasEmail(leadID)
	if err != nil {
		automationLog.Warn("Could not check for existing email, proceeding anyway", map[string]interface{}{
			"lead_id": leadID,
			"error":   err.Error(),
		})
	} else if hasEmail {
		automationLog.Info("Lead already has email - skipping generation", map[string]interface{}{
			"lead_id": leadID,
		})
		result.Success = true
		result.Email = true
		return result
	}

	// Get lead data
	lead, err := p.supabase.GetLeadByID(leadID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get lead: %v", err)
		return result
	}

	// Get pre-call report if exists
	preCallContent, _ := p.supabase.GetPreCallReportForLead(leadID)

	// Build input for email generation
	orgResult := handlers.OrganicResult{
		Title: lead.CompanyName,
	}
	if lead.Website != nil {
		orgResult.Link = *lead.Website
	}

	// Add extracted data from lead
	orgResult.ExtractedData = &handlers.ExtractedData{
		Success:     true,
		Company:     lead.CompanyName,
		Contact:     lead.ContactName,
		ContactRole: lead.ContactRole,
		Emails:      lead.Emails,
		Phones:      lead.Phones,
		Address:     lead.Address,
		SocialMedia: lead.SocialMedia,
	}

	// If no pre-call content and we have extra_data, add it to scraped content
	if preCallContent == "" && lead.ExtraData != nil {
		orgResult.ScrapedContent = buildContentFromExtraData(lead)
		automationLog.Info("Using CNPJ data for email generation", map[string]interface{}{
			"lead_id":          leadID,
			"cnpj":             lead.ExtraData.CNPJ,
			"cnae_description": lead.ExtraData.CNAEDescription,
		})
	}

	input := handlers.EmailGenerationInput{
		Result:        orgResult,
		PreCallReport: preCallContent,
	}

	// Generate email with retry
	var email *handlers.ColdEmail
	emailStart := time.Now()
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			automationLog.Warn("Retrying email generation", map[string]interface{}{
				"lead_id": leadID,
				"attempt": attempt,
				"max":     MaxRetries,
			})
			time.Sleep(RetryDelay)
		}
		email = p.coldEmailHandler.GenerateEmail(ctx, input)
		if email.Success {
			break
		}
	}
	emailDuration := time.Since(emailStart)

	if !email.Success {
		automationLog.Error("Failed to generate email", map[string]interface{}{
			"lead_id":      leadID,
			"error":        email.Error,
			"duration_sec": emailDuration.Seconds(),
		})
		result.Error = fmt.Sprintf("failed to generate email: %s", email.Error)
		return result
	}

	// Get recipient email
	toEmail := ""
	if len(lead.Emails) > 0 {
		toEmail = lead.Emails[0]
	}

	// Save to database
	emailRecord := &dto.ColdEmailRecord{
		LeadID:  leadID,
		Subject: email.Subject,
		Body:    email.Body,
		ToEmail: toEmail,
	}
	if profile != nil {
		emailRecord.FromName = profile.SenderName
	}

	if _, err := p.supabase.InsertColdEmail(emailRecord); err != nil {
		result.Error = fmt.Sprintf("failed to save email: %v", err)
		return result
	}

	// Update lead status to email_gerado
	p.supabase.UpdateLeadStatus(leadID, "email_gerado")

	result.Success = true
	result.Email = true
	automationLog.Info("✓ Cold email generated", map[string]interface{}{
		"lead_id":      leadID,
		"company_name": lead.CompanyName,
		"to_email":     toEmail,
		"subject":      email.Subject,
		"duration_sec": emailDuration.Seconds(),
		"has_precall":  preCallContent != "",
	})
	return result
}

// processFullEnrichment does enrichment + pre-call + email in sequence
func (p *AutomationProcessor) processFullEnrichment(ctx context.Context, leadIDs []string, businessProfileID *string, taskID string) []dto.EnrichmentResult {
	results := make([]dto.EnrichmentResult, len(leadIDs))

	// Get business profile
	var profile *dto.BusinessProfile
	if businessProfileID != nil {
		var err error
		profile, err = p.supabase.GetBusinessProfile(*businessProfileID)
		if err != nil {
			automationLog.Warn("Could not get business profile for full enrichment", map[string]interface{}{
				"task_id":             taskID,
				"business_profile_id": *businessProfileID,
				"error":               err.Error(),
			})
		}
	}

	// Set on handlers
	if profile != nil {
		p.preCallReportHandler.SetBusinessProfile(profile)
		p.coldEmailHandler.SetBusinessProfile(profile)
		defer p.preCallReportHandler.ClearBusinessProfile()
		defer p.coldEmailHandler.ClearBusinessProfile()
	}

	// Process with semaphore for scraping
	sem := make(chan struct{}, MaxConcurrentScrapes)
	var wg sync.WaitGroup
	var mu sync.Mutex
	processedCount := 0

	for i, leadID := range leadIDs {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := dto.EnrichmentResult{LeadID: id}

			// Step 1: Enrich (optional - continues even if no website)
			enrichResult := p.enrichSingleLead(ctx, id)
			if enrichResult.Success {
				result.Enriched = true
			} else {
				automationLog.Warn("Enrichment skipped, continuing with pre-call/email", map[string]interface{}{
					"lead_id": id,
					"reason":  enrichResult.Error,
				})
			}

			// Step 2: Pre-call (runs even without enrichment)
			preCallResult := p.generatePreCallForLead(ctx, id)
			if preCallResult.Success {
				result.PreCall = true
			}

			// Step 3: Email (runs even without enrichment)
			emailResult := p.generateEmailForLead(ctx, id, profile)
			if emailResult.Success {
				result.Email = true
			}

			// Success if at least pre-call or email was generated
			result.Success = result.PreCall || result.Email

			mu.Lock()
			results[idx] = result
			processedCount++

			// Update progress
			succeeded := 0
			failed := 0
			for _, r := range results {
				if r.Success {
					succeeded++
				} else if r.Error != "" {
					failed++
				}
			}
			p.supabase.UpdateAutomationTaskStatus(taskID, string(dto.TaskStatusProcessing), len(leadIDs), succeeded, failed, nil)
			mu.Unlock()

			automationLog.Debug("Full enrichment progress", map[string]interface{}{
				"task_id":   taskID,
				"processed": processedCount,
				"total":     len(leadIDs),
				"percent":   float64(processedCount) / float64(len(leadIDs)) * 100,
			})
		}(i, leadID)
	}

	wg.Wait()
	return results
}

// ProcessLeadCreated handles auto-enrichment when a new lead is created
func (p *AutomationProcessor) ProcessLeadCreated(ctx context.Context, lead *dto.Lead) {
	startTime := time.Now()

	automationLog.Info("───────────────────────────────────────────────────────────", nil)
	automationLog.Info("AUTO-ENRICHMENT TRIGGERED", map[string]interface{}{
		"lead_id":      lead.ID,
		"user_id":      lead.UserID,
		"company_name": lead.CompanyName,
		"website":      lead.Website,
		"triggered_at": startTime.Format(time.RFC3339),
	})

	// Get user's automation config
	config, err := p.supabase.GetAutomationConfig(lead.UserID)
	if err != nil {
		automationLog.Info("No automation config found for user - skipping", map[string]interface{}{
			"user_id": lead.UserID,
			"lead_id": lead.ID,
			"reason":  err.Error(),
		})
		return
	}

	automationLog.Info("User automation config loaded", map[string]interface{}{
		"user_id":            lead.UserID,
		"auto_enrich":        config.AutoEnrichNewLeads,
		"auto_precall":       config.AutoGeneratePreCall,
		"auto_email":         config.AutoGenerateEmail,
		"default_profile_id": config.DefaultBusinessProfileID,
		"daily_limit":        config.DailyAutomationLimit,
	})

	// Check if any automation is enabled
	if !config.AutoEnrichNewLeads && !config.AutoGeneratePreCall && !config.AutoGenerateEmail {
		automationLog.Info("All automations disabled for user - skipping", map[string]interface{}{
			"user_id": lead.UserID,
			"lead_id": lead.ID,
		})
		return
	}

	// Determine task type based on config
	var taskType dto.TaskType
	if config.AutoEnrichNewLeads && config.AutoGeneratePreCall && config.AutoGenerateEmail {
		taskType = dto.TaskTypeFullEnrichment
	} else if config.AutoEnrichNewLeads {
		taskType = dto.TaskTypeLeadEnrichment
	} else if config.AutoGeneratePreCall {
		taskType = dto.TaskTypePreCallGeneration
	} else if config.AutoGenerateEmail {
		taskType = dto.TaskTypeEmailGeneration
	}

	automationLog.Info("Starting auto-processing", map[string]interface{}{
		"lead_id":    lead.ID,
		"user_id":    lead.UserID,
		"task_type":  taskType,
		"profile_id": config.DefaultBusinessProfileID,
	})

	// Create task ID for logging (processing inline for single leads)
	taskID := fmt.Sprintf("auto-%s-%d", lead.ID, time.Now().UnixNano())
	automationLog.Debug("Created inline task", map[string]interface{}{
		"task_id":   taskID,
		"lead_id":   lead.ID,
		"task_type": taskType,
	})

	// Note: We process inline for single leads, no need to persist task to DB
	_ = &dto.AutomationTask{
		ID:                taskID,
		UserID:            lead.UserID,
		TaskType:          taskType,
		LeadID:            &lead.ID,
		BusinessProfileID: config.DefaultBusinessProfileID,
		Priority:          dto.TaskPriorityMedium,
		Status:            dto.TaskStatusPending,
		ItemsTotal:        1,
		MaxRetries:        MaxRetries,
	}

	// Process based on task type (inline for single lead)
	switch taskType {
	case dto.TaskTypeLeadEnrichment:
		p.enrichSingleLead(ctx, lead.ID)
	case dto.TaskTypePreCallGeneration:
		if config.DefaultBusinessProfileID != nil {
			profile, _ := p.supabase.GetBusinessProfile(*config.DefaultBusinessProfileID)
			if profile != nil {
				p.preCallReportHandler.SetBusinessProfile(profile)
				defer p.preCallReportHandler.ClearBusinessProfile()
			}
		}
		p.generatePreCallForLead(ctx, lead.ID)
	case dto.TaskTypeEmailGeneration:
		var profile *dto.BusinessProfile
		if config.DefaultBusinessProfileID != nil {
			profile, _ = p.supabase.GetBusinessProfile(*config.DefaultBusinessProfileID)
			if profile != nil {
				p.coldEmailHandler.SetBusinessProfile(profile)
				defer p.coldEmailHandler.ClearBusinessProfile()
			}
		}
		p.generateEmailForLead(ctx, lead.ID, profile)
	case dto.TaskTypeFullEnrichment:
		var profile *dto.BusinessProfile
		if config.DefaultBusinessProfileID != nil {
			profile, _ = p.supabase.GetBusinessProfile(*config.DefaultBusinessProfileID)
			if profile != nil {
				p.preCallReportHandler.SetBusinessProfile(profile)
				p.coldEmailHandler.SetBusinessProfile(profile)
				defer p.preCallReportHandler.ClearBusinessProfile()
				defer p.coldEmailHandler.ClearBusinessProfile()
			}
		}

		// Full pipeline (continues even if enrichment fails)
		enrichResult := p.enrichSingleLead(ctx, lead.ID)
		if !enrichResult.Success {
			automationLog.Warn("Enrichment skipped, continuing with pre-call/email", map[string]interface{}{
				"lead_id": lead.ID,
				"reason":  enrichResult.Error,
			})
		}
		// Generate pre-call and email regardless of enrichment result
		p.generatePreCallForLead(ctx, lead.ID)
		p.generateEmailForLead(ctx, lead.ID, profile)
	}

	duration := time.Since(startTime)
	automationLog.Info("AUTO-ENRICHMENT COMPLETED", map[string]interface{}{
		"lead_id":      lead.ID,
		"user_id":      lead.UserID,
		"task_type":    taskType,
		"duration_sec": duration.Seconds(),
	})
	automationLog.Info("───────────────────────────────────────────────────────────", nil)
}

// buildContentFromExtraData creates rich content from CNPJ import data for AI processing
func buildContentFromExtraData(lead *dto.Lead) string {
	if lead.ExtraData == nil {
		return ""
	}

	extra := lead.ExtraData
	var content strings.Builder

	content.WriteString("# Informações da Empresa (Dados da Receita Federal)\n\n")

	// Company identification
	if extra.RazaoSocial != "" {
		content.WriteString(fmt.Sprintf("**Razão Social**: %s\n", extra.RazaoSocial))
	}
	if extra.NomeFantasia != "" {
		content.WriteString(fmt.Sprintf("**Nome Fantasia**: %s\n", extra.NomeFantasia))
	}
	if extra.CNPJ != "" {
		content.WriteString(fmt.Sprintf("**CNPJ**: %s\n", extra.CNPJ))
	}

	// Business activity
	content.WriteString("\n## Atividade Principal\n")
	if extra.CNAEDescription != "" {
		content.WriteString(fmt.Sprintf("**Descrição**: %s\n", extra.CNAEDescription))
	}
	if extra.CNAECode != "" {
		content.WriteString(fmt.Sprintf("**Código CNAE**: %s\n", extra.CNAECode))
	}

	// Secondary activities
	if extra.SecondaryActivities != nil {
		if descriptions, ok := extra.SecondaryActivities["descriptions"].([]interface{}); ok && len(descriptions) > 0 {
			content.WriteString("\n## Atividades Secundárias\n")
			for _, desc := range descriptions {
				if str, ok := desc.(string); ok {
					content.WriteString(fmt.Sprintf("- %s\n", str))
				}
			}
		}
	}

	// Company details
	content.WriteString("\n## Dados da Empresa\n")
	if extra.LegalNature != "" {
		content.WriteString(fmt.Sprintf("**Natureza Jurídica**: %s\n", extra.LegalNature))
	}
	if extra.CompanySize != "" {
		content.WriteString(fmt.Sprintf("**Porte**: %s\n", extra.CompanySize))
	}
	if extra.Capital != "" && extra.Capital != "0" {
		content.WriteString(fmt.Sprintf("**Capital Social**: R$ %s\n", extra.Capital))
	}
	if extra.FoundedAt != "" {
		content.WriteString(fmt.Sprintf("**Data de Fundação**: %s\n", extra.FoundedAt))
	}
	if extra.Status != "" {
		content.WriteString(fmt.Sprintf("**Situação Cadastral**: %s\n", extra.Status))
	}

	// Tax regime
	if extra.SimplesOptante {
		content.WriteString("**Optante pelo Simples**: Sim\n")
	}
	if extra.MEIOptante {
		content.WriteString("**MEI**: Sim\n")
	}

	// Partners/Owners
	if len(extra.Partners) > 0 {
		content.WriteString("\n## Sócios/Proprietários\n")
		for _, partner := range extra.Partners {
			content.WriteString(fmt.Sprintf("- %s\n", partner))
		}
	}

	// Contact info from lead
	content.WriteString("\n## Informações de Contato\n")
	if lead.ContactName != "" && lead.ContactName != "Não informado" {
		content.WriteString(fmt.Sprintf("**Contato**: %s", lead.ContactName))
		if lead.ContactRole != "" {
			content.WriteString(fmt.Sprintf(" (%s)", lead.ContactRole))
		}
		content.WriteString("\n")
	}
	if len(lead.Emails) > 0 {
		content.WriteString(fmt.Sprintf("**E-mails**: %s\n", strings.Join(lead.Emails, ", ")))
	}
	if len(lead.Phones) > 0 {
		content.WriteString(fmt.Sprintf("**Telefones**: %s\n", strings.Join(lead.Phones, ", ")))
	}
	if lead.Address != "" {
		content.WriteString(fmt.Sprintf("**Endereço**: %s\n", lead.Address))
	}

	return content.String()
}
