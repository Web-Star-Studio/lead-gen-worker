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
	SupabaseKey     string // Supabase anon/service key
	WebhookSecret   string // Secret for validating Supabase webhook requests
	// Google AI / Vertex AI configuration
	GoogleAPIKey string // Google API key for Gemini (Google AI Studio backend)
	GeminiModel  string // Optional: Gemini model to use (default: gemini-2.5-pro-preview-06-05)
	UseVertexAI  bool   // Use Vertex AI backend instead of Google AI Studio
	GCPProject   string // Google Cloud project ID (for Vertex AI)
	GCPLocation  string // Google Cloud location (for Vertex AI, e.g., "us-central1")
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
		SupabaseKey:     os.Getenv("SUPABASE_KEY"),
		WebhookSecret:   os.Getenv("WEBHOOK_SECRET"), // For validating Supabase webhooks
		GoogleAPIKey:    os.Getenv("GOOGLE_API_KEY"),
		GeminiModel:     os.Getenv("GEMINI_MODEL"),                        // Optional
		UseVertexAI:     os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "true", // Optional
		GCPProject:      os.Getenv("GOOGLE_CLOUD_PROJECT"),                // For Vertex AI
		GCPLocation:     os.Getenv("GOOGLE_CLOUD_LOCATION"),               // For Vertex AI
	}
}
