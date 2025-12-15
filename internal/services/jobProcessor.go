package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"webstar/noturno-leadgen-worker/internal/dto"
	"webstar/noturno-leadgen-worker/internal/handlers"
)

// JobProcessor handles background job processing
type JobProcessor struct {
	supabase      *handlers.SupabaseHandler
	searchHandler *handlers.GoogleSearchHandler
}

// NewJobProcessor creates a new JobProcessor instance
func NewJobProcessor(supabase *handlers.SupabaseHandler, searchHandler *handlers.GoogleSearchHandler) *JobProcessor {
	return &JobProcessor{
		supabase:      supabase,
		searchHandler: searchHandler,
	}
}

// ProcessJob processes a job in the background using streaming mode
// Each result is saved immediately after being fully processed (scraped, extracted, report + email generated)
// This allows users to see leads appearing in real-time without waiting for all results to complete
func (p *JobProcessor) ProcessJob(ctx context.Context, job *dto.Job) {
	log.Printf("[JobProcessor] Starting job processing (streaming mode): id=%s, icp_name=%s", job.ID, job.ICPName)

	// 1. Update status to "processing"
	if err := p.supabase.UpdateJobStatus(job.ID, "processing", nil, nil); err != nil {
		log.Printf("[JobProcessor] Failed to update job status to processing: %v", err)
		p.failJob(job.ID, fmt.Sprintf("Failed to update status: %v", err))
		return
	}

	// 2. Set location for language detection (must be set before business profile for proper detection)
	if job.Region != "" {
		p.searchHandler.SetLocation(job.Region)
	}

	// 2.5. Set user context for usage tracking
	p.searchHandler.SetUserContext(job.UserID, &job.ID)
	defer p.searchHandler.ClearUserContext()

	// 3. Fetch Business Profile if business_profile is provided
	var businessProfile *dto.BusinessProfile
	if job.BusinessProfileID != nil && *job.BusinessProfileID != "" {
		var err error
		businessProfile, err = p.supabase.GetBusinessProfile(*job.BusinessProfileID)
		if err != nil {
			log.Printf("[JobProcessor] Warning: Failed to get BusinessProfile: %v (continuing without personalization)", err)
			// Don't fail the job, just continue without personalization
		} else {
			// Set business profile on the search handler's pre-call report handler
			p.searchHandler.SetBusinessProfile(businessProfile)
			defer p.searchHandler.ClearBusinessProfile() // Clear after processing
		}
	}

	// 3. Fetch ICP if icp_id is provided
	var icp *dto.ICP
	if job.ICPID != nil && *job.ICPID != "" {
		var err error
		icp, err = p.supabase.GetICP(*job.ICPID)
		if err != nil {
			log.Printf("[JobProcessor] Failed to get ICP: %v", err)
			p.failJob(job.ID, fmt.Sprintf("Failed to get ICP: %v", err))
			return
		}
	}

	// 4. Build search query from ICP
	searchQuery := p.buildSearchQuery(job, icp)
	log.Printf("[JobProcessor] Search query: %s", searchQuery)

	// 5. Build search request
	searchRequest := handlers.GoogleSearchParams{
		Q:              searchQuery,
		Location:       job.Region,
		Hl:             "pt-br",
		Gl:             "br",
		ExcludeDomains: job.ExcludedDomains,
		Num:            job.LeadQuantity,
	}

	// 6. Execute streaming search - each result is saved immediately after processing
	leadsGenerated := 0

	// Callback function that saves each result as it's completed
	saveResultCallback := func(result *handlers.OrganicResult, index int) bool {
		// Check if result has extracted data
		if result.ExtractedData == nil {
			log.Printf("[JobProcessor] Skipping result %d without extracted data: %s", index+1, result.Link)
			return true // Continue to next result
		}

		// Check required fields
		if !p.meetsRequiredFields(result.ExtractedData, job.RequiredFields) {
			log.Printf("[JobProcessor] Result %d does not meet required fields: %s", index+1, result.Link)
			return true // Continue to next result
		}

		// Create and save lead immediately
		lead := p.createLead(job, result)
		leadID, err := p.supabase.InsertLead(lead)
		if err != nil {
			log.Printf("[JobProcessor] Failed to insert lead %d: %v", index+1, err)
			return true // Continue to next result
		}

		// Insert pre-call report if available
		if result.PreCallReport != "" {
			if err := p.supabase.InsertPreCallReport(leadID, result.PreCallReport); err != nil {
				log.Printf("[JobProcessor] Failed to insert pre-call report for lead %d: %v", index+1, err)
				// Continue anyway, lead was created
			}
		}

		// Insert cold email if available
		if result.ColdEmail != nil && result.ColdEmail.Success {
			toEmail := ""
			if result.ExtractedData != nil && len(result.ExtractedData.Emails) > 0 {
				toEmail = result.ExtractedData.Emails[0]
			}

			coldEmailRecord := &dto.ColdEmailRecord{
				LeadID:            leadID,
				Subject:           result.ColdEmail.Subject,
				Body:              result.ColdEmail.Body,
				ToEmail:           toEmail,
				BusinessProfileID: job.BusinessProfileID,
			}

			if businessProfile != nil && businessProfile.SenderName != "" {
				coldEmailRecord.FromName = businessProfile.SenderName
			}

			if _, err := p.supabase.InsertColdEmail(coldEmailRecord); err != nil {
				log.Printf("[JobProcessor] Failed to insert cold email for lead %d: %v", index+1, err)
				// Continue anyway, lead was created
			}
		}

		leadsGenerated++
		log.Printf("[JobProcessor] ✓ Lead %d saved immediately: id=%s, company=%s", index+1, leadID, lead.CompanyName)

		// Update job with current lead count (real-time progress)
		_ = p.supabase.UpdateJobStatus(job.ID, "processing", &leadsGenerated, nil)

		return true // Continue processing
	}

	log.Printf("[JobProcessor] Starting streaming search with callback (num=%d)", searchRequest.Num)
	_, err := p.searchHandler.SearchWithStreaming(searchRequest, saveResultCallback)
	if err != nil {
		log.Printf("[JobProcessor] Search failed: %v", err)
		p.failJob(job.ID, fmt.Sprintf("Search failed: %v", err))
		return
	}

	// 7. Update job to completed
	if err := p.supabase.UpdateJobStatus(job.ID, "completed", &leadsGenerated, nil); err != nil {
		log.Printf("[JobProcessor] Failed to update job status to completed: %v", err)
		return
	}

	log.Printf("[JobProcessor] Job completed (streaming mode): id=%s, leads_generated=%d", job.ID, leadsGenerated)
}

