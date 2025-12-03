package services

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("[AutomationProcessor] Processing task: id=%s, type=%s, priority=%d",
		task.ID, task.TaskType, task.Priority)

	// Update status to processing
	if err := p.supabase.UpdateAutomationTaskStatus(task.ID, string(dto.TaskStatusProcessing), 0, 0, 0, nil); err != nil {
		log.Printf("[AutomationProcessor] Failed to update task status: %v", err)
		return
	}

	// Collect lead IDs to process
	var leadIDs []string
	if task.LeadID != nil {
		leadIDs = append(leadIDs, *task.LeadID)
	}
	leadIDs = append(leadIDs, task.LeadIDs...)

	if len(leadIDs) == 0 {
		errMsg := "no leads to process"
		p.supabase.UpdateAutomationTaskStatus(task.ID, string(dto.TaskStatusFailed), 0, 0, 0, &errMsg)
		return
	}

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

	p.supabase.UpdateAutomationTaskStatus(task.ID, string(status), len(results), succeeded, failed, nil)
	log.Printf("[AutomationProcessor] Task completed: id=%s, succeeded=%d, failed=%d", task.ID, succeeded, failed)
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

			log.Printf("[AutomationProcessor] Enrichment progress: %d/%d", processedCount, len(leadIDs))
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
		result.Error = "lead has no website"
		return result
	}

	// Scrape website with retry
	var scraped *handlers.ScrapedPage
	var scrapeErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[AutomationProcessor] Retry %d for scraping lead %s", attempt, leadID)
			time.Sleep(RetryDelay)
		}
		scraped, scrapeErr = p.firecrawlHandler.ScrapeURL(*lead.Website)
		if scrapeErr == nil && scraped.Success {
			break
		}
	}

	if scrapeErr != nil || !scraped.Success {
		errMsg := "unknown error"
		if scrapeErr != nil {
			errMsg = scrapeErr.Error()
		} else if scraped != nil {
			errMsg = scraped.Error
		}
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
	log.Printf("[AutomationProcessor] ✓ Lead enriched: id=%s", leadID)
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
			log.Printf("[AutomationProcessor] Warning: could not get business profile: %v", err)
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
		log.Printf("[AutomationProcessor] Pre-call progress: %d/%d", i+1, len(leadIDs))
	}

	return results
}

// generatePreCallForLead generates a pre-call report for a single lead
func (p *AutomationProcessor) generatePreCallForLead(ctx context.Context, leadID string) dto.EnrichmentResult {
	result := dto.EnrichmentResult{LeadID: leadID}

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

	// Try to get scraped content if we have website
	if lead.Website != nil && *lead.Website != "" {
		scraped, err := p.firecrawlHandler.ScrapeURL(*lead.Website)
		if err == nil && scraped.Success {
			orgResult.ScrapedContent = scraped.Markdown
		}
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
	log.Printf("[AutomationProcessor] ✓ Pre-call generated: lead_id=%s", leadID)
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
			log.Printf("[AutomationProcessor] Warning: could not get business profile: %v", err)
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
		log.Printf("[AutomationProcessor] Email progress: %d/%d", i+1, len(leadIDs))
	}

	return results
}

// generateEmailForLead generates a cold email for a single lead
func (p *AutomationProcessor) generateEmailForLead(ctx context.Context, leadID string, profile *dto.BusinessProfile) dto.EnrichmentResult {
	result := dto.EnrichmentResult{LeadID: leadID}

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

	input := handlers.EmailGenerationInput{
		Result:        orgResult,
		PreCallReport: preCallContent,
	}

	// Generate email with retry
	var email *handlers.ColdEmail
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[AutomationProcessor] Retry %d for email lead %s", attempt, leadID)
			time.Sleep(RetryDelay)
		}
		email = p.coldEmailHandler.GenerateEmail(ctx, input)
		if email.Success {
			break
		}
	}

	if !email.Success {
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
	log.Printf("[AutomationProcessor] ✓ Email generated: lead_id=%s", leadID)
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
			log.Printf("[AutomationProcessor] Warning: could not get business profile: %v", err)
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

			// Step 1: Enrich
			enrichResult := p.enrichSingleLead(ctx, id)
			if !enrichResult.Success {
				result.Error = enrichResult.Error
				mu.Lock()
				results[idx] = result
				processedCount++
				mu.Unlock()
				return
			}
			result.Enriched = true

			// Step 2: Pre-call
			preCallResult := p.generatePreCallForLead(ctx, id)
			if preCallResult.Success {
				result.PreCall = true
			}

			// Step 3: Email
			emailResult := p.generateEmailForLead(ctx, id, profile)
			if emailResult.Success {
				result.Email = true
			}

			result.Success = result.Enriched && (result.PreCall || result.Email)

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

			log.Printf("[AutomationProcessor] Full enrichment progress: %d/%d", processedCount, len(leadIDs))
		}(i, leadID)
	}

	wg.Wait()
	return results
}

// ProcessLeadCreated handles auto-enrichment when a new lead is created
func (p *AutomationProcessor) ProcessLeadCreated(ctx context.Context, lead *dto.Lead) {
	log.Printf("[AutomationProcessor] Processing new lead: id=%s, company=%s", lead.ID, lead.CompanyName)

	// Get user's automation config
	config, err := p.supabase.GetAutomationConfig(lead.UserID)
	if err != nil {
		log.Printf("[AutomationProcessor] No automation config for user %s: %v", lead.UserID, err)
		return
	}

	// Check if any automation is enabled
	if !config.AutoEnrichNewLeads && !config.AutoGeneratePreCall && !config.AutoGenerateEmail {
		log.Printf("[AutomationProcessor] No automations enabled for user %s", lead.UserID)
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

	log.Printf("[AutomationProcessor] Auto-processing lead %s with task type: %s", lead.ID, taskType)

	// Create and process task inline (for single leads, no need to queue)
	task := &dto.AutomationTask{
		ID:                fmt.Sprintf("auto-%s-%d", lead.ID, time.Now().UnixNano()),
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

		// Full pipeline
		enrichResult := p.enrichSingleLead(ctx, lead.ID)
		if enrichResult.Success {
			p.generatePreCallForLead(ctx, lead.ID)
			p.generateEmailForLead(ctx, lead.ID, profile)
		}
	}

	log.Printf("[AutomationProcessor] Auto-processing completed for lead %s", task.ID)
}
