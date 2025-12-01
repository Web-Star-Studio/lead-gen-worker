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

// ProcessJob processes a job in the background
// This function is meant to be called as a goroutine
func (p *JobProcessor) ProcessJob(ctx context.Context, job *dto.Job) {
	log.Printf("[JobProcessor] Starting job processing: id=%s, icp_name=%s", job.ID, job.ICPName)

	// 1. Update status to "processing"
	if err := p.supabase.UpdateJobStatus(job.ID, "processing", nil, nil); err != nil {
		log.Printf("[JobProcessor] Failed to update job status to processing: %v", err)
		p.failJob(job.ID, fmt.Sprintf("Failed to update status: %v", err))
		return
	}

	// 2. Fetch Business Profile if business_profile is provided
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

	// 3. Build search query from ICP
	searchQuery := p.buildSearchQuery(job, icp)
	log.Printf("[JobProcessor] Search query: %s", searchQuery)

	// 4. Build search request
	searchRequest := handlers.GoogleSearchParams{
		Q:              searchQuery,
		Location:       job.Region,
		Hl:             "pt-br",
		Gl:             "br",
		ExcludeDomains: job.ExcludedDomains,
		Num:            job.LeadQuantity,
	}

	// 5. Execute search
	log.Printf("[JobProcessor] Executing search with num=%d", searchRequest.Num)
	searchResult, err := p.searchHandler.Search(searchRequest)
	if err != nil {
		log.Printf("[JobProcessor] Search failed: %v", err)
		p.failJob(job.ID, fmt.Sprintf("Search failed: %v", err))
		return
	}

	log.Printf("[JobProcessor] Search returned %d results", len(searchResult.OrganicResults))

	// 6. Process each result, filter by required fields, and save leads
	leadsGenerated := 0
	for _, result := range searchResult.OrganicResults {
		// Check if result has extracted data
		if result.ExtractedData == nil {
			log.Printf("[JobProcessor] Skipping result without extracted data: %s", result.Link)
			continue
		}

		// Check required fields
		if !p.meetsRequiredFields(result.ExtractedData, job.RequiredFields) {
			log.Printf("[JobProcessor] Result does not meet required fields: %s", result.Link)
			continue
		}

		// Create lead
		lead := p.createLead(job, &result)
		leadID, err := p.supabase.InsertLead(lead)
		if err != nil {
			log.Printf("[JobProcessor] Failed to insert lead: %v", err)
			continue
		}

		// Insert pre-call report if available
		if result.PreCallReport != "" {
			if err := p.supabase.InsertPreCallReport(leadID, result.PreCallReport); err != nil {
				log.Printf("[JobProcessor] Failed to insert pre-call report: %v", err)
				// Continue anyway, lead was created
			}
		}

		// Insert cold email if available
		if result.ColdEmail != nil && result.ColdEmail.Success {
			// Get recipient email from extracted data
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

			// Set from_name from business profile if available
			if businessProfile != nil && businessProfile.SenderName != "" {
				coldEmailRecord.FromName = businessProfile.SenderName
			}

			if _, err := p.supabase.InsertColdEmail(coldEmailRecord); err != nil {
				log.Printf("[JobProcessor] Failed to insert cold email: %v", err)
				// Continue anyway, lead was created
			}
		}

		leadsGenerated++
		log.Printf("[JobProcessor] Lead created: id=%s, company=%s", leadID, lead.CompanyName)
	}

	// 7. Update job to completed
	if err := p.supabase.UpdateJobStatus(job.ID, "completed", &leadsGenerated, nil); err != nil {
		log.Printf("[JobProcessor] Failed to update job status to completed: %v", err)
		return
	}

	log.Printf("[JobProcessor] Job completed: id=%s, leads_generated=%d", job.ID, leadsGenerated)
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
		case "contact":
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