// buildSearchQuery builds the search query from job and ICP data
func (p *JobProcessor) buildSearchQuery(job *dto.Job, icp *dto.ICP) string {
	var parts []string

	if icp != nil {
		// Use ICP niche
		if icp.Niche != "" {
			parts = append(parts, icp.Niche)
		}
		// Add keywords
		if len(icp.Keywords) > 0 {
			parts = append(parts, strings.Join(icp.Keywords, " "))
		}
		// Add ICP name if different from niche
		if icp.Name != "" && icp.Name != icp.Niche {
			parts = append(parts, icp.Name)
		}
	} else {
		// Fallback to job's icp_name if no ICP found
		if job.ICPName != "" {
			parts = append(parts, job.ICPName)
		}
	}

	return strings.Join(parts, " ")
}

// meetsRequiredFields checks if the extracted data meets all required fields
func (p *JobProcessor) meetsRequiredFields(data *handlers.ExtractedData, requiredFields []string) bool {
	if len(requiredFields) == 0 {
		return true // No required fields, always pass
	}

	for _, field := range requiredFields {
		switch strings.ToLower(field) {
		case "email", "emails":
			if len(data.Emails) == 0 {
				return false
			}
		case "phone", "phones":
			if len(data.Phones) == 0 {
				return false
			}
		case "contact", "name":
			if data.Contact == "" {
				return false
			}
		case "address":
			if data.Address == "" {
				return false
			}
		case "company":
			if data.Company == "" {
				return false
			}
		}
	}

	return true
}

// createLead creates a Lead DTO from search result
func (p *JobProcessor) createLead(job *dto.Job, result *handlers.OrganicResult) *dto.Lead {
	lead := &dto.Lead{
		JobID:       job.ID,
		UserID:      job.UserID,
		CompanyName: result.ExtractedData.Company,
		ContactName: result.ExtractedData.Contact,
		ContactRole: result.ExtractedData.ContactRole,
		Emails:      result.ExtractedData.Emails,
		Phones:      result.ExtractedData.Phones,
		Address:     result.ExtractedData.Address,
		SocialMedia: result.ExtractedData.SocialMedia,
		Source:      "Google",
	}

	// Set website
	if result.Link != "" {
		lead.Website = &result.Link
	}

	// Fallback company name to title if not extracted
	if lead.CompanyName == "" {
		lead.CompanyName = result.Title
	}

	return lead
}

// failJob marks a job as failed with an error message
func (p *JobProcessor) failJob(jobID string, errorMessage string) {
	log.Printf("[JobProcessor] Job failed: id=%s, error=%s", jobID, errorMessage)
	if err := p.supabase.UpdateJobStatus(jobID, "failed", nil, &errorMessage); err != nil {
		log.Printf("[JobProcessor] Failed to update job status to failed: %v", err)
	}
}
