package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	Port            string
	SerpAPIKey      string
	FirecrawlAPIKey string
	FirecrawlAPIURL string // Optional: custom Firecrawl API URL (leave empty for default)
	SupabaseURL     string // Supabase project URL
	SupabaseKey     string // Supabase secret key (sb_secret_xxx) - replaces legacy service_role key
	WebhookSecret   string // Secret for validating Supabase webhook requests
	// Google AI / Vertex AI configuration
	GoogleAPIKey string // Google API key for Gemini (Google AI Studio backend)
	GeminiModel  string // Optional: Gemini model to use (default: gemini-2.5-pro-preview-06-05)
	UseVertexAI  bool   // Use Vertex AI backend instead of Google AI Studio
	GCPProject   string // Google Cloud project ID (for Vertex AI)
	GCPLocation  string // Google Cloud location (for Vertex AI, e.g., "us-central1")
	// OpenRouter configuration (alternative to Google AI)
	UseOpenRouter     bool   // Use OpenRouter instead of Google AI
	OpenRouterAPIKey  string // OpenRouter API key
	OpenRouterModel   string // OpenRouter model (e.g., "anthropic/claude-3.5-sonnet", "openai/gpt-4o")
	OpenRouterBaseURL string // Optional: custom OpenRouter base URL
}

// getEnvWithFallback returns the value of the primary env var, or fallback if primary is empty
func getEnvWithFallback(primary, fallback string) string {
	if val := os.Getenv(primary); val != "" {
		return val
	}
	return os.Getenv(fallback)
}

// Load reads configuration from environment variables
func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:            port,
		SerpAPIKey:      os.Getenv("SERPAPI_KEY"),
		FirecrawlAPIKey: os.Getenv("FIRECRAWL_API_KEY"),
		FirecrawlAPIURL: os.Getenv("FIRECRAWL_API_URL"), // Optional
		SupabaseURL:     os.Getenv("SUPABASE_URL"),
		SupabaseKey:     getEnvWithFallback("SUPABASE_SECRET_KEY", "SUPABASE_KEY"),
		WebhookSecret:   os.Getenv("WEBHOOK_SECRET"), // For validating Supabase webhooks
		GoogleAPIKey:    os.Getenv("GOOGLE_API_KEY"),
		GeminiModel:     os.Getenv("GEMINI_MODEL"),                        // Optional
		UseVertexAI:     os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "true", // Optional
		GCPProject:      os.Getenv("GOOGLE_CLOUD_PROJECT"),                // For Vertex AI
		GCPLocation:     os.Getenv("GOOGLE_CLOUD_LOCATION"),               // For Vertex AI
		// OpenRouter configuration
		UseOpenRouter:     os.Getenv("USE_OPENROUTER") == "true",
		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:   os.Getenv("OPENROUTER_MODEL"),
		OpenRouterBaseURL: os.Getenv("OPENROUTER_BASE_URL"), // Optional, defaults to https://openrouter.ai/api/v1
	}
}
