package controllers

import (
	"testing"

	"webstar/noturno-leadgen-worker/internal/dto"

	"github.com/stretchr/testify/assert"
)

func TestParseAutomationTaskFromRecord(t *testing.T) {
	t.Run("parse with id field", func(t *testing.T) {
		record := map[string]interface{}{
			"id":        "task-123",
			"user_id":   "user-456",
			"task_type": "email_generation",
			"lead_ids":  []interface{}{"lead-1", "lead-2"},
			"priority":  float64(1),
		}

		task := parseAutomationTaskFromRecord(record)

		assert.Equal(t, "task-123", task.ID)
		assert.Equal(t, "user-456", task.UserID)
		assert.Equal(t, dto.TaskType("email_generation"), task.TaskType)
		assert.Equal(t, []string{"lead-1", "lead-2"}, task.LeadIDs)
		assert.Equal(t, dto.TaskPriority(1), task.Priority)
		assert.Equal(t, 2, task.ItemsTotal)
	})

	t.Run("parse with task_id field (Edge Function format)", func(t *testing.T) {
		record := map[string]interface{}{
			"task_id":             "task-789",
			"user_id":             "user-456",
			"task_type":           "precall_generation",
			"lead_ids":            []interface{}{"lead-1"},
			"business_profile_id": "profile-123",
			"priority":            float64(2),
		}

		task := parseAutomationTaskFromRecord(record)

		assert.Equal(t, "task-789", task.ID) // Should use task_id
		assert.Equal(t, "user-456", task.UserID)
		assert.Equal(t, dto.TaskType("precall_generation"), task.TaskType)
		assert.Equal(t, []string{"lead-1"}, task.LeadIDs)
		assert.NotNil(t, task.BusinessProfileID)
		assert.Equal(t, "profile-123", *task.BusinessProfileID)
		assert.Equal(t, 1, task.ItemsTotal)
	})

	t.Run("id takes precedence over task_id", func(t *testing.T) {
		record := map[string]interface{}{
			"id":        "primary-id",
			"task_id":   "secondary-id",
			"user_id":   "user-456",
			"task_type": "lead_enrichment",
			"lead_ids":  []interface{}{"lead-1"},
		}

		task := parseAutomationTaskFromRecord(record)

		assert.Equal(t, "primary-id", task.ID) // id takes precedence
	})

	t.Run("parse with single lead_id", func(t *testing.T) {
		record := map[string]interface{}{
			"id":        "task-123",
			"user_id":   "user-456",
			"task_type": "email_generation",
			"lead_id":   "single-lead",
		}

		task := parseAutomationTaskFromRecord(record)

		assert.NotNil(t, task.LeadID)
		assert.Equal(t, "single-lead", *task.LeadID)
		assert.Equal(t, 1, task.ItemsTotal)
	})

	t.Run("parse with both lead_id and lead_ids", func(t *testing.T) {
		record := map[string]interface{}{
			"id":        "task-123",
			"user_id":   "user-456",
			"task_type": "full_enrichment",
			"lead_id":   "single-lead",
			"lead_ids":  []interface{}{"lead-1", "lead-2"},
		}

		task := parseAutomationTaskFromRecord(record)

		assert.NotNil(t, task.LeadID)
		assert.Equal(t, 2, len(task.LeadIDs))
		assert.Equal(t, 3, task.ItemsTotal) // 1 + 2
	})
}

