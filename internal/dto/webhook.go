package dto

import (
	"time"
)

// Job represents a job record from the jobs table
type Job struct {
	ID                string     `json:"job_id"`
	UserID            string     `json:"user_id"`
	Status            string     `json:"status,omitempty"` // pending, processing, completed, failed
	ICPID             *string    `json:"icp_id,omitempty"`
	ICPName           string     `json:"icp_name"`
	Region            string     `json:"region"`
	LeadQuantity      int        `json:"lead_quantity"`
	ExcludedDomains   []string   `json:"excluded_domains"`
	RequiredFields    []string   `json:"required_fields"`
	BusinessProfileID *string    `json:"business_profile,omitempty"` // ID of the business profile to use for personalization
	LeadsGenerated    int        `json:"leads_generated,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	CreatedAt         time.Time  `json:"created_at,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// BusinessProfile represents a business profile for personalizing pre-call reports
type BusinessProfile struct {
	ID                 string   `json:"id"`
	UserID             string   `json:"user_id"`
	CompanyName        string   `json:"company_name"`
	CompanyDescription string   `json:"company_description,omitempty"`
	ProblemSolved      string   `json:"problem_solved,omitempty"`
	Differentials      []string `json:"differentials,omitempty"`
	SuccessCase        string   `json:"success_case,omitempty"`
	CommunicationTone  string   `json:"communication_tone,omitempty"`
	SenderName         string   `json:"sender_name,omitempty"`
}

// ICP represents an Ideal Customer Profile record from the icps table
type ICP struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Niche     string    `json:"niche"`
	Region    string    `json:"region"`
	Keywords  []string  `json:"keywords"`
	CreatedAt time.Time `json:"created_at"`
}

// Lead represents a lead record for insertion into the leads table
type Lead struct {
	ID          string            `json:"id,omitempty"`
	JobID       string            `json:"job_id"`
	UserID      string            `json:"user_id"`
	CompanyName string            `json:"company_name"`
	ContactName string            `json:"contact_name"`
	ContactRole string            `json:"contact_role,omitempty"`
	Emails      []string          `json:"emails,omitempty"`
	Phones      []string          `json:"phones,omitempty"`
	Website     *string           `json:"website,omitempty"`
	Address     string            `json:"address,omitempty"`
	SocialMedia map[string]string `json:"social_media,omitempty"`
	Source      string            `json:"source"` // Always "Google"
}

// PreCallReportRecord represents a pre-call report record for insertion
type PreCallReportRecord struct {
	ID        string    `json:"id,omitempty"`
	LeadID    string    `json:"lead_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// JobStatusUpdate represents the fields to update when changing job status
type JobStatusUpdate struct {
	Status         string     `json:"status"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	LeadsGenerated *int       `json:"leads_generated,omitempty"`
	ErrorMessage   *string    `json:"error_message,omitempty"`
}
