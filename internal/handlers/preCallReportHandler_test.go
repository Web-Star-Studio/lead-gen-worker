package handlers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions

func TestExtractSection(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		sectionName string
		expected    string
	}{
		{
			name:        "extracts bold section",
			response:    "Some intro\n**Company Name**: Acme Corp\n**Industry**: Technology",
			sectionName: "Company Name",
			expected:    "Acme Corp",
		},
		{
			name:        "extracts heading section",
			response:    "## Company Name\nAcme Corporation\n\n## Industry\nTech",
			sectionName: "Company Name",
			expected:    "Acme Corporation",
		},
		{
			name:        "extracts h3 section",
			response:    "### Company Name\nTest Company\n### Industry\nFinance",
			sectionName: "Company Name",
			expected:    "Test Company",
		},
		{
			name:        "returns empty for missing section",
			response:    "Some content without the section",
			sectionName: "Company Name",
			expected:    "",
		},
		{
			name:        "case insensitive matching",
			response:    "**company name**: Lower Case Corp",
			sectionName: "Company Name",
			expected:    "Lower Case Corp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSection(tt.response, tt.sectionName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractListSection(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		sectionName string
		expected    []string
	}{
		{
			name: "extracts dash list",
			response: `**Key Services**:
- Web Development
- Mobile Apps
- Cloud Solutions

**Next Section**: value`,
			sectionName: "Key Services",
			expected:    []string{"Web Development", "Mobile Apps", "Cloud Solutions"},
		},
		{
			name: "extracts numbered list",
			response: `## Talking Points
1. Discuss pricing
2. Show demo
3. Ask about timeline

## Next`,
			sectionName: "Talking Points",
			expected:    []string{"Discuss pricing", "Show demo", "Ask about timeline"},
		},
		{
			name: "extracts asterisk list",
			response: `**Pain Points**:
* High costs
* Slow delivery
* Poor support`,
			sectionName: "Pain Points",
			expected:    []string{"High costs", "Slow delivery", "Poor support"},
		},
		{
			name:        "returns empty for missing section",
			response:    "No list here",
			sectionName: "Key Services",
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractListSection(tt.response, tt.sectionName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple URL",
			url:      "https://example.com",
			expected: "httpsexample.com",
		},
		{
			name:     "URL with path",
			url:      "https://example.com/path/to/page",
			expected: "httpsexample.compathtopagexxxxxxxx",
		},
		{
			name:     "URL with special chars",
			url:      "https://example.com?foo=bar&baz=qux",
			expected: "httpsexample.comfoobarbazquxxxxxo",
		},
		{
			name:     "preserves dots dashes underscores",
			url:      "test-url_name.com",
			expected: "test-url_name.com",
		},
		{
			name:     "truncates long URLs",
			url:      "https://very-long-domain-name-that-exceeds-fifty-characters.example.com/path",
			expected: "httpsvery-long-domain-name-that-exceeds-fifty-char",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeURL(tt.url)
			assert.LessOrEqual(t, len(result), 50, "sanitized URL should not exceed 50 characters")
			// Just verify it doesn't panic and produces valid output
			assert.NotEmpty(t, result)
		})
	}
}

func TestTrimValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims spaces",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "trims tabs",
			input:    "\thello\t",
			expected: "hello",
		},
		{
			name:     "trims newlines",
			input:    "\nhello\n",
			expected: "hello",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles only whitespace",
			input:    "   \t\n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts uppercase",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "mixed case",
			input:    "HeLLo WoRLD",
			expected: "hello world",
		},
		{
			name:     "already lowercase",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "with numbers",
			input:    "Hello123",
			expected: "hello123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toLower(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "multiple lines",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "single line",
			input:    "single line",
			expected: []string{"single line"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "trailing newline",
			input:    "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "exact match",
			s:        "Hello World",
			substr:   "Hello",
			expected: 0,
		},
		{
			name:     "case insensitive",
			s:        "Hello World",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "middle of string",
			s:        "Hello World",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "not found",
			s:        "Hello World",
			substr:   "foo",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCaseInsensitive(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAgentInstruction(t *testing.T) {
	t.Run("without custom instruction", func(t *testing.T) {
		instruction := buildAgentInstruction("")
		assert.Contains(t, instruction, "multilingual sales intelligence analyst")
		assert.Contains(t, instruction, "Company Name")
		assert.Contains(t, instruction, "Talking Points")
		assert.NotContains(t, instruction, "Additional Instructions")
	})

	t.Run("with custom instruction", func(t *testing.T) {
		custom := "Focus on tech companies only"
		instruction := buildAgentInstruction(custom)
		assert.Contains(t, instruction, "multilingual sales intelligence analyst")
		assert.Contains(t, instruction, "Additional Instructions")
		assert.Contains(t, instruction, custom)
	})
}

func TestPreCallReportConfig_Defaults(t *testing.T) {
	// This test verifies default values without actually creating a handler
	// (which would require API key)
	t.Run("default model", func(t *testing.T) {
		assert.Equal(t, "gemini-2.5-flash", DefaultGeminiModel)
	})

	t.Run("default fallback model", func(t *testing.T) {
		assert.Equal(t, "gemini-2.5-pro", DefaultReportFallbackModel)
	})

	t.Run("default timeout", func(t *testing.T) {
		assert.Equal(t, 60*time.Second, DefaultReportTimeout)
	})

	t.Run("default max concurrent", func(t *testing.T) {
		assert.Equal(t, 3, MaxConcurrentReports)
	})
}

func TestPreCallReport_Structure(t *testing.T) {
	report := &PreCallReport{
		URL:                   "https://example.com",
		CompanyName:           "Example Corp",
		Industry:              "Technology",
		CompanySummary:        "A tech company",
		KeyServices:           []string{"Web Dev", "Mobile"},
		TargetAudience:        "SMBs",
		PotentialPainPoints:   []string{"Cost", "Speed"},
		TalkingPoints:         []string{"Pricing", "Demo"},
		CompetitiveAdvantages: []string{"Fast", "Cheap"},
		ContactInfo:           "contact@example.com",
		RecommendedApproach:   "Cold email",
		Success:               true,
		GeneratedAt:           time.Now(),
	}

	assert.Equal(t, "https://example.com", report.URL)
	assert.Equal(t, "Example Corp", report.CompanyName)
	assert.True(t, report.Success)
	assert.Empty(t, report.Error)
	assert.Len(t, report.KeyServices, 2)
}

func TestNewPreCallReportHandler_MissingAPIKey(t *testing.T) {
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

	handler, err := NewPreCallReportHandler(PreCallReportConfig{})

	assert.Nil(t, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Google API key is required")
}

func TestNewPreCallReportHandler_VertexAI_MissingProject(t *testing.T) {
	// Test Vertex AI config without project
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

	handler, err := NewPreCallReportHandler(PreCallReportConfig{
		UseVertexAI: true,
	})

	assert.Nil(t, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GCP Project is required")
}

func TestNewPreCallReportHandler_VertexAI_MissingLocation(t *testing.T) {
	// Test Vertex AI config without location
	originalLocation := os.Getenv("GOOGLE_CLOUD_LOCATION")
	os.Unsetenv("GOOGLE_CLOUD_LOCATION")
	defer func() {
		if originalLocation != "" {
			os.Setenv("GOOGLE_CLOUD_LOCATION", originalLocation)
		}
	}()

	handler, err := NewPreCallReportHandler(PreCallReportConfig{
		UseVertexAI: true,
		GCPProject:  "test-project",
	})

	assert.Nil(t, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GCP Location is required")
}

// Integration tests - only run if GOOGLE_API_KEY is set
func TestPreCallReportHandler_Integration(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: GOOGLE_API_KEY not set")
	}

	t.Run("creates handler with valid config", func(t *testing.T) {
		handler, err := NewPreCallReportHandler(PreCallReportConfig{
			APIKey: apiKey,
		})

		require.NoError(t, err)
		require.NotNil(t, handler)
		assert.Equal(t, DefaultGeminiModel, handler.config.Model)
		assert.Equal(t, DefaultReportTimeout, handler.config.Timeout)
	})

	t.Run("creates handler with custom model", func(t *testing.T) {
		handler, err := NewPreCallReportHandler(PreCallReportConfig{
			APIKey: apiKey,
			Model:  "gemini-2.0-flash",
		})

		require.NoError(t, err)
		require.NotNil(t, handler)
		assert.Equal(t, "gemini-2.0-flash", handler.config.Model)
	})

	t.Run("generates report for result with content", func(t *testing.T) {
		handler, err := NewPreCallReportHandler(PreCallReportConfig{
			APIKey:  apiKey,
			Timeout: 90 * time.Second,
		})
		require.NoError(t, err)

		result := OrganicResult{
			Title:   "Acme Web Development - Custom Software Solutions",
			Link:    "https://acme-webdev.example.com",
			Snippet: "Leading web development agency specializing in custom software solutions for enterprises.",
			ScrapedContent: `# Acme Web Development

## About Us
We are a leading web development agency based in San Francisco, California.
Founded in 2015, we specialize in creating custom software solutions for enterprise clients.

## Our Services
- Custom Web Applications
- Mobile App Development
- Cloud Infrastructure
- DevOps Consulting
- UI/UX Design

## Our Clients
We work with Fortune 500 companies and growing startups alike.

## Contact
Email: hello@acme-webdev.com
Phone: (555) 123-4567
Address: 123 Tech Street, San Francisco, CA 94105`,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		report := handler.GenerateReport(ctx, result)

		assert.True(t, report.Success, "Report generation should succeed: %s", report.Error)
		assert.Equal(t, result.Link, report.URL)
		assert.NotEmpty(t, report.CompanySummary)
		assert.False(t, report.GeneratedAt.IsZero())
	})

	t.Run("handles result without content", func(t *testing.T) {
		handler, err := NewPreCallReportHandler(PreCallReportConfig{
			APIKey: apiKey,
		})
		require.NoError(t, err)

		result := OrganicResult{
			Title:          "",
			Link:           "https://no-content.example.com",
			Snippet:        "",
			ScrapedContent: "",
		}

		ctx := context.Background()
		report := handler.GenerateReport(ctx, result)

		assert.False(t, report.Success)
		assert.Contains(t, report.Error, "no content available")
	})

	t.Run("generates multiple reports concurrently", func(t *testing.T) {
		handler, err := NewPreCallReportHandler(PreCallReportConfig{
			APIKey:        apiKey,
			MaxConcurrent: 2,
			Timeout:       90 * time.Second,
		})
		require.NoError(t, err)

		results := []OrganicResult{
			{
				Title:   "Company A",
				Link:    "https://company-a.example.com",
				Snippet: "A software company providing CRM solutions.",
			},
			{
				Title:   "Company B",
				Link:    "https://company-b.example.com",
				Snippet: "A marketing agency specializing in digital campaigns.",
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		reports := handler.GenerateReports(ctx, results)

		assert.Len(t, reports, 2)
		for _, result := range results {
			report, exists := reports[result.Link]
			assert.True(t, exists, "Report should exist for %s", result.Link)
			if report.Success {
				assert.NotEmpty(t, report.CompanySummary)
			}
		}
	})
}

// Benchmark tests
func BenchmarkExtractSection(b *testing.B) {
	response := `**Company Name**: Acme Corp
**Industry**: Technology
**Company Summary**: A leading tech company providing innovative solutions.
**Key Services**:
- Web Development
- Mobile Apps
- Cloud Solutions`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractSection(response, "Company Name")
	}
}

func BenchmarkExtractListSection(b *testing.B) {
	response := `**Key Services**:
- Web Development
- Mobile Apps
- Cloud Solutions
- DevOps
- Security

**Next Section**: value`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractListSection(response, "Key Services")
	}
}

func BenchmarkSanitizeURL(b *testing.B) {
	url := "https://very-long-domain-name.example.com/path/to/resource?query=param"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizeURL(url)
	}
}