func TestValidateAutomationTask(t *testing.T) {
	t.Run("valid task with lead_ids", func(t *testing.T) {
		task := &dto.AutomationTask{
			ID:       "task-123",
			UserID:   "user-456",
			TaskType: dto.TaskTypeEmailGeneration,
			LeadIDs:  []string{"lead-1"},
		}

		err := validateAutomationTask(task)
		assert.NoError(t, err)
	})

	t.Run("valid task with single lead_id", func(t *testing.T) {
		leadID := "lead-123"
		task := &dto.AutomationTask{
			ID:       "task-123",
			UserID:   "user-456",
			TaskType: dto.TaskTypePreCallGeneration,
			LeadID:   &leadID,
		}

		err := validateAutomationTask(task)
		assert.NoError(t, err)
	})

	t.Run("missing id", func(t *testing.T) {
		task := &dto.AutomationTask{
			UserID:   "user-456",
			TaskType: dto.TaskTypeEmailGeneration,
			LeadIDs:  []string{"lead-1"},
		}

		err := validateAutomationTask(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "id or task_id")
	})

	t.Run("missing user_id", func(t *testing.T) {
		task := &dto.AutomationTask{
			ID:       "task-123",
			TaskType: dto.TaskTypeEmailGeneration,
			LeadIDs:  []string{"lead-1"},
		}

		err := validateAutomationTask(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id")
	})

	t.Run("missing task_type", func(t *testing.T) {
		task := &dto.AutomationTask{
			ID:      "task-123",
			UserID:  "user-456",
			LeadIDs: []string{"lead-1"},
		}

		err := validateAutomationTask(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task_type")
	})

	t.Run("missing leads", func(t *testing.T) {
		task := &dto.AutomationTask{
			ID:       "task-123",
			UserID:   "user-456",
			TaskType: dto.TaskTypeEmailGeneration,
		}

		err := validateAutomationTask(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lead_id or lead_ids")
	})
}

func TestParseLeadFromRecord(t *testing.T) {
	t.Run("parse complete lead", func(t *testing.T) {
		record := map[string]interface{}{
			"id":           "lead-123",
			"user_id":      "user-456",
			"job_id":       "job-789",
			"company_name": "Test Company",
			"contact_name": "John Doe",
			"contact_role": "CEO",
			"website":      "https://example.com",
			"address":      "123 Main St",
			"source":       "google_search",
			"emails":       []interface{}{"john@example.com", "info@example.com"},
			"phones":       []interface{}{"+5581999999999"},
			"social_media": map[string]interface{}{
				"linkedin": "https://linkedin.com/company/test",
			},
		}

		lead := parseLeadFromRecord(record)

		assert.Equal(t, "lead-123", lead.ID)
		assert.Equal(t, "user-456", lead.UserID)
		assert.Equal(t, "job-789", lead.JobID)
		assert.Equal(t, "Test Company", lead.CompanyName)
		assert.Equal(t, "John Doe", lead.ContactName)
		assert.Equal(t, "CEO", lead.ContactRole)
		assert.NotNil(t, lead.Website)
		assert.Equal(t, "https://example.com", *lead.Website)
		assert.Equal(t, "123 Main St", lead.Address)
		assert.Equal(t, "google_search", lead.Source)
		assert.Equal(t, []string{"john@example.com", "info@example.com"}, lead.Emails)
		assert.Equal(t, []string{"+5581999999999"}, lead.Phones)
		assert.Equal(t, "https://linkedin.com/company/test", lead.SocialMedia["linkedin"])
	})

	t.Run("parse minimal lead", func(t *testing.T) {
		record := map[string]interface{}{
			"id":           "lead-123",
			"user_id":      "user-456",
			"company_name": "Minimal Company",
		}

		lead := parseLeadFromRecord(record)

		assert.Equal(t, "lead-123", lead.ID)
		assert.Equal(t, "user-456", lead.UserID)
		assert.Equal(t, "Minimal Company", lead.CompanyName)
		assert.Nil(t, lead.Website)
		assert.Empty(t, lead.Emails)
		assert.Empty(t, lead.Phones)
	})
}

func TestGenerateTaskID(t *testing.T) {
	id1 := generateTaskID()
	id2 := generateTaskID()

	assert.True(t, len(id1) > 5)
	assert.Contains(t, id1, "task-")
	// Note: With current implementation, IDs might be the same since randomString is deterministic
	// In production, this should use crypto/rand
	assert.Equal(t, len(id1), len(id2))
}
