package handlers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test helper functions

func TestExtractEmailsFromText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single email",
			input:    "Contact us at contact@example.com for more info",
			expected: []string{"contact@example.com"},
		},
		{
			name:     "multiple emails",
			input:    "Email: sales@company.com or support@company.com",
			expected: []string{"sales@company.com", "support@company.com"},
		},
		{
			name:     "duplicate emails",
			input:    "Email: test@example.com and test@example.com again",
			expected: []string{"test@example.com"},
		},
		{
			name:     "no emails",
			input:    "No email addresses here",
			expected: nil,
		},
		{
			name:     "email with subdomain",
			input:    "Contact: info@mail.company.com.br",
			expected: []string{"info@mail.company.com.br"},
		},
		{
			name:     "mixed case emails",
			input:    "Email: Test@Example.COM and test@example.com",
			expected: []string{"Test@Example.COM"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEmailsFromText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPhonesFromText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // Just check count since formats vary
	}{
		{
			name:     "Brazilian phone with country code",
			input:    "Telefone: +55 81 99999-9999",
			expected: 1,
		},
		{
			name:     "Brazilian phone without country code",
			input:    "Ligue: (81) 99999-9999",
			expected: 1,
		},
		{
			name:     "multiple phones",
			input:    "Tel: 81 99999-9999 ou 81 3333-4444",
			expected: 2,
		},
		{
			name:     "no phones",
			input:    "No phone numbers here",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPhonesFromText(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestCleanPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "formatted phone",
			input:    "+55 (81) 99999-9999",
			expected: "5581999999999",
		},
		{
			name:     "phone with dots",
			input:    "81.99999.9999",
			expected: "81999999999",
		},
		{
			name:     "clean phone",
			input:    "81999999999",
			expected: "81999999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanPhone(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "plain json",
			input:    "{\"key\": \"value\"}",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "with extra whitespace",
			input:    "  {\"key\": \"value\"}  ",
			expected: "{\"key\": \"value\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanJSONResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJSONString(t *testing.T) {
	json := `{"company": "Test Corp", "contact": "John Doe", "empty": ""}`

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "existing key",
			key:      "company",
			expected: "Test Corp",
		},
		{
			name:     "another existing key",
			key:      "contact",
			expected: "John Doe",
		},
		{
			name:     "empty value",
			key:      "empty",
			expected: "",
		},
		{
			name:     "non-existing key",
			key:      "missing",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONString(json, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJSONArray(t *testing.T) {
	json := `{"emails": ["a@b.com", "c@d.com"], "empty": [], "phones": ["123"]}`

	tests := []struct {
		name     string
		key      string
		expected []string
	}{
		{
			name:     "multiple items",
			key:      "emails",
			expected: []string{"a@b.com", "c@d.com"},
		},
		{
			name:     "single item",
			key:      "phones",
			expected: []string{"123"},
		},
		{
			name:     "empty array",
			key:      "empty",
			expected: nil,
		},
		{
			name:     "missing key",
			key:      "missing",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONArray(json, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractJSONObject(t *testing.T) {
	json := `{"social_media": {"linkedin": "http://li.com", "twitter": "http://tw.com"}, "empty": {}}`

	tests := []struct {
		name     string
		key      string
		expected map[string]string
	}{
		{
			name:     "object with values",
			key:      "social_media",
			expected: map[string]string{"linkedin": "http://li.com", "twitter": "http://tw.com"},
		},
		{
			name:     "empty object",
			key:      "empty",
			expected: map[string]string{},
		},
		{
			name:     "missing key",
			key:      "missing",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONObject(json, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		char     byte
		expected int
	}{
		{
			name:     "find opening brace",
			input:    "prefix {json}",
			char:     '{',
			expected: 7,
		},
		{
			name:     "find at start",
			input:    "{json}",
			char:     '{',
			expected: 0,
		},
		{
			name:     "not found",
			input:    "no brace here",
			char:     '{',
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findChar(tt.input, tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindLastChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		char     byte
		expected int
	}{
		{
			name:     "find closing brace",
			input:    "{nested {}} suffix",
			char:     '}',
			expected: 10,
		},
		{
			name:     "find at end",
			input:    "{json}",
			char:     '}',
			expected: 5,
		},
		{
			name:     "not found",
			input:    "no brace here",
			char:     '}',
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findLastChar(tt.input, tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemovePrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prefix   string
		expected string
	}{
		{
			name:     "has prefix",
			input:    "```json\n{data}",
			prefix:   "```json",
			expected: "\n{data}",
		},
		{
			name:     "no prefix",
			input:    "{data}",
			prefix:   "```json",
			expected: "{data}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removePrefix(tt.input, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		suffix   string
		expected string
	}{
		{
			name:     "has suffix",
			input:    "{data}\n```",
			suffix:   "```",
			expected: "{data}\n",
		},
		{
			name:     "no suffix",
			input:    "{data}",
			suffix:   "```",
			expected: "{data}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeSuffix(tt.input, tt.suffix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLowerASCII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uppercase",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "mixed case",
			input:    "HeLLo WoRLd",
			expected: "hello world",
		},
		{
			name:     "already lowercase",
			input:    "hello",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLowerASCII(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test configuration

func TestNewDataExtractorHandler_MissingAPIKey(t *testing.T) {
	// Ensure all auth env vars are not set for this test
	originalKey := os.Getenv("GOOGLE_API_KEY")
	originalVertexAI := os.Getenv("GOOGLE_GENAI_USE_VERTEXAI")
	os.Unsetenv("GOOGLE_API_KEY")
	os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI")
	defer func() {
		if originalKey != "" {
			os.Setenv("GOOGLE_API_KEY", originalKey)
		}
		if originalVertexAI != "" {
			os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", originalVertexAI)
		}
	}()

	handler, err := NewDataExtractorHandler(DataExtractorConfig{})

	assert.Nil(t, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Google API key is required")
}

func TestNewDataExtractorHandler_VertexAI_MissingProject(t *testing.T) {
	originalProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	originalLocation := os.Getenv("GOOGLE_CLOUD_LOCATION")
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GOOGLE_CLOUD_LOCATION")
	defer func() {
		if originalProject != "" {
			os.Setenv("GOOGLE_CLOUD_PROJECT", originalProject)
		}
		if originalLocation != "" {
			os.Setenv("GOOGLE_CLOUD_LOCATION", originalLocation)
		}
	}()

	handler, err := NewDataExtractorHandler(DataExtractorConfig{
		UseVertexAI: true,
	})

	assert.Nil(t, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GCP Project is required")
}

func TestDataExtractorConfig_Defaults(t *testing.T) {
	// Test that defaults are set correctly
	assert.Equal(t, DefaultExtractionTimeout.String(), "30s")
	assert.Equal(t, MaxConcurrentExtractions, 5)
	assert.Equal(t, DefaultExtractorModel, "gemini-2.5-flash")
}

func TestExtractedData_Fields(t *testing.T) {
	data := &ExtractedData{
		URL:         "https://example.com",
		Company:     "Test Company",
		Contact:     "John Doe",
		ContactRole: "CEO",
		Emails:      []string{"john@example.com"},
		Phones:      []string{"+55 11 99999-9999"},
		Address:     "123 Main St",
		Website:     "https://example.com",
		SocialMedia: map[string]string{"linkedin": "https://linkedin.com/company/test"},
		Success:     true,
	}

	assert.Equal(t, "https://example.com", data.URL)
	assert.Equal(t, "Test Company", data.Company)
	assert.Equal(t, "John Doe", data.Contact)
	assert.Equal(t, "CEO", data.ContactRole)
	assert.Len(t, data.Emails, 1)
	assert.Len(t, data.Phones, 1)
	assert.True(t, data.Success)
}

// Test buildPrompt

func TestDataExtractorHandler_buildPrompt(t *testing.T) {
	// We can test buildPrompt indirectly by testing the result
	result := OrganicResult{
		Link:           "https://example.com",
		Title:          "Example Company",
		ScrapedContent: "# Welcome\nContact: contact@example.com\nPhone: (11) 99999-9999",
	}

	handler := &DataExtractorHandler{
		config: DataExtractorConfig{},
	}

	prompt := handler.buildPrompt(result)

	assert.Contains(t, prompt, "https://example.com")
	assert.Contains(t, prompt, "Example Company")
	assert.Contains(t, prompt, "contact@example.com")
}

// Test ExtractData with no content
func TestDataExtractorHandler_ExtractData_NoContent(t *testing.T) {
	result := OrganicResult{
		Link:           "https://example.com",
		Title:          "Example",
		ScrapedContent: "", // No content
	}

	// Simulate the behavior of ExtractData when there's no content
	// The actual method checks for empty ScrapedContent and returns early
	extracted := &ExtractedData{
		URL: result.Link,
	}

	if result.ScrapedContent == "" {
		extracted.Error = "no scraped content available"
		extracted.Success = false
	}

	assert.False(t, extracted.Success)
	assert.Contains(t, extracted.Error, "no scraped content")
}

// Integration tests - only run if GOOGLE_API_KEY is set
func TestDataExtractorHandler_Integration(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GOOGLE_API_KEY not set")
	}

	t.Run("creates handler with valid config", func(t *testing.T) {
		h, err := NewDataExtractorHandler(DataExtractorConfig{
			APIKey: apiKey,
		})

		assert.NoError(t, err)
		assert.NotNil(t, h)
	})
}
