package dto

import "time"

// TaskType represents the type of automation task
type TaskType string

const (
	TaskTypeLeadEnrichment    TaskType = "lead_enrichment"    // Scrape + Extract data
	TaskTypePreCallGeneration TaskType = "precall_generation" // Generate pre-call report
	TaskTypeEmailGeneration   TaskType = "email_generation"   // Generate cold email
	TaskTypeFullEnrichment    TaskType = "full_enrichment"    // All of the above
)

// TaskStatus represents the status of an automation task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

// TaskPriority represents the priority level of a task
type TaskPriority int

const (
	TaskPriorityHigh   TaskPriority = 1 // Lead search jobs
	TaskPriorityMedium TaskPriority = 2 // Auto-triggered enrichment
	TaskPriorityLow    TaskPriority = 3 // Manual batch operations
)

// AutomationConfig represents user automation settings
type AutomationConfig struct {
	ID                       string    `json:"id"`
	UserID                   string    `json:"user_id"`
	AutoEnrichNewLeads       bool      `json:"auto_enrich_new_leads"`       // Scrape website on lead create
	AutoGeneratePreCall      bool      `json:"auto_generate_precall"`       // Generate pre-call after enrich
	AutoGenerateEmail        bool      `json:"auto_generate_email"`         // Generate email after pre-call
	DefaultBusinessProfileID *string   `json:"default_business_profile_id"` // Default profile for automations
	DailyAutomationLimit     int       `json:"daily_automation_limit"`      // Max automations per day
	CreatedAt                time.Time `json:"created_at,omitempty"`
	UpdatedAt                time.Time `json:"updated_at,omitempty"`
}

// AutomationTask represents a task to be processed by the automation worker
type AutomationTask struct {
	ID                string       `json:"id"`
	UserID            string       `json:"user_id"`
	TaskType          TaskType     `json:"task_type"`
	LeadID            *string      `json:"lead_id,omitempty"`  // Single lead
	LeadIDs           []string     `json:"lead_ids,omitempty"` // Batch of leads
	BusinessProfileID *string      `json:"business_profile_id,omitempty"`
	Priority          TaskPriority `json:"priority"`
	Status            TaskStatus   `json:"status"`
	ItemsTotal        int          `json:"items_total"`
	ItemsProcessed    int          `json:"items_processed"`
	ItemsSucceeded    int          `json:"items_succeeded"`
	ItemsFailed       int          `json:"items_failed"`
	ErrorMessage      *string      `json:"error_message,omitempty"`
	RetryCount        int          `json:"retry_count"`
	MaxRetries        int          `json:"max_retries"`
	CreatedAt         time.Time    `json:"created_at,omitempty"`
	StartedAt         *time.Time   `json:"started_at,omitempty"`
	CompletedAt       *time.Time   `json:"completed_at,omitempty"`
}

// AutomationTaskCreate is used when creating a new automation task
type AutomationTaskCreate struct {
	UserID            string       `json:"user_id"`
	TaskType          TaskType     `json:"task_type"`
	LeadID            *string      `json:"lead_id,omitempty"`
	LeadIDs           []string     `json:"lead_ids,omitempty"`
	BusinessProfileID *string      `json:"business_profile_id,omitempty"`
	Priority          TaskPriority `json:"priority,omitempty"`
}

// LeadForEnrichment represents minimal lead data needed for enrichment
type LeadForEnrichment struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	CompanyName string            `json:"company_name"`
	Website     *string           `json:"website,omitempty"`
	Emails      []string          `json:"emails,omitempty"`
	ContactName string            `json:"contact_name,omitempty"`
	ContactRole string            `json:"contact_role,omitempty"`
	Phones      []string          `json:"phones,omitempty"`
	Address     string            `json:"address,omitempty"`
	SocialMedia map[string]string `json:"social_media,omitempty"`
}

// EnrichmentResult holds the result of enriching a single lead
type EnrichmentResult struct {
	LeadID   string `json:"lead_id"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Enriched bool   `json:"enriched"` // Data extraction done
	PreCall  bool   `json:"precall"`  // Pre-call report generated
	Email    bool   `json:"email"`    // Cold email generated
}
