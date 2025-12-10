package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvWithFallback(t *testing.T) {
	tests := []struct {
		name          string
		primary       string
		primaryValue  string
		fallback      string
		fallbackValue string
		expected      string
	}{
		{
			name:          "primary exists",
			primary:       "TEST_PRIMARY_VAR",
			primaryValue:  "primary_value",
			fallback:      "TEST_FALLBACK_VAR",
			fallbackValue: "fallback_value",
			expected:      "primary_value",
		},
		{
			name:          "primary empty, fallback exists",
			primary:       "TEST_PRIMARY_EMPTY",
			primaryValue:  "",
			fallback:      "TEST_FALLBACK_EXISTS",
			fallbackValue: "fallback_value",
			expected:      "fallback_value",
		},
		{
			name:          "both empty",
			primary:       "TEST_BOTH_EMPTY_P",
			primaryValue:  "",
			fallback:      "TEST_BOTH_EMPTY_F",
			fallbackValue: "",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			if tt.primaryValue != "" {
				os.Setenv(tt.primary, tt.primaryValue)
				defer os.Unsetenv(tt.primary)
			}
			if tt.fallbackValue != "" {
				os.Setenv(tt.fallback, tt.fallbackValue)
				defer os.Unsetenv(tt.fallback)
			}

			result := getEnvWithFallback(tt.primary, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoad_DefaultPort(t *testing.T) {
	// Clear PORT env var
	os.Unsetenv("PORT")

	config := Load()
	assert.Equal(t, "8080", config.Port)
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("PORT", "3000")
	defer os.Unsetenv("PORT")

	config := Load()
	assert.Equal(t, "3000", config.Port)
}

func TestLoad_AllEnvVars(t *testing.T) {
	// Set all env vars
	envVars := map[string]string{
		"PORT":                      "9000",
		"SERPAPI_KEY":               "test_serp_key",
		"FIRECRAWL_API_KEY":         "test_firecrawl_key",
		"FIRECRAWL_API_URL":         "https://custom.firecrawl.dev",
		"SUPABASE_URL":              "https://test.supabase.co",
		"SUPABASE_SECRET_KEY":       "test_secret_key",
		"WEBHOOK_SECRET":            "webhook_secret_123",
		"GOOGLE_API_KEY":            "google_api_key_test",
		"GEMINI_MODEL":              "gemini-2.5-pro",
		"GOOGLE_GENAI_USE_VERTEXAI": "true",
		"GOOGLE_CLOUD_PROJECT":      "my-project",
		"GOOGLE_CLOUD_LOCATION":     "us-central1",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	config := Load()

	assert.Equal(t, "9000", config.Port)
	assert.Equal(t, "test_serp_key", config.SerpAPIKey)
	assert.Equal(t, "test_firecrawl_key", config.FirecrawlAPIKey)
	assert.Equal(t, "https://custom.firecrawl.dev", config.FirecrawlAPIURL)
	assert.Equal(t, "https://test.supabase.co", config.SupabaseURL)
	assert.Equal(t, "test_secret_key", config.SupabaseKey)
	assert.Equal(t, "webhook_secret_123", config.WebhookSecret)
	assert.Equal(t, "google_api_key_test", config.GoogleAPIKey)
	assert.Equal(t, "gemini-2.5-pro", config.GeminiModel)
	assert.True(t, config.UseVertexAI)
	assert.Equal(t, "my-project", config.GCPProject)
	assert.Equal(t, "us-central1", config.GCPLocation)
}

func TestLoad_SupabaseKeyFallback(t *testing.T) {
	// Test fallback from SUPABASE_KEY to SUPABASE_SECRET_KEY
	os.Unsetenv("SUPABASE_SECRET_KEY")
	os.Setenv("SUPABASE_KEY", "legacy_key")
	defer os.Unsetenv("SUPABASE_KEY")

	config := Load()
	assert.Equal(t, "legacy_key", config.SupabaseKey)
}

func TestLoad_UseVertexAI_False(t *testing.T) {
	os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "false")
	defer os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI")

	config := Load()
	assert.False(t, config.UseVertexAI)
}

func TestLoad_UseVertexAI_NotSet(t *testing.T) {
	os.Unsetenv("GOOGLE_GENAI_USE_VERTEXAI")

	config := Load()
	assert.False(t, config.UseVertexAI)
}
