package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestColdEmailConfig_Defaults(t *testing.T) {
	t.Run("default email model", func(t *testing.T) {
		assert.Equal(t, "gemini-2.5-flash", DefaultEmailModel)
	})

	t.Run("default fallback model", func(t *testing.T) {
		assert.Equal(t, "gemini-2.5-pro", DefaultFallbackModel)
	})

	t.Run("default timeout", func(t *testing.T) {
		assert.Equal(t, 45*time.Second, DefaultEmailTimeout)
	})

	t.Run("default max concurrent", func(t *testing.T) {
		assert.Equal(t, 3, MaxConcurrentEmails)
	})
}

func TestBuildEmailAgentInstruction(t *testing.T) {
	t.Run("without custom instruction", func(t *testing.T) {
		instruction := buildEmailAgentInstruction("")
		assert.Contains(t, instruction, "B2B copywriting and sales expert")
		assert.Contains(t, instruction, "MUST have at least 2 paragraphs")
		assert.Contains(t, instruction, "professional greeting")
		assert.Contains(t, instruction, "Olá!")
		assert.Contains(t, instruction, "Hi there!")
		assert.NotContains(t, instruction, "Additional Instructions")
	})

	t.Run("with custom instruction", func(t *testing.T) {
		custom := "Focus on SaaS companies"
		instruction := buildEmailAgentInstruction(custom)
		assert.Contains(t, instruction, "B2B copywriting and sales expert")
		assert.Contains(t, instruction, "Additional Instructions")
		assert.Contains(t, instruction, custom)
	})

	t.Run("greeting instructions", func(t *testing.T) {
		instruction := buildEmailAgentInstruction("")
		// Should NOT use direct names or time-based greetings
		assert.Contains(t, instruction, "does NOT use the recipient's name directly")
		assert.Contains(t, instruction, "Bom dia")
		assert.Contains(t, instruction, "Good morning")
	})

	t.Run("paragraph requirements", func(t *testing.T) {
		instruction := buildEmailAgentInstruction("")
		assert.Contains(t, instruction, "First paragraph")
		assert.Contains(t, instruction, "Second paragraph")
	})
}

func TestColdEmail_Structure(t *testing.T) {
	email := &ColdEmail{
		URL:              "https://example.com",
		RecipientName:    "João Silva",
		RecipientCompany: "Example Corp",
		Subject:          "Test Subject",
		Body:             "Test Body",
		PlainTextBody:    "Test Body",
		CallToAction:     "Schedule a call",
		Success:          true,
		GeneratedAt:      time.Now(),
	}

	assert.Equal(t, "https://example.com", email.URL)
	assert.Equal(t, "João Silva", email.RecipientName)
	assert.Equal(t, "Example Corp", email.RecipientCompany)
	assert.Equal(t, "Test Subject", email.Subject)
	assert.Equal(t, "Test Body", email.Body)
	assert.True(t, email.Success)
}

func TestParseEmailResponse(t *testing.T) {
	handler := &ColdEmailHandler{}

	t.Run("parse Portuguese response", func(t *testing.T) {
		response := `ASSUNTO: Teste de assunto

---

CORPO:
Olá! Tudo bem?

Este é o corpo do email.

---

CTA: Agendar uma conversa

---

NOTAS DE PERSONALIZAÇÃO: Email personalizado para a empresa`

		email := &ColdEmail{}
		handler.parseEmailResponse(response, email)

		assert.Equal(t, "Teste de assunto", email.Subject)
		assert.Contains(t, email.Body, "Olá!")
		assert.Equal(t, "Agendar uma conversa", email.CallToAction)
		assert.Contains(t, email.PersonalizationNotes, "Email personalizado")
	})

	t.Run("parse English response", func(t *testing.T) {
		response := `SUBJECT: Test subject line

---

BODY:
Hi there! Hope you're doing well.

This is the email body.

---

CTA: Schedule a quick call

---

PERSONALIZATION NOTES: Email personalized for the company`

		email := &ColdEmail{}
		handler.parseEmailResponse(response, email)

		assert.Equal(t, "Test subject line", email.Subject)
		assert.Contains(t, email.Body, "Hi there!")
		assert.Equal(t, "Schedule a quick call", email.CallToAction)
		assert.Contains(t, email.PersonalizationNotes, "Email personalized")
	})

	t.Run("parse empty response falls back to raw", func(t *testing.T) {
		response := "Just some random text without markers"
		email := &ColdEmail{}
		handler.parseEmailResponse(response, email)

		assert.Equal(t, response, email.Body)
		assert.Equal(t, "", email.Subject)
	})
}

func TestExtractEmailSection(t *testing.T) {
	tests := []struct {
		name     string
		response string
		section  string
		expected string
	}{
		{
			name:     "extract subject",
			response: "SUBJECT: My Subject Line\n\n---\n\nBODY: Content here",
			section:  "SUBJECT",
			expected: "My Subject Line",
		},
		{
			name:     "extract body",
			response: "SUBJECT: Title\n\n---\n\nBODY:\nLine 1\nLine 2\n\n---\n\nCTA: Action",
			section:  "BODY",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "case insensitive",
			response: "subject: lowercase title\n\n---",
			section:  "SUBJECT",
			expected: "lowercase title",
		},
		{
			name:     "with bold markers",
			response: "**ASSUNTO**: Título aqui\n\n---",
			section:  "ASSUNTO",
			expected: "Título aqui",
		},
		{
			name:     "not found",
			response: "Some content without markers",
			section:  "SUBJECT",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEmailSection(tt.response, tt.section)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsQuotaExceededError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "429 error",
			err:      &testError{msg: "Error 429: Rate limit exceeded"},
			expected: true,
		},
		{
			name:     "resource exhausted error",
			err:      &testError{msg: "RESOURCE_EXHAUSTED: quota exceeded"},
			expected: true,
		},
		{
			name:     "quota in message",
			err:      &testError{msg: "You exceeded your current quota"},
			expected: true,
		},
		{
			name:     "other error",
			err:      &testError{msg: "connection timeout"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isQuotaExceededError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestEmailGenerationInput_Validation(t *testing.T) {
	t.Run("input with extracted data", func(t *testing.T) {
		input := EmailGenerationInput{
			Result: OrganicResult{
				Link:  "https://example.com",
				Title: "Example Company",
				ExtractedData: &ExtractedData{
					Company: "Example Corp",
					Contact: "John Doe",
					Emails:  []string{"john@example.com"},
				},
			},
		}

		assert.NotNil(t, input.Result.ExtractedData)
		assert.Equal(t, "Example Corp", input.Result.ExtractedData.Company)
	})

	t.Run("input with pre-call report", func(t *testing.T) {
		input := EmailGenerationInput{
			Result: OrganicResult{
				Link:  "https://example.com",
				Title: "Example Company",
			},
			PreCallReport: "Detailed analysis of the company...",
		}

		assert.NotEmpty(t, input.PreCallReport)
	})
}
